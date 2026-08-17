import { UsersIcon } from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { UserAvatarMark } from '@/components/chat/user-avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { formatUserError } from '@/lib/api/client'
import {
  listWorkspaceInvites,
  revokeWorkspaceInvite,
  type ApiInvite,
} from '@/lib/api/invites'
import {
  leaveWorkspace,
  listWorkspaceMembers,
  removeWorkspaceMember,
  transferWorkspaceOwnership,
  updateWorkspaceMemberRole,
  type ApiWorkspaceMember,
} from '@/lib/api/workspaces'

function roleLabel(role: string) {
  switch (role) {
    case 'owner':
      return 'Owner'
    case 'admin':
      return 'Admin'
    default:
      return 'Member'
  }
}

export function WorkspacePeopleSection({
  workspaceId,
  workspaceName,
  serverUrl,
  token,
  canManage,
  currentUserId,
  currentUserRole,
  onLeftWorkspace,
}: {
  workspaceId: string
  workspaceName: string
  serverUrl: string
  token: string
  canManage: boolean
  currentUserId?: string
  currentUserRole: string
  onLeftWorkspace?: () => void
}) {
  const [members, setMembers] = useState<ApiWorkspaceMember[]>([])
  const [invites, setInvites] = useState<ApiInvite[]>([])
  const [loading, setLoading] = useState(true)
  const [busyId, setBusyId] = useState<string | null>(null)

  const reload = async (signal?: AbortSignal) => {
    const [memberResult, inviteResult] = await Promise.all([
      listWorkspaceMembers(serverUrl, token, workspaceId, signal),
      canManage
        ? listWorkspaceInvites(serverUrl, token, workspaceId, signal)
        : Promise.resolve({ invites: [] as ApiInvite[] }),
    ])
    setMembers(memberResult.members)
    setInvites(inviteResult.invites)
  }

  useEffect(() => {
    const controller = new AbortController()
    setLoading(true)
    void reload(controller.signal)
      .catch((err) => {
        if (controller.signal.aborted) return
        const message =
          formatUserError(err, 'Could not load people')
        toast.error(message)
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })
    return () => controller.abort()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [serverUrl, token, workspaceId, canManage])

  const isOwner = currentUserRole === 'owner'

  return (
    <section className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h2 className="flex items-center gap-2 text-sm font-semibold">
          <UsersIcon className="size-4" />
          People in {workspaceName}
        </h2>
        <p className="text-sm text-muted-foreground">
          Manage members, roles, and pending invites. People from another
          workspace on this server appear as external.
        </p>
      </div>

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading people…</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {members.map((member) => {
            const isSelf = member.userId === currentUserId
            const busy = busyId === member.userId
            return (
              <li
                key={member.userId}
                className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <UserAvatarMark
                    userId={member.userId}
                    name={member.displayName}
                    hasAvatar={member.hasAvatar}
                    avatarUpdatedAt={member.avatarUpdatedAt}
                    presence={member.presence}
                    showPresence
                    serverUrl={serverUrl}
                    token={token}
                    className="size-8"
                  />
                  <div className="min-w-0">
                    <div className="flex min-w-0 items-center gap-2">
                      <p className="truncate text-sm font-medium">
                        {member.displayName}
                      </p>
                      {member.isExternal ? (
                        <Badge variant="secondary" className="shrink-0 text-[10px]">
                          {member.homeWorkspaceName
                            ? `From ${member.homeWorkspaceName}`
                            : 'External'}
                        </Badge>
                      ) : null}
                    </div>
                    <p className="truncate text-xs text-muted-foreground">
                      {member.handle ? `@${member.handle}` : roleLabel(member.role)}
                    </p>
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  {canManage && !isSelf && member.role !== 'owner' ? (
                    <select
                      className="h-8 rounded-md border bg-background px-2 text-xs"
                      value={member.role}
                      disabled={busy}
                      aria-label={`Role for ${member.displayName}`}
                      onChange={(event) => {
                        const role = event.target.value
                        setBusyId(member.userId)
                        void updateWorkspaceMemberRole(
                          serverUrl,
                          token,
                          workspaceId,
                          member.userId,
                          role,
                        )
                          .then(() => {
                            setMembers((prev) =>
                              prev.map((item) =>
                                item.userId === member.userId
                                  ? { ...item, role }
                                  : item,
                              ),
                            )
                            toast.success('Role updated')
                          })
                          .catch((err) => {
                            const message =
                              formatUserError(err, 'Could not update role')
                            toast.error(message)
                          })
                          .finally(() => setBusyId(null))
                      }}
                    >
                      <option value="member">Member</option>
                      <option value="admin">Admin</option>
                      {isOwner ? <option value="owner">Owner</option> : null}
                    </select>
                  ) : null}
                  {isOwner && !isSelf ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={busy}
                      onClick={() => {
                        setBusyId(member.userId)
                        void transferWorkspaceOwnership(
                          serverUrl,
                          token,
                          workspaceId,
                          member.userId,
                        )
                          .then(() => reload())
                          .then(() => toast.success('Ownership transferred'))
                          .catch((err) => {
                            const message =
                              formatUserError(err, 'Could not transfer ownership')
                            toast.error(message)
                          })
                          .finally(() => setBusyId(null))
                      }}
                    >
                      Make owner
                    </Button>
                  ) : null}
                  {canManage && !isSelf && member.role !== 'owner' ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      disabled={busy}
                      onClick={() => {
                        setBusyId(member.userId)
                        void removeWorkspaceMember(
                          serverUrl,
                          token,
                          workspaceId,
                          member.userId,
                        )
                          .then(() => {
                            setMembers((prev) =>
                              prev.filter((item) => item.userId !== member.userId),
                            )
                            toast.success('Member removed')
                          })
                          .catch((err) => {
                            const message =
                              formatUserError(err, 'Could not remove member')
                            toast.error(message)
                          })
                          .finally(() => setBusyId(null))
                      }}
                    >
                      Remove
                    </Button>
                  ) : null}
                  {isSelf ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={busy}
                      onClick={() => {
                        setBusyId(member.userId)
                        void leaveWorkspace(serverUrl, token, workspaceId)
                          .then(() => {
                            toast.success('You left the workspace')
                            onLeftWorkspace?.()
                          })
                          .catch((err) => {
                            const message =
                              formatUserError(err, 'Could not leave workspace')
                            toast.error(message)
                          })
                          .finally(() => setBusyId(null))
                      }}
                    >
                      Leave
                    </Button>
                  ) : null}
                </div>
              </li>
            )
          })}
        </ul>
      )}

      {canManage ? (
        <div className="flex flex-col gap-2">
          <h3 className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Pending invites
          </h3>
          {invites.length === 0 ? (
            <p className="text-sm text-muted-foreground">No pending invites.</p>
          ) : (
            <ul className="flex flex-col gap-2">
              {invites.map((invite) => (
                <li
                  key={invite.id}
                  className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{invite.email}</p>
                    <p className="text-xs text-muted-foreground">
                      {roleLabel(invite.role)}
                    </p>
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    disabled={busyId === invite.id}
                    onClick={() => {
                      setBusyId(invite.id)
                      void revokeWorkspaceInvite(
                        serverUrl,
                        token,
                        workspaceId,
                        invite.id,
                      )
                        .then(() => {
                          setInvites((prev) =>
                            prev.filter((item) => item.id !== invite.id),
                          )
                          toast.success('Invite revoked')
                        })
                        .catch((err) => {
                          const message =
                            formatUserError(err, 'Could not revoke invite')
                          toast.error(message)
                        })
                        .finally(() => setBusyId(null))
                    }}
                  >
                    Revoke
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
      ) : null}
    </section>
  )
}

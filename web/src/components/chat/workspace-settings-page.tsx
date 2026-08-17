import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

import { WorkspaceBillingSection } from '@/components/chat/workspace-billing-section'
import { WorkspaceIconMark } from '@/components/chat/workspace-icon'
import { WorkspacePeopleSection } from '@/components/chat/workspace-people-section'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { ApiError } from '@/lib/api/client'
import {
  addDomain,
  listDomains,
  removeDomain,
  updateDomain,
  verifyDomain,
  type ApiWorkspaceDomain,
} from '@/lib/api/domains'
import {
  deleteWorkspace,
  removeWorkspaceIcon,
  renameWorkspace,
  uploadWorkspaceIcon,
  type ApiWorkspace,
} from '@/lib/api/workspaces'
import {
  ArrowLeftIcon,
  CheckIcon,
  CopyIcon,
  GlobeIcon,
  ImageIcon,
  TrashIcon,
} from '@phosphor-icons/react'

function capitaliseName(name: string) {
  const trimmed = name.trim()
  if (!trimmed) return trimmed
  return trimmed.charAt(0).toLocaleUpperCase() + trimmed.slice(1)
}

async function copyText(value: string, label: string) {
  try {
    await navigator.clipboard.writeText(value)
    toast.success(`Copied ${label}`)
  } catch {
    toast.error('Could not copy to clipboard')
  }
}

function DomainRow({
  item,
  canManage,
  busyId,
  onVerify,
  onToggleAutoJoin,
  onRequestRemove,
}: {
  item: ApiWorkspaceDomain
  canManage: boolean
  busyId: string | null
  onVerify: (id: string) => void
  onToggleAutoJoin: (id: string, autoJoin: boolean) => void
  onRequestRemove: (item: ApiWorkspaceDomain) => void
}) {
  const busy = busyId === item.id

  return (
    <div className="flex flex-col gap-3 rounded-lg border p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="truncate text-sm font-medium">{item.domain}</span>
            {item.verified ? (
              <Badge variant="secondary">Verified</Badge>
            ) : (
              <Badge variant="outline">Pending</Badge>
            )}
          </div>
          <p className="text-xs text-muted-foreground">
            {item.verified
              ? 'New signups with this email domain can auto-join when enabled.'
              : 'Add the DNS TXT record below, then verify.'}
          </p>
        </div>
        {canManage && (
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={`Remove ${item.domain}`}
            disabled={busy}
            onClick={() => onRequestRemove(item)}
          >
            <TrashIcon strokeWidth={2} data-icon />
          </Button>
        )}
      </div>

      <div className="flex flex-col gap-2 rounded-md bg-muted/40 p-3 text-xs">
        <div className="flex items-center justify-between gap-2">
          <span className="text-muted-foreground">Host</span>
          <div className="flex min-w-0 items-center gap-1">
            <code className="truncate font-mono">{item.dnsHost}</code>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label="Copy DNS host"
              onClick={() => void copyText(item.dnsHost, 'DNS host')}
            >
              <CopyIcon strokeWidth={2} />
            </Button>
          </div>
        </div>
        <div className="flex items-center justify-between gap-2">
          <span className="text-muted-foreground">Type</span>
          <code className="font-mono">{item.dnsType}</code>
        </div>
        <div className="flex items-center justify-between gap-2">
          <span className="text-muted-foreground">Value</span>
          <div className="flex min-w-0 items-center gap-1">
            <code className="truncate font-mono">{item.dnsValue}</code>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label="Copy DNS value"
              onClick={() => void copyText(item.dnsValue, 'DNS value')}
            >
              <CopyIcon strokeWidth={2} />
            </Button>
          </div>
        </div>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <Switch
            id={`auto-join-${item.id}`}
            checked={item.autoJoin}
            disabled={!canManage || !item.verified || busy}
            onCheckedChange={(checked) => onToggleAutoJoin(item.id, checked)}
          />
          <Label htmlFor={`auto-join-${item.id}`} className="text-sm font-normal">
            Auto-join on signup
          </Label>
        </div>
        {canManage && !item.verified && (
          <Button
            type="button"
            size="sm"
            disabled={busy}
            onClick={() => onVerify(item.id)}
          >
            <CheckIcon strokeWidth={2} data-icon="inline-start" />
            {busy ? 'Checking…' : 'Verify DNS'}
          </Button>
        )}
      </div>
    </div>
  )
}

export function WorkspaceSettingsPage({
  workspace,
  serverUrl,
  token,
  canManage,
  currentUserId,
  onBack,
  onRenamed,
  onIconChanged,
  onDeleted,
  onLeftWorkspace,
}: {
  workspace: ApiWorkspace
  serverUrl: string
  token: string
  canManage: boolean
  currentUserId?: string
  onBack: () => void
  onRenamed: (workspace: ApiWorkspace) => void
  onIconChanged?: (workspace: ApiWorkspace) => void
  onDeleted: (workspaceId: string) => void
  onLeftWorkspace?: () => void
}) {
  const [name, setName] = useState(workspace.name)
  const [confirmName, setConfirmName] = useState('')
  const [saving, setSaving] = useState(false)
  const [iconBusy, setIconBusy] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [domains, setDomains] = useState<ApiWorkspaceDomain[]>([])
  const [domainsLoading, setDomainsLoading] = useState(true)
  const [newDomain, setNewDomain] = useState('')
  const [addingDomain, setAddingDomain] = useState(false)
  const [busyDomainId, setBusyDomainId] = useState<string | null>(null)
  const [domainPendingDelete, setDomainPendingDelete] =
    useState<ApiWorkspaceDomain | null>(null)
  const [confirmDomain, setConfirmDomain] = useState('')
  const [removingDomain, setRemovingDomain] = useState(false)
  const iconInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setName(workspace.name)
    setConfirmName('')
  }, [workspace.id, workspace.name])

  useEffect(() => {
    const controller = new AbortController()
    setDomainsLoading(true)
    void (async () => {
      try {
        const result = await listDomains(
          serverUrl,
          token,
          workspace.id,
          controller.signal,
        )
        if (!controller.signal.aborted) {
          setDomains(result.domains)
        }
      } catch (err) {
        if (controller.signal.aborted) return
        const message =
          err instanceof ApiError ? err.message : 'Could not load domains'
        toast.error(message)
      } finally {
        if (!controller.signal.aborted) {
          setDomainsLoading(false)
        }
      }
    })()
    return () => controller.abort()
  }, [serverUrl, token, workspace.id])

  const nameChanged = capitaliseName(name) !== workspace.name
  const deleteReady = confirmName.trim() === workspace.name
  const domainDeleteReady =
    domainPendingDelete !== null &&
    confirmDomain.trim().toLowerCase() === domainPendingDelete.domain.toLowerCase()

  const upsertDomain = (next: ApiWorkspaceDomain) => {
    setDomains((prev) => {
      const idx = prev.findIndex((item) => item.id === next.id)
      if (idx === -1) return [...prev, next]
      const copy = [...prev]
      copy[idx] = next
      return copy
    })
  }

  return (
    <div className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
      <Dialog
        open={domainPendingDelete !== null}
        onOpenChange={(open) => {
          if (!open && !removingDomain) {
            setDomainPendingDelete(null)
            setConfirmDomain('')
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove domain</DialogTitle>
            <DialogDescription>
              This stops verification and auto-join for{' '}
              <span className="font-medium text-foreground">
                {domainPendingDelete?.domain}
              </span>
              . Type the domain to confirm.
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <Label htmlFor="confirm-delete-domain">Domain</Label>
            <Input
              id="confirm-delete-domain"
              value={confirmDomain}
              onChange={(event) => setConfirmDomain(event.target.value)}
              placeholder={domainPendingDelete?.domain}
              disabled={removingDomain}
              autoComplete="off"
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={removingDomain}
              onClick={() => {
                setDomainPendingDelete(null)
                setConfirmDomain('')
              }}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={!domainDeleteReady || removingDomain}
              onClick={() => {
                if (!domainPendingDelete || !domainDeleteReady || removingDomain) {
                  return
                }
                const target = domainPendingDelete
                setRemovingDomain(true)
                setBusyDomainId(target.id)
                void (async () => {
                  try {
                    await removeDomain(serverUrl, token, workspace.id, target.id)
                    setDomains((prev) => prev.filter((d) => d.id !== target.id))
                    setDomainPendingDelete(null)
                    setConfirmDomain('')
                    toast.success(`Removed ${target.domain}`)
                  } catch (err) {
                    const message =
                      err instanceof ApiError
                        ? err.message
                        : 'Could not remove domain'
                    toast.error(message)
                  } finally {
                    setRemovingDomain(false)
                    setBusyDomainId(null)
                  }
                })()
              }}
            >
              <TrashIcon strokeWidth={2} data-icon="inline-start" />
              {removingDomain ? 'Removing…' : 'Remove domain'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <header className="flex h-14 shrink-0 items-center gap-2 border-b px-3">
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="Back to chat"
          onClick={onBack}
        >
          <ArrowLeftIcon strokeWidth={2} data-icon />
        </Button>
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <h1 className="truncate text-sm font-semibold">Workspace settings</h1>
          <p className="truncate text-xs text-muted-foreground">{workspace.name}</p>
        </div>
        <WorkspaceIconMark
          workspaceId={workspace.id}
          name={workspace.name}
          hasIcon={workspace.hasIcon}
          iconUpdatedAt={workspace.iconUpdatedAt}
          serverUrl={serverUrl}
          token={token}
          className="size-8 overflow-hidden rounded-md object-cover"
          initialsClassName="flex size-8 items-center justify-center rounded-md bg-primary text-xs font-semibold text-primary-foreground"
        />
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-8 p-6">
          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <h2 className="text-sm font-semibold">General</h2>
              <p className="text-sm text-muted-foreground">
                Update how this workspace appears to members.
              </p>
            </div>

            <div className="flex flex-col gap-3">
              <Label>Icon</Label>
              <div className="flex items-center gap-4">
                <WorkspaceIconMark
                  workspaceId={workspace.id}
                  name={workspace.name}
                  hasIcon={workspace.hasIcon}
                  iconUpdatedAt={workspace.iconUpdatedAt}
                  serverUrl={serverUrl}
                  token={token}
                  className="size-14 overflow-hidden rounded-lg object-cover"
                  initialsClassName="flex size-14 items-center justify-center rounded-lg bg-primary text-lg font-semibold text-primary-foreground"
                />
                <div className="flex min-w-0 flex-1 flex-col gap-2">
                  <p className="text-xs text-muted-foreground">
                    PNG, JPEG, WebP, or GIF up to 2 MB.
                  </p>
                  {canManage ? (
                    <div className="flex flex-wrap gap-2">
                      <input
                        ref={iconInputRef}
                        type="file"
                        accept="image/png,image/jpeg,image/webp,image/gif"
                        className="sr-only"
                        onChange={(event) => {
                          const file = event.target.files?.[0]
                          event.target.value = ''
                          if (!file || iconBusy) return
                          setIconBusy(true)
                          void (async () => {
                            try {
                              const result = await uploadWorkspaceIcon(
                                serverUrl,
                                token,
                                workspace.id,
                                file,
                              )
                              onIconChanged?.(result.workspace)
                              onRenamed(result.workspace)
                              toast.success('Workspace icon updated')
                            } catch (err) {
                              const message =
                                err instanceof ApiError
                                  ? err.message
                                  : 'Could not upload icon'
                              toast.error(message)
                            } finally {
                              setIconBusy(false)
                            }
                          })()
                        }}
                      />
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={iconBusy}
                        onClick={() => iconInputRef.current?.click()}
                      >
                        <ImageIcon strokeWidth={2} data-icon="inline-start" />
                        {iconBusy ? 'Uploading…' : 'Upload icon'}
                      </Button>
                      {workspace.hasIcon ? (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          disabled={iconBusy}
                          onClick={() => {
                            setIconBusy(true)
                            void (async () => {
                              try {
                                const result = await removeWorkspaceIcon(
                                  serverUrl,
                                  token,
                                  workspace.id,
                                )
                                onIconChanged?.(result.workspace)
                                onRenamed(result.workspace)
                                toast.success('Workspace icon removed')
                              } catch (err) {
                                const message =
                                  err instanceof ApiError
                                    ? err.message
                                    : 'Could not remove icon'
                                toast.error(message)
                              } finally {
                                setIconBusy(false)
                              }
                            })()
                          }}
                        >
                          <TrashIcon strokeWidth={2} data-icon="inline-start" />
                          Remove
                        </Button>
                      ) : null}
                    </div>
                  ) : (
                    <p className="text-xs text-muted-foreground">
                      Only workspace owners and admins can change the icon.
                    </p>
                  )}
                </div>
              </div>
            </div>

            <form
              className="flex flex-col gap-3"
              onSubmit={(event) => {
                event.preventDefault()
                if (!canManage || !nameChanged || saving) return
                const nextName = capitaliseName(name)
                if (!nextName) return
                setSaving(true)
                void (async () => {
                  try {
                    const result = await renameWorkspace(
                      serverUrl,
                      token,
                      workspace.id,
                      { name: nextName },
                    )
                    onRenamed(result.workspace)
                    toast.success('Workspace renamed')
                  } catch (err) {
                    const message =
                      err instanceof ApiError
                        ? err.message
                        : 'Could not rename workspace'
                    toast.error(message)
                  } finally {
                    setSaving(false)
                  }
                })()
              }}
            >
              <div className="flex flex-col gap-2">
                <Label htmlFor="settings-workspace-name">Name</Label>
                <Input
                  id="settings-workspace-name"
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                  maxLength={80}
                  disabled={!canManage || saving}
                />
                <p className="text-xs text-muted-foreground">
                  Slug: {workspace.slug}
                  {canManage ? ' (updates when you save a new name)' : ''}
                </p>
              </div>
              {canManage ? (
                <div className="flex justify-end">
                  <Button type="submit" disabled={!nameChanged || saving || !name.trim()}>
                    {saving ? 'Saving…' : 'Save name'}
                  </Button>
                </div>
              ) : (
                <p className="text-xs text-muted-foreground">
                  Only workspace owners and admins can rename this workspace.
                </p>
              )}
            </form>
          </section>

          <WorkspaceBillingSection
            workspaceId={workspace.id}
            serverUrl={serverUrl}
            token={token}
            canManage={canManage}
          />

          <Separator />

          <WorkspacePeopleSection
            workspaceId={workspace.id}
            workspaceName={workspace.name}
            serverUrl={serverUrl}
            token={token}
            canManage={canManage}
            currentUserId={currentUserId}
            currentUserRole={workspace.role}
            onLeftWorkspace={onLeftWorkspace}
          />

          <Separator />

          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <h2 className="flex items-center gap-2 text-sm font-semibold">
                <GlobeIcon strokeWidth={2} className="size-4" />
                Verified domains
              </h2>
              <p className="text-sm text-muted-foreground">
                Prove ownership with a DNS TXT record. When auto-join is on, new accounts
                registering with that email domain join this workspace as members.
              </p>
            </div>

            {canManage && (
              <form
                className="flex flex-col gap-3 sm:flex-row sm:items-end"
                onSubmit={(event) => {
                  event.preventDefault()
                  const trimmed = newDomain.trim().toLowerCase()
                  if (!trimmed || addingDomain) return
                  setAddingDomain(true)
                  void (async () => {
                    try {
                      const result = await addDomain(serverUrl, token, workspace.id, {
                        domain: trimmed,
                      })
                      upsertDomain(result.domain)
                      setNewDomain('')
                      toast.success(`Added ${result.domain.domain}`)
                    } catch (err) {
                      const message =
                        err instanceof ApiError
                          ? err.message
                          : 'Could not add domain'
                      toast.error(message)
                    } finally {
                      setAddingDomain(false)
                    }
                  })()
                }}
              >
                <div className="flex min-w-0 flex-1 flex-col gap-2">
                  <Label htmlFor="workspace-domain">Domain</Label>
                  <Input
                    id="workspace-domain"
                    value={newDomain}
                    onChange={(event) => setNewDomain(event.target.value)}
                    placeholder="example.com"
                    disabled={addingDomain}
                    autoComplete="off"
                  />
                </div>
                <Button type="submit" disabled={!newDomain.trim() || addingDomain}>
                  {addingDomain ? 'Adding…' : 'Add domain'}
                </Button>
              </form>
            )}

            {domainsLoading ? (
              <p className="text-sm text-muted-foreground">Loading domains…</p>
            ) : domains.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No domains yet.
                {canManage ? ' Add a company domain to get started.' : ''}
              </p>
            ) : (
              <div className="flex flex-col gap-3">
                {domains.map((item) => (
                  <DomainRow
                    key={item.id}
                    item={item}
                    canManage={canManage}
                    busyId={busyDomainId}
                    onVerify={(id) => {
                      setBusyDomainId(id)
                      void (async () => {
                        try {
                          const result = await verifyDomain(
                            serverUrl,
                            token,
                            workspace.id,
                            id,
                          )
                          upsertDomain(result.domain)
                          toast.success(`Verified ${result.domain.domain}`)
                        } catch (err) {
                          const message =
                            err instanceof ApiError
                              ? err.message
                              : 'Could not verify domain'
                          toast.error(message)
                        } finally {
                          setBusyDomainId(null)
                        }
                      })()
                    }}
                    onToggleAutoJoin={(id, autoJoin) => {
                      setBusyDomainId(id)
                      void (async () => {
                        try {
                          const result = await updateDomain(
                            serverUrl,
                            token,
                            workspace.id,
                            id,
                            { autoJoin },
                          )
                          upsertDomain(result.domain)
                          toast.success(
                            autoJoin
                              ? 'Auto-join enabled'
                              : 'Auto-join disabled',
                          )
                        } catch (err) {
                          const message =
                            err instanceof ApiError
                              ? err.message
                              : 'Could not update auto-join'
                          toast.error(message)
                        } finally {
                          setBusyDomainId(null)
                        }
                      })()
                    }}
                    onRequestRemove={(item) => {
                      setDomainPendingDelete(item)
                      setConfirmDomain('')
                    }}
                  />
                ))}
              </div>
            )}

            {!canManage && (
              <p className="text-xs text-muted-foreground">
                Only workspace owners and admins can manage domains.
              </p>
            )}
          </section>

          <Separator />

          <section className="flex flex-col gap-4">
            <div className="flex flex-col gap-1">
              <h2 className="text-sm font-semibold text-destructive">Danger zone</h2>
              <p className="text-sm text-muted-foreground">
                Deleting a workspace removes its channels and memberships. This cannot be undone.
              </p>
            </div>

            {canManage ? (
              <div className="flex flex-col gap-3 rounded-lg border border-destructive/30 bg-destructive/5 p-4">
                <div className="flex flex-col gap-2">
                  <Label htmlFor="confirm-delete-workspace">
                    Type {workspace.name} to confirm
                  </Label>
                  <Input
                    id="confirm-delete-workspace"
                    value={confirmName}
                    onChange={(event) => setConfirmName(event.target.value)}
                    placeholder={workspace.name}
                    disabled={deleting}
                    autoComplete="off"
                  />
                </div>
                <div className="flex justify-end">
                  <Button
                    type="button"
                    variant="destructive"
                    disabled={!deleteReady || deleting}
                    onClick={() => {
                      if (!deleteReady || deleting) return
                      setDeleting(true)
                      void (async () => {
                        try {
                          await deleteWorkspace(serverUrl, token, workspace.id)
                          toast.success(`Deleted ${workspace.name}`)
                          onDeleted(workspace.id)
                        } catch (err) {
                          const message =
                            err instanceof ApiError
                              ? err.message
                              : 'Could not delete workspace'
                          toast.error(message)
                          setDeleting(false)
                        }
                      })()
                    }}
                  >
                    <TrashIcon strokeWidth={2} data-icon="inline-start" />
                    {deleting ? 'Deleting…' : 'Delete workspace'}
                  </Button>
                </div>
              </div>
            ) : (
              <p className="text-xs text-muted-foreground">
                Only workspace owners and admins can delete this workspace.
              </p>
            )}
          </section>
        </div>
      </ScrollArea>
    </div>
  )
}

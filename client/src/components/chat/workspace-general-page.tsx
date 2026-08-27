import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'

import { WorkspaceIconMark } from '@/components/chat/workspace-icon'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { formatUserError } from '@/lib/api/client'
import {
  deleteWorkspace,
  removeWorkspaceIcon,
  renameWorkspace,
  uploadWorkspaceIcon,
  type ApiWorkspace,
} from '@/lib/api/workspaces'
import { useLocale } from '@/lib/i18n'
import { ImageIcon, TrashIcon } from '@phosphor-icons/react'

function capitaliseName(name: string) {
  const trimmed = name.trim()
  if (!trimmed) return trimmed
  return trimmed.charAt(0).toLocaleUpperCase() + trimmed.slice(1)
}

export function WorkspaceGeneralPage({
  workspace,
  serverUrl,
  token,
  canManage,
  onRenamed,
  onIconChanged,
  onDeleted,
}: {
  workspace: ApiWorkspace
  serverUrl: string
  token: string
  canManage: boolean
  onRenamed: (workspace: ApiWorkspace) => void
  onIconChanged?: (workspace: ApiWorkspace) => void
  onDeleted: (workspaceId: string) => void
}) {
  const { t } = useLocale()
  const [name, setName] = useState(workspace.name)
  const [confirmName, setConfirmName] = useState('')
  const [saving, setSaving] = useState(false)
  const [iconBusy, setIconBusy] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const iconInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setName(workspace.name)
    setConfirmName('')
  }, [workspace.id, workspace.name])

  const nameChanged = capitaliseName(name) !== workspace.name
  const deleteReady = confirmName.trim() === workspace.name

  return (
    <>
      <section className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <h2 className="text-sm font-semibold">{t('workspace.general')}</h2>
          <p className="text-sm text-muted-foreground">
            {t('workspace.generalHint')}
          </p>
        </div>

        <div className="flex flex-col gap-3">
          <Label>{t('workspace.icon')}</Label>
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
                {t('workspace.iconHint')}
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
                          toast.success(t('workspace.iconUpdated'))
                        } catch (err) {
                          const message =
                            formatUserError(err, t('workspace.iconUploadError'))
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
                    {iconBusy ? t('workspace.uploading') : t('workspace.uploadIcon')}
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
                            toast.success(t('workspace.iconRemoved'))
                          } catch (err) {
                            const message =
                              formatUserError(err, t('workspace.iconRemoveError'))
                            toast.error(message)
                          } finally {
                            setIconBusy(false)
                          }
                        })()
                      }}
                    >
                      <TrashIcon strokeWidth={2} data-icon="inline-start" />
                      {t('common.remove')}
                    </Button>
                  ) : null}
                </div>
              ) : (
                <p className="text-xs text-muted-foreground">
                  {t('workspace.iconPermission')}
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
                toast.success(t('workspace.renamed'))
              } catch (err) {
                const message =
                  formatUserError(err, t('workspace.renameError'))
                toast.error(message)
              } finally {
                setSaving(false)
              }
            })()
          }}
        >
          <div className="flex flex-col gap-2">
            <Label htmlFor="settings-workspace-name">{t('common.name')}</Label>
            <Input
              id="settings-workspace-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              maxLength={80}
              disabled={!canManage || saving}
            />
            <p className="text-xs text-muted-foreground">
              {t('workspace.slug', { slug: workspace.slug })}
              {canManage ? t('workspace.slugUpdates') : ''}
            </p>
          </div>
          {canManage ? (
            <div className="flex justify-end">
              <Button type="submit" disabled={!nameChanged || saving || !name.trim()}>
                {saving ? t('common.saving') : t('workspace.saveName')}
              </Button>
            </div>
          ) : (
            <p className="text-xs text-muted-foreground">
              {t('workspace.renamePermission')}
            </p>
          )}
        </form>
      </section>

      <Separator />

      <section className="flex flex-col gap-4">
        <div className="flex flex-col gap-1">
          <h2 className="text-sm font-semibold text-destructive">
            {t('workspace.danger.title')}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t('workspace.danger.description')}
          </p>
        </div>

        {canManage ? (
          <div className="flex flex-col gap-3 rounded-lg border border-destructive/30 bg-destructive/5 p-4">
            <div className="flex flex-col gap-2">
              <Label htmlFor="confirm-delete-workspace">
                {t('workspace.danger.confirmPlaceholder', {
                  name: workspace.name,
                })}
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
                      toast.success(
                        t('workspace.danger.deleted', { name: workspace.name }),
                      )
                      onDeleted(workspace.id)
                    } catch (err) {
                      const message =
                        formatUserError(err, t('workspace.danger.deleteError'))
                      toast.error(message)
                      setDeleting(false)
                    }
                  })()
                }}
              >
                <TrashIcon strokeWidth={2} data-icon="inline-start" />
                {deleting
                  ? t('common.deleting')
                  : t('workspace.danger.delete')}
              </Button>
            </div>
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">
            {t('workspace.danger.permission')}
          </p>
        )}
      </section>
    </>
  )
}

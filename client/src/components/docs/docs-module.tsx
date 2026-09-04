import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { DocsIconPicker } from '@/components/docs/docs-icon-picker'
import { DocsPage } from '@/components/docs/docs-page'
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
import { formatUserError } from '@/lib/api/client'
import {
  createDocument,
  getDocument,
  type ApiDocument,
} from '@/lib/api/documents'
import { useLocale } from '@/lib/i18n'

export function DocsModule({
  serverUrl,
  token,
  documentId,
  canDelete,
  onUpdated,
  onDeleted,
}: {
  serverUrl: string
  token: string
  documentId: string
  canDelete: boolean
  onUpdated: (document: ApiDocument) => void
  onDeleted: (documentId: string) => void
}) {
  const { t } = useLocale()
  const [page, setPage] = useState<ApiDocument | null>(null)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!documentId) {
      setPage(null)
      return
    }
    const controller = new AbortController()
    setLoading(true)
    void (async () => {
      try {
        const result = await getDocument(
          serverUrl,
          token,
          documentId,
          controller.signal,
        )
        if (controller.signal.aborted) return
        setPage(result.document)
      } catch (err) {
        if (controller.signal.aborted) return
        toast.error(formatUserError(err, t('docs.loadError')))
        setPage(null)
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    })()
    return () => controller.abort()
  }, [documentId, serverUrl, token, t])

  if (!documentId) {
    return (
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
        <p className="text-sm text-muted-foreground">{t('docs.emptyHint')}</p>
      </div>
    )
  }

  if (loading && !page) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center p-8">
        <p className="text-sm text-muted-foreground">{t('docs.loading')}</p>
      </div>
    )
  }

  if (!page) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center p-8">
        <p className="text-sm text-muted-foreground">{t('docs.missing')}</p>
      </div>
    )
  }

  return (
    <DocsPage
      document={page}
      serverUrl={serverUrl}
      token={token}
      canDelete={canDelete}
      onUpdated={(next) => {
        setPage(next)
        onUpdated(next)
      }}
      onDeleted={onDeleted}
    />
  )
}

export function CreateDocumentDialog({
  open,
  onOpenChange,
  serverUrl,
  token,
  workspaceId,
  parentId,
  parentTitle,
  onCreated,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  serverUrl: string
  token: string
  workspaceId: string
  parentId?: string
  parentTitle?: string
  onCreated: (document: ApiDocument) => void
}) {
  const { t } = useLocale()
  const [title, setTitle] = useState('')
  const [icon, setIcon] = useState('')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) return
    setTitle('')
    setIcon('')
    setSubmitting(false)
  }, [open])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {parentId ? t('docs.newChildPage') : t('docs.newPage')}
          </DialogTitle>
          <DialogDescription>
            {parentId && parentTitle
              ? t('docs.createChildHint', { title: parentTitle })
              : t('docs.createHint')}
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            const trimmed = title.trim()
            if (!trimmed || submitting) return
            setSubmitting(true)
            void (async () => {
              try {
                const result = await createDocument(serverUrl, token, workspaceId, {
                  title: trimmed,
                  icon: icon || undefined,
                  parentId,
                })
                onCreated(result.document)
                onOpenChange(false)
                toast.success(t('docs.created', { title: result.document.title }))
              } catch (err) {
                toast.error(formatUserError(err, t('docs.createError')))
                setSubmitting(false)
              }
            })()
          }}
        >
          <div className="flex items-end gap-3">
            <DocsIconPicker value={icon} onChange={setIcon} disabled={submitting} />
            <div className="flex min-w-0 flex-1 flex-col gap-2">
              <Label htmlFor="doc-create-title">{t('docs.title')}</Label>
              <Input
                id="doc-create-title"
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                autoFocus
                maxLength={200}
                disabled={submitting}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              {t('common.cancel')}
            </Button>
            <Button type="submit" disabled={submitting || !title.trim()}>
              {submitting ? t('common.creating') : t('common.create')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

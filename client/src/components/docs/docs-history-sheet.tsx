import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { DocsMarkdown } from '@/components/docs/docs-markdown'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { formatUserError } from '@/lib/api/client'
import {
  getDocumentRevision,
  listDocumentRevisions,
  restoreDocumentRevision,
  type ApiDocument,
  type ApiDocumentRevision,
  type ApiDocumentRevisionSummary,
} from '@/lib/api/documents'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

function formatRevisionTime(iso: string) {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString()
}

export function DocsHistorySheet({
  open,
  onOpenChange,
  documentId,
  serverUrl,
  token,
  onRestored,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  documentId: string
  serverUrl: string
  token: string
  onRestored: (document: ApiDocument) => void
}) {
  const { t } = useLocale()
  const [revisions, setRevisions] = useState<ApiDocumentRevisionSummary[]>([])
  const [selectedId, setSelectedId] = useState('')
  const [preview, setPreview] = useState<ApiDocumentRevision | null>(null)
  const [loading, setLoading] = useState(false)
  const [restoring, setRestoring] = useState(false)

  useEffect(() => {
    if (!open) {
      setRevisions([])
      setSelectedId('')
      setPreview(null)
      return
    }
    const controller = new AbortController()
    setLoading(true)
    void (async () => {
      try {
        const result = await listDocumentRevisions(
          serverUrl,
          token,
          documentId,
          controller.signal,
        )
        if (controller.signal.aborted) return
        setRevisions(result.revisions)
        setSelectedId(result.revisions[0]?.id ?? '')
      } catch (err) {
        if (controller.signal.aborted) return
        toast.error(formatUserError(err, t('docs.historyLoadError')))
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    })()
    return () => controller.abort()
  }, [open, documentId, serverUrl, token, t])

  useEffect(() => {
    if (!open || !selectedId) {
      setPreview(null)
      return
    }
    const controller = new AbortController()
    void (async () => {
      try {
        const result = await getDocumentRevision(
          serverUrl,
          token,
          documentId,
          selectedId,
          controller.signal,
        )
        if (controller.signal.aborted) return
        setPreview(result.revision)
      } catch (err) {
        if (controller.signal.aborted) return
        toast.error(formatUserError(err, t('docs.historyLoadError')))
        setPreview(null)
      }
    })()
    return () => controller.abort()
  }, [open, selectedId, documentId, serverUrl, token, t])

  const restore = async () => {
    if (!selectedId || restoring) return
    if (!window.confirm(t('docs.restoreConfirm'))) return
    setRestoring(true)
    try {
      const result = await restoreDocumentRevision(
        serverUrl,
        token,
        documentId,
        selectedId,
      )
      onRestored(result.document)
      onOpenChange(false)
      toast.success(t('docs.restored'))
    } catch (err) {
      toast.error(formatUserError(err, t('docs.restoreError')))
      setRestoring(false)
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-lg" side="right">
        <SheetHeader>
          <SheetTitle>{t('docs.history')}</SheetTitle>
          <SheetDescription>{t('docs.historyHint')}</SheetDescription>
        </SheetHeader>
        <div className="flex min-h-0 flex-1 flex-col gap-3 px-4 pb-4">
          {loading ? (
            <p className="text-sm text-muted-foreground">{t('docs.loading')}</p>
          ) : revisions.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t('docs.historyEmpty')}</p>
          ) : (
            <>
              <ScrollArea className="h-40 shrink-0">
                <ul className="flex flex-col gap-1 pr-2">
                  {revisions.map((rev) => (
                    <li key={rev.id}>
                      <button
                        type="button"
                        className={cn(
                          'flex w-full flex-col items-start rounded-md px-2 py-1.5 text-left text-sm',
                          selectedId === rev.id
                            ? 'bg-muted font-medium'
                            : 'hover:bg-muted/60',
                        )}
                        onClick={() => setSelectedId(rev.id)}
                      >
                        <span className="truncate">
                          {t('docs.historyVersion', { version: rev.version })}
                          {rev.title ? ` · ${rev.title}` : ''}
                        </span>
                        <span className="text-xs font-normal text-muted-foreground">
                          {rev.createdByName || t('docs.someone')} ·{' '}
                          {formatRevisionTime(rev.createdAt)}
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              </ScrollArea>
              <ScrollArea className="min-h-0 flex-1 rounded-md border">
                <div className="p-3">
                  {preview?.body?.trim() ? (
                    <DocsMarkdown body={preview.body} />
                  ) : (
                    <p className="text-sm text-muted-foreground">
                      {t('docs.emptyBody')}
                    </p>
                  )}
                </div>
              </ScrollArea>
              {revisions.length > 1 ? (
                <Button
                  type="button"
                  disabled={restoring || !selectedId}
                  onClick={() => void restore()}
                >
                  {restoring ? t('docs.restoring') : t('docs.restore')}
                </Button>
              ) : null}
            </>
          )}
        </div>
      </SheetContent>
    </Sheet>
  )
}

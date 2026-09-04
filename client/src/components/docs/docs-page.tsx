import {
  ClockCounterClockwiseIcon,
  DotsThreeVerticalIcon,
  PencilSimpleIcon,
  TrashIcon,
} from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { DocsEditor } from '@/components/docs/docs-editor'
import { DocsHistorySheet } from '@/components/docs/docs-history-sheet'
import { DocsIconPicker } from '@/components/docs/docs-icon-picker'
import { DocsMarkdown } from '@/components/docs/docs-markdown'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { formatUserError } from '@/lib/api/client'
import {
  deleteDocument,
  updateDocument,
  type ApiDocument,
} from '@/lib/api/documents'
import { useLocale } from '@/lib/i18n'

export function DocsPage({
  document,
  serverUrl,
  token,
  canDelete,
  onUpdated,
  onDeleted,
}: {
  document: ApiDocument
  serverUrl: string
  token: string
  canDelete: boolean
  onUpdated: (document: ApiDocument) => void
  onDeleted: (documentId: string) => void
}) {
  const { t } = useLocale()
  const pageBody = document.body ?? ''
  const pageIcon = document.icon ?? ''
  const [editing, setEditing] = useState(false)
  const [historyOpen, setHistoryOpen] = useState(false)
  const [title, setTitle] = useState(document.title)
  const [slug, setSlug] = useState(document.slug)
  const [icon, setIcon] = useState(pageIcon)
  const [body, setBody] = useState(pageBody)
  const [saving, setSaving] = useState(false)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    setEditing(false)
    setTitle(document.title)
    setSlug(document.slug)
    setIcon(document.icon ?? '')
    setBody(document.body ?? '')
  }, [document.id, document.title, document.slug, document.icon, document.body])

  const save = async () => {
    const nextTitle = title.trim()
    if (!nextTitle || saving) return
    setSaving(true)
    try {
      const result = await updateDocument(serverUrl, token, document.id, {
        title: nextTitle,
        slug: slug.trim() || undefined,
        icon,
        body,
      })
      onUpdated(result.document)
      setEditing(false)
      toast.success(t('docs.saved'))
    } catch (err) {
      toast.error(formatUserError(err, t('docs.saveError')))
    } finally {
      setSaving(false)
    }
  }

  const remove = async () => {
    if (!canDelete || deleting) return
    if (!window.confirm(t('docs.deleteConfirm', { title: document.title }))) {
      return
    }
    setDeleting(true)
    try {
      await deleteDocument(serverUrl, token, document.id)
      onDeleted(document.id)
      toast.success(t('docs.deleted'))
    } catch (err) {
      toast.error(formatUserError(err, t('docs.deleteError')))
      setDeleting(false)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <header className="flex h-14 shrink-0 items-center gap-2 border-b bg-background px-4">
        {editing ? (
          <DocsIconPicker value={icon} onChange={setIcon} disabled={saving} />
        ) : pageIcon ? (
          <span className="flex size-9 shrink-0 items-center justify-center text-lg leading-none" aria-hidden>
            {pageIcon}
          </span>
        ) : null}
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-sm font-semibold">{document.title}</h1>
          <p className="truncate text-xs text-muted-foreground">
            [[{document.slug}]]
          </p>
        </div>
        {editing ? (
          <>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={saving}
              onClick={() => {
                setTitle(document.title)
                setSlug(document.slug)
                setIcon(document.icon ?? '')
                setBody(document.body ?? '')
                setEditing(false)
              }}
            >
              {t('common.cancel')}
            </Button>
            <Button type="button" size="sm" disabled={saving} onClick={() => void save()}>
              {saving ? t('common.saving') : t('common.save')}
            </Button>
          </>
        ) : (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon-sm"
                aria-label={t('docs.actions')}
              >
                <DotsThreeVerticalIcon strokeWidth={2} />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="min-w-44">
              <DropdownMenuItem onSelect={() => setEditing(true)}>
                <PencilSimpleIcon strokeWidth={2} />
                {t('docs.edit')}
              </DropdownMenuItem>
              <DropdownMenuItem onSelect={() => setHistoryOpen(true)}>
                <ClockCounterClockwiseIcon strokeWidth={2} />
                {t('docs.history')}
              </DropdownMenuItem>
              {canDelete ? (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    variant="destructive"
                    disabled={deleting}
                    onSelect={() => void remove()}
                  >
                    <TrashIcon strokeWidth={2} />
                    {deleting ? t('common.deleting') : t('common.delete')}
                  </DropdownMenuItem>
                </>
              ) : null}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </header>
      <ScrollArea className="min-h-0 flex-1">
        <div className="mx-auto flex w-full max-w-3xl flex-col gap-4 p-6">
          {editing ? (
            <>
              <div className="grid gap-2">
                <Label htmlFor="doc-title">{t('docs.title')}</Label>
                <Input
                  id="doc-title"
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  maxLength={200}
                  disabled={saving}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="doc-slug">{t('docs.slug')}</Label>
                <Input
                  id="doc-slug"
                  value={slug}
                  onChange={(event) => setSlug(event.target.value)}
                  maxLength={80}
                  disabled={saving}
                />
              </div>
              <DocsEditor value={body} onChange={setBody} disabled={saving} />
            </>
          ) : pageBody.trim() ? (
            <DocsMarkdown body={pageBody} />
          ) : (
            <p className="text-sm text-muted-foreground">{t('docs.emptyBody')}</p>
          )}
        </div>
      </ScrollArea>
      <DocsHistorySheet
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        documentId={document.id}
        serverUrl={serverUrl}
        token={token}
        onRestored={onUpdated}
      />
    </div>
  )
}

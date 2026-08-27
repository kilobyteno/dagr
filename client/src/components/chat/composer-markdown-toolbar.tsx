import {
  CodeBlockIcon,
  CodeIcon,
  LinkSimpleIcon,
  ListBulletsIcon,
  QuotesIcon,
  TextBIcon,
  TextItalicIcon,
  TextStrikethroughIcon,
} from '@phosphor-icons/react'

import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

export type ComposerSelectionEdit = {
  next: string
  selectionStart: number
  selectionEnd: number
}

function wrapOrInsert(
  value: string,
  start: number,
  end: number,
  before: string,
  after: string,
  placeholder: string,
): ComposerSelectionEdit {
  const selected = value.slice(start, end)
  const inner = selected || placeholder
  const next = `${value.slice(0, start)}${before}${inner}${after}${value.slice(end)}`
  const selectionStart = start + before.length
  return {
    next,
    selectionStart,
    selectionEnd: selectionStart + inner.length,
  }
}

function prefixLines(
  value: string,
  start: number,
  end: number,
  prefix: string,
  placeholder: string,
): ComposerSelectionEdit {
  const selected = value.slice(start, end)
  const block = selected || placeholder
  const prefixed = block
    .split('\n')
    .map((line) => `${prefix}${line}`)
    .join('\n')
  const next = `${value.slice(0, start)}${prefixed}${value.slice(end)}`
  if (selected) {
    return {
      next,
      selectionStart: start,
      selectionEnd: start + prefixed.length,
    }
  }
  const selectionStart = start + prefix.length
  return {
    next,
    selectionStart,
    selectionEnd: selectionStart + placeholder.length,
  }
}

type MarkdownAction = {
  label: string
  title: string
  icon: typeof TextBIcon
  apply: (value: string, start: number, end: number) => ComposerSelectionEdit
}

function getActions(t: ReturnType<typeof useLocale>['t']): MarkdownAction[] {
  return [
    {
      label: t('chat.markdown.bold'),
      title: t('chat.markdown.bold'),
      icon: TextBIcon,
      apply: (value, start, end) =>
        wrapOrInsert(value, start, end, '**', '**', t('chat.markdown.boldPlaceholder')),
    },
    {
      label: t('chat.markdown.italic'),
      title: t('chat.markdown.italic'),
      icon: TextItalicIcon,
      apply: (value, start, end) =>
        wrapOrInsert(
          value,
          start,
          end,
          '*',
          '*',
          t('chat.markdown.italicPlaceholder'),
        ),
    },
    {
      label: t('chat.markdown.strikethrough'),
      title: t('chat.markdown.strikethrough'),
      icon: TextStrikethroughIcon,
      apply: (value, start, end) =>
        wrapOrInsert(
          value,
          start,
          end,
          '~~',
          '~~',
          t('chat.markdown.strikePlaceholder'),
        ),
    },
    {
      label: t('chat.markdown.inlineCode'),
      title: t('chat.markdown.inlineCode'),
      icon: CodeIcon,
      apply: (value, start, end) =>
        wrapOrInsert(value, start, end, '`', '`', t('chat.markdown.codePlaceholder')),
    },
    {
      label: t('chat.markdown.codeBlock'),
      title: t('chat.markdown.codeBlock'),
      icon: CodeBlockIcon,
      apply: (value, start, end) =>
        wrapOrInsert(
          value,
          start,
          end,
          '```\n',
          '\n```',
          t('chat.markdown.codePlaceholder'),
        ),
    },
    {
      label: t('chat.markdown.link'),
      title: t('chat.markdown.link'),
      icon: LinkSimpleIcon,
      apply: (value, start, end) => {
        const selected = value.slice(start, end)
        const url = t('chat.markdown.url')
        if (selected) {
          const next = `${value.slice(0, start)}[${selected}](${url})${value.slice(end)}`
          const selectionStart = start + selected.length + 3
          return {
            next,
            selectionStart,
            selectionEnd: selectionStart + url.length,
          }
        }
        const linkText = t('chat.markdown.linkText')
        const next = `${value.slice(0, start)}[${linkText}](${url})${value.slice(end)}`
        return {
          next,
          selectionStart: start + 1,
          selectionEnd: start + 1 + linkText.length,
        }
      },
    },
    {
      label: t('chat.markdown.quote'),
      title: t('chat.markdown.quote'),
      icon: QuotesIcon,
      apply: (value, start, end) =>
        prefixLines(value, start, end, '> ', t('chat.markdown.quotePlaceholder')),
    },
    {
      label: t('chat.markdown.bulletedList'),
      title: t('chat.markdown.bulletedList'),
      icon: ListBulletsIcon,
      apply: (value, start, end) =>
        prefixLines(value, start, end, '- ', t('chat.markdown.listItem')),
    },
  ]
}

export function ComposerMarkdownToolbar({
  disabled,
  value,
  getSelection,
  onApply,
  className,
}: {
  disabled?: boolean
  value: string
  getSelection: () => { start: number; end: number }
  onApply: (edit: ComposerSelectionEdit) => void
  className?: string
}) {
  const { t } = useLocale()
  const actions = getActions(t)

  return (
    <div
      role="toolbar"
      aria-label={t('chat.markdown.toolbar')}
      className={cn('flex flex-wrap items-center gap-0.5', className)}
    >
      {actions.map((action, index) => {
        const Icon = action.icon
        const showSeparator = index === 3 || index === 6
        return (
          <div key={action.label} className="flex items-center gap-0.5">
            {showSeparator ? (
              <Separator orientation="vertical" className="mx-1 h-4" />
            ) : null}
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              disabled={disabled}
              title={action.title}
              aria-label={action.label}
              onMouseDown={(event) => {
                // Keep composer selection when clicking the toolbar.
                event.preventDefault()
              }}
              onClick={() => {
                const { start, end } = getSelection()
                onApply(action.apply(value, start, end))
              }}
            >
              <Icon strokeWidth={2} />
            </Button>
          </div>
        )
      })}
    </div>
  )
}

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

const ACTIONS: MarkdownAction[] = [
  {
    label: 'Bold',
    title: 'Bold',
    icon: TextBIcon,
    apply: (value, start, end) =>
      wrapOrInsert(value, start, end, '**', '**', 'bold text'),
  },
  {
    label: 'Italic',
    title: 'Italic',
    icon: TextItalicIcon,
    apply: (value, start, end) =>
      wrapOrInsert(value, start, end, '*', '*', 'italic text'),
  },
  {
    label: 'Strikethrough',
    title: 'Strikethrough',
    icon: TextStrikethroughIcon,
    apply: (value, start, end) =>
      wrapOrInsert(value, start, end, '~~', '~~', 'struck text'),
  },
  {
    label: 'Inline code',
    title: 'Inline code',
    icon: CodeIcon,
    apply: (value, start, end) =>
      wrapOrInsert(value, start, end, '`', '`', 'code'),
  },
  {
    label: 'Code block',
    title: 'Code block',
    icon: CodeBlockIcon,
    apply: (value, start, end) =>
      wrapOrInsert(value, start, end, '```\n', '\n```', 'code'),
  },
  {
    label: 'Link',
    title: 'Link',
    icon: LinkSimpleIcon,
    apply: (value, start, end) => {
      const selected = value.slice(start, end)
      if (selected) {
        const next = `${value.slice(0, start)}[${selected}](url)${value.slice(end)}`
        const selectionStart = start + selected.length + 3
        return {
          next,
          selectionStart,
          selectionEnd: selectionStart + 3,
        }
      }
      const next = `${value.slice(0, start)}[link text](url)${value.slice(end)}`
      return {
        next,
        selectionStart: start + 1,
        selectionEnd: start + 10,
      }
    },
  },
  {
    label: 'Quote',
    title: 'Quote',
    icon: QuotesIcon,
    apply: (value, start, end) =>
      prefixLines(value, start, end, '> ', 'quote'),
  },
  {
    label: 'Bulleted list',
    title: 'Bulleted list',
    icon: ListBulletsIcon,
    apply: (value, start, end) =>
      prefixLines(value, start, end, '- ', 'list item'),
  },
]

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
  return (
    <div
      role="toolbar"
      aria-label="Markdown formatting"
      className={cn('flex flex-wrap items-center gap-0.5', className)}
    >
      {ACTIONS.map((action, index) => {
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

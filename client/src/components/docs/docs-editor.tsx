import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from 'react'

import { ComposerAutocomplete } from '@/components/chat/composer-autocomplete'
import { EmojiPickerButton } from '@/components/chat/emoji-picker'
import {
  ComposerMarkdownToolbar,
  type ComposerSelectionEdit,
} from '@/components/chat/composer-markdown-toolbar'
import { Textarea } from '@/components/ui/textarea'
import {
  applyAutocompleteInsertion,
  composerAutocompleteInsertion,
  detectComposerTrigger,
  listComposerAutocompleteItems,
} from '@/lib/composer-autocomplete'
import { useLocale } from '@/lib/i18n'

function insertAtSelection(
  value: string,
  start: number,
  end: number,
  text: string,
): ComposerSelectionEdit {
  const next = `${value.slice(0, start)}${text}${value.slice(end)}`
  const caret = start + text.length
  return { next, selectionStart: caret, selectionEnd: caret }
}

export function DocsEditor({
  value,
  onChange,
  disabled,
}: {
  value: string
  onChange: (next: string) => void
  disabled?: boolean
}) {
  const { t } = useLocale()
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const [caret, setCaret] = useState(value.length)
  const [autocompleteIndex, setAutocompleteIndex] = useState(0)
  const [autocompleteDismissed, setAutocompleteDismissed] = useState(false)

  const applyEdit = (edit: ComposerSelectionEdit) => {
    onChange(edit.next)
    requestAnimationFrame(() => {
      const el = inputRef.current
      if (!el) return
      el.focus()
      el.setSelectionRange(edit.selectionStart, edit.selectionEnd)
      setCaret(edit.selectionEnd)
    })
  }

  const trigger = useMemo(() => {
    const found = detectComposerTrigger(value, caret)
    return found?.kind === 'emoji' ? found : null
  }, [value, caret])
  const activeTrigger = autocompleteDismissed ? null : trigger
  const autocompleteItems = useMemo(
    () =>
      activeTrigger
        ? listComposerAutocompleteItems(activeTrigger, {
            members: [],
            channels: [],
          })
        : [],
    [activeTrigger],
  )
  const autocompleteOpen = Boolean(activeTrigger)
  useEffect(() => {
    if (!trigger) setAutocompleteDismissed(false)
  }, [trigger])

  useEffect(() => {
    setAutocompleteIndex(0)
  }, [activeTrigger?.start, activeTrigger?.query])

  const highlighted =
    autocompleteItems[
      Math.min(autocompleteIndex, Math.max(autocompleteItems.length - 1, 0))
    ] ?? null

  const selectItem = (item: (typeof autocompleteItems)[number]) => {
    if (!activeTrigger) return
    applyEdit(
      applyAutocompleteInsertion(
        value,
        activeTrigger,
        caret,
        composerAutocompleteInsertion(item),
      ),
    )
    setAutocompleteDismissed(false)
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (!autocompleteOpen || event.nativeEvent.isComposing) return
    if (event.key === 'Escape') {
      event.preventDefault()
      setAutocompleteDismissed(true)
      return
    }
    if (autocompleteItems.length === 0 || !highlighted) return
    if (event.key === 'ArrowDown') {
      event.preventDefault()
      setAutocompleteIndex((index) => (index + 1) % autocompleteItems.length)
      return
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault()
      setAutocompleteIndex(
        (index) =>
          (index - 1 + autocompleteItems.length) % autocompleteItems.length,
      )
      return
    }
    if (event.key === 'Enter' || event.key === 'Tab') {
      event.preventDefault()
      selectItem(highlighted)
    }
  }

  return (
    <div className="relative flex min-h-0 flex-1 flex-col gap-2">
      <div className="flex flex-wrap items-center gap-0.5">
        <ComposerMarkdownToolbar
          disabled={disabled}
          value={value}
          getSelection={() => ({
            start: inputRef.current?.selectionStart ?? value.length,
            end: inputRef.current?.selectionEnd ?? value.length,
          })}
          onApply={applyEdit}
        />
        <EmojiPickerButton
          disabled={disabled}
          onSelect={(shortcode) => {
            const start = inputRef.current?.selectionStart ?? value.length
            const end = inputRef.current?.selectionEnd ?? value.length
            const before = value[start - 1]
            const prefix = before && !/\s/.test(before) ? ' ' : ''
            applyEdit(insertAtSelection(value, start, end, `${prefix}${shortcode} `))
          }}
        />
      </div>
      <div className="relative min-h-0 flex-1">
        <ComposerAutocomplete
          open={autocompleteOpen}
          kind="emoji"
          items={autocompleteItems}
          activeIndex={Math.min(
            autocompleteIndex,
            Math.max(autocompleteItems.length - 1, 0),
          )}
          onActiveIndexChange={setAutocompleteIndex}
          onSelect={selectItem}
        />
        <Textarea
          ref={inputRef}
          value={value}
          disabled={disabled}
          onChange={(event) => {
            onChange(event.target.value)
            setCaret(event.target.selectionStart)
            setAutocompleteDismissed(false)
          }}
          onSelect={(event) => setCaret(event.currentTarget.selectionStart)}
          onKeyUp={(event) => setCaret(event.currentTarget.selectionStart)}
          onClick={(event) => setCaret(event.currentTarget.selectionStart)}
          onKeyDown={handleKeyDown}
          placeholder={t('docs.bodyPlaceholder')}
          className="min-h-64 flex-1 resize-y font-mono text-sm"
        />
      </div>
    </div>
  )
}

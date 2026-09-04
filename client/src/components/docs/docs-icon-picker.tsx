import { FileTextIcon } from '@phosphor-icons/react'
import { useState } from 'react'

import { EmojiPickerPopover } from '@/components/chat/emoji-picker'
import { Button } from '@/components/ui/button'
import { resolveEmoji } from '@/lib/emoji'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

export function DocsIconPicker({
  value,
  onChange,
  disabled,
  className,
}: {
  value: string
  onChange: (icon: string) => void
  disabled?: boolean
  className?: string
}) {
  const { t } = useLocale()
  const [open, setOpen] = useState(false)

  return (
    <EmojiPickerPopover
      open={open}
      onOpenChange={setOpen}
      align="start"
      side="bottom"
      onSelect={(emojiId) => {
        const resolved = resolveEmoji(emojiId)
        if (resolved?.kind === 'unicode') {
          onChange(resolved.native)
        }
        setOpen(false)
      }}
    >
      <Button
        type="button"
        variant="outline"
        size="icon"
        disabled={disabled}
        className={cn('size-9 shrink-0 text-base', className)}
        aria-label={value ? t('docs.changeIcon') : t('docs.addIcon')}
        title={value ? t('docs.changeIcon') : t('docs.addIcon')}
        onContextMenu={(event) => {
          if (!value) return
          event.preventDefault()
          onChange('')
        }}
      >
        {value ? (
          <span aria-hidden className="leading-none">
            {value}
          </span>
        ) : (
          <FileTextIcon className="size-4 text-muted-foreground" />
        )}
      </Button>
    </EmojiPickerPopover>
  )
}

import { SmileyIcon } from '@phosphor-icons/react'
import { useMemo, useState, type ReactNode } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { InputGroupButton } from '@/components/ui/input-group'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  getEmojiCatalog,
  searchEmojiCatalog,
  toSlackShortcode,
  type EmojiCatalogItem,
} from '@/lib/emoji'
import { cn } from '@/lib/utils'

function EmojiGrid({
  emojis,
  onSelect,
}: {
  emojis: EmojiCatalogItem[]
  onSelect: (emoji: EmojiCatalogItem) => void
}) {
  return (
    <div className="grid grid-cols-8 gap-0.5">
      {emojis.map((emoji) => (
        <button
          key={emoji.id}
          type="button"
          title={`:${emoji.id}: ${emoji.name}`}
          aria-label={`${emoji.name}, :${emoji.id}:`}
          className={cn(
            'flex size-8 items-center justify-center rounded-md text-lg leading-none',
            'hover:bg-muted focus-visible:bg-muted focus-visible:outline-none',
          )}
          onClick={() => onSelect(emoji)}
        >
          <span aria-hidden>{emoji.native}</span>
        </button>
      ))}
    </div>
  )
}

export function EmojiPickerPopover({
  open,
  onOpenChange,
  onSelect,
  children,
  align = 'end',
  side = 'top',
}: {
  open?: boolean
  onOpenChange?: (open: boolean) => void
  onSelect: (emojiId: string) => void
  children: ReactNode
  align?: 'start' | 'center' | 'end'
  side?: 'top' | 'right' | 'bottom' | 'left'
}) {
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false)
  const [query, setQuery] = useState('')
  const isControlled = open !== undefined
  const resolvedOpen = isControlled ? open : uncontrolledOpen
  const setOpen = (next: boolean) => {
    if (!isControlled) setUncontrolledOpen(next)
    onOpenChange?.(next)
    if (!next) setQuery('')
  }

  const catalog = useMemo(() => getEmojiCatalog(), [])
  const searchResults = useMemo(
    () => (query.trim() ? searchEmojiCatalog(query) : null),
    [query],
  )

  const handleSelect = (emoji: EmojiCatalogItem) => {
    onSelect(emoji.id)
    setOpen(false)
  }

  return (
    <Popover open={resolvedOpen} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent
        align={align}
        side={side}
        className="w-[22rem] gap-2 p-2"
        onOpenAutoFocus={(event) => event.preventDefault()}
        onCloseAutoFocus={(event) => event.preventDefault()}
      >
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search emoji"
          aria-label="Search emoji"
          autoComplete="off"
        />
        <ScrollArea className="h-64 pr-2">
          {searchResults ? (
            searchResults.length === 0 ? (
              <p className="px-1 py-6 text-center text-sm text-muted-foreground">
                No emoji match that search.
              </p>
            ) : (
              <EmojiGrid emojis={searchResults} onSelect={handleSelect} />
            )
          ) : (
            <div className="flex flex-col gap-3">
              {catalog.map((category) => (
                <section key={category.id} className="flex flex-col gap-1.5">
                  <h3 className="px-1 text-xs font-medium text-muted-foreground">
                    {category.label}
                  </h3>
                  <EmojiGrid emojis={category.emojis} onSelect={handleSelect} />
                </section>
              ))}
            </div>
          )}
        </ScrollArea>
      </PopoverContent>
    </Popover>
  )
}

export function EmojiPickerButton({
  disabled,
  onSelect,
  className,
}: {
  disabled?: boolean
  onSelect: (shortcode: string) => void
  className?: string
}) {
  return (
    <EmojiPickerPopover
      onSelect={(emojiId) => onSelect(toSlackShortcode(emojiId))}
    >
      <InputGroupButton
        type="button"
        size="icon-xs"
        variant="ghost"
        disabled={disabled}
        className={className}
        aria-label="Insert emoji"
        title="Insert emoji"
      >
        <SmileyIcon strokeWidth={2} />
      </InputGroupButton>
    </EmojiPickerPopover>
  )
}

export function ReactionPickerButton({
  disabled,
  onSelect,
  className,
}: {
  disabled?: boolean
  onSelect: (emojiId: string) => void
  className?: string
}) {
  return (
    <EmojiPickerPopover onSelect={onSelect} align="end" side="bottom">
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        disabled={disabled}
        className={className}
        aria-label="Add reaction"
        title="Add reaction"
        onClick={(event) => event.stopPropagation()}
      >
        <SmileyIcon strokeWidth={2} />
      </Button>
    </EmojiPickerPopover>
  )
}

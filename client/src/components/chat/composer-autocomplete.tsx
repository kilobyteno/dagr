import { BookOpenIcon, HashIcon, LockKeyIcon } from '@phosphor-icons/react'
import { useEffect, useRef } from 'react'

import { UserAvatarMark } from '@/components/chat/user-avatar'
import {
  COMPOSER_AUTOCOMPLETE_LIST_ID,
  type ComposerAutocompleteItem,
} from '@/lib/composer-autocomplete'
import { useLocale } from '@/lib/i18n'
import { cn } from '@/lib/utils'

export function ComposerAutocomplete({
  open,
  kind,
  items,
  activeIndex,
  onActiveIndexChange,
  onSelect,
  serverUrl,
  token,
}: {
  open: boolean
  kind: 'mention' | 'channel' | 'document' | 'emoji'
  items: ComposerAutocompleteItem[]
  activeIndex: number
  onActiveIndexChange: (index: number) => void
  onSelect: (item: ComposerAutocompleteItem) => void
  serverUrl?: string
  token?: string
}) {
  const { t } = useLocale()
  const activeRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    const el = activeRef.current
    if (!el) return
    const root = el.closest('[data-autocomplete-scroll]')
    if (!(root instanceof HTMLElement)) return
    const elRect = el.getBoundingClientRect()
    const rootRect = root.getBoundingClientRect()
    if (elRect.bottom > rootRect.bottom) {
      root.scrollTop += elRect.bottom - rootRect.bottom
    } else if (elRect.top < rootRect.top) {
      root.scrollTop -= rootRect.top - elRect.top
    }
  }, [activeIndex, items])

  if (!open) return null

  const emptyKey =
    kind === 'mention'
      ? 'chat.autocomplete.mentionsEmpty'
      : kind === 'channel'
        ? 'chat.autocomplete.channelsEmpty'
        : kind === 'document'
          ? 'chat.autocomplete.documentsEmpty'
          : 'chat.autocomplete.emojiEmpty'
  const labelKey =
    kind === 'mention'
      ? 'chat.autocomplete.mentionsLabel'
      : kind === 'channel'
        ? 'chat.autocomplete.channelsLabel'
        : kind === 'document'
          ? 'chat.autocomplete.documentsLabel'
          : 'chat.autocomplete.emojiLabel'

  return (
    <div
      id={COMPOSER_AUTOCOMPLETE_LIST_ID}
      role="listbox"
      aria-label={t(labelKey)}
      className="absolute inset-x-0 bottom-full z-50 mb-1 overflow-hidden rounded-lg bg-popover p-1 shadow-md ring-1 ring-foreground/10"
    >
      {items.length === 0 ? (
        <p className="px-2 py-3 text-center text-sm text-muted-foreground">
          {t(emptyKey)}
        </p>
      ) : (
        <div data-autocomplete-scroll className="max-h-64 overflow-y-auto">
          <ul className="flex flex-col gap-0.5 p-0.5">
            {items.map((item, index) => {
              const active = index === activeIndex
              return (
                <li key={item.id} role="presentation">
                  <button
                    ref={active ? activeRef : undefined}
                    type="button"
                    id={item.id}
                    role="option"
                    aria-selected={active}
                    className={cn(
                      'flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm',
                      'hover:bg-muted focus-visible:bg-muted focus-visible:outline-none',
                      active && 'bg-muted',
                    )}
                    onMouseEnter={() => onActiveIndexChange(index)}
                    onMouseDown={(event) => event.preventDefault()}
                    onClick={() => onSelect(item)}
                  >
                    {item.kind === 'mention' ? (
                      <>
                        <UserAvatarMark
                          userId={item.user.userId}
                          name={item.user.displayName || item.user.handle}
                          hasAvatar={item.user.hasAvatar}
                          avatarUpdatedAt={item.user.avatarUpdatedAt}
                          presence={item.user.presence}
                          showPresence
                          serverUrl={serverUrl}
                          token={token}
                          size="sm"
                        />
                        <span className="min-w-0 flex-1 truncate font-medium">
                          {item.user.displayName || item.user.handle}
                        </span>
                        <span className="shrink-0 text-xs text-muted-foreground">
                          @{item.user.handle}
                        </span>
                      </>
                    ) : item.kind === 'channel' ? (
                      <>
                        {item.channel.isPrivate ? (
                          <LockKeyIcon
                            strokeWidth={2}
                            className="size-4 shrink-0 text-muted-foreground"
                          />
                        ) : (
                          <HashIcon
                            strokeWidth={2}
                            className="size-4 shrink-0 text-muted-foreground"
                          />
                        )}
                        <span className="min-w-0 flex-1 truncate font-medium">
                          {item.channel.name}
                        </span>
                        {item.channel.topic ? (
                          <span className="min-w-0 max-w-[40%] truncate text-xs text-muted-foreground">
                            {item.channel.topic}
                          </span>
                        ) : null}
                      </>
                    ) : item.kind === 'document' ? (
                      <>
                        <BookOpenIcon
                          strokeWidth={2}
                          className="size-4 shrink-0 text-muted-foreground"
                        />
                        <span className="min-w-0 flex-1 truncate font-medium">
                          {item.document.title}
                        </span>
                        <span className="shrink-0 text-xs text-muted-foreground">
                          [[{item.document.slug}]]
                        </span>
                      </>
                    ) : (
                      <>
                        <span
                          aria-hidden
                          className="flex size-6 items-center justify-center text-lg leading-none"
                        >
                          {item.emoji.native}
                        </span>
                        <span className="min-w-0 flex-1 truncate">
                          {item.emoji.name}
                        </span>
                        <span className="shrink-0 text-xs text-muted-foreground">
                          :{item.emoji.id}:
                        </span>
                      </>
                    )}
                  </button>
                </li>
              )
            })}
          </ul>
        </div>
      )}
    </div>
  )
}

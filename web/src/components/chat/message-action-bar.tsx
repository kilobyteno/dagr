import {
  CopyIcon,
  DotsThreeVerticalIcon,
  EnvelopeSimpleIcon,
  PencilSimpleIcon,
  PlusIcon,
  SmileyIcon,
  TrashIcon,
} from '@phosphor-icons/react'
import {
  forwardRef,
  useState,
  type ButtonHTMLAttributes,
  type ReactNode,
} from 'react'
import { toast } from 'sonner'

import { EmojiPickerPopover } from '@/components/chat/emoji-picker'
import { MessageEmoji } from '@/components/chat/message-emoji'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

const QUICK_REACTIONS = [
  { id: 'white_check_mark', label: 'React with white check mark' },
  { id: 'eyes', label: 'React with eyes' },
  { id: 'raised_hands', label: 'React with raising hands' },
] as const

const ToolbarIconButton = forwardRef<
  HTMLButtonElement,
  {
    label: string
    disabled?: boolean
    className?: string
    children: ReactNode
  } & Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'type' | 'children'>
>(function ToolbarIconButton(
  { label, disabled, className, children, onClick, onPointerDown, ...props },
  ref,
) {
  return (
    <Button
      ref={ref}
      type="button"
      variant="ghost"
      size="icon-xs"
      disabled={disabled}
      title={label}
      {...props}
      aria-label={label}
      className={cn(
        'size-7 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground',
        className,
      )}
      onPointerDown={(event) => {
        event.stopPropagation()
        onPointerDown?.(event)
      }}
      onClick={(event) => {
        event.stopPropagation()
        onClick?.(event)
      }}
    >
      {children}
    </Button>
  )
})

function AddReactionIcon() {
  return (
    <span className="relative inline-flex size-4 items-center justify-center">
      <SmileyIcon strokeWidth={2} className="size-4" />
      <PlusIcon
        weight="bold"
        className="absolute -top-0.5 -right-1 size-2.5 text-muted-foreground"
        aria-hidden
      />
    </span>
  )
}

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    toast.success('Message copied')
  } catch {
    toast.error('Could not copy message')
  }
}

export function MessageActionBar({
  isOwn,
  isUnread,
  disabled,
  forceVisible,
  messageBody,
  onReact,
  onMarkUnread,
  onEdit,
  onDelete,
  className,
}: {
  isOwn: boolean
  isUnread: boolean
  disabled?: boolean
  /** Keep the bar visible (for example when the message is keyboard-selected). */
  forceVisible?: boolean
  messageBody: string
  onReact: (emojiId: string) => void
  onMarkUnread?: () => void
  onEdit?: () => void
  onDelete?: () => void
  className?: string
}) {
  const [pickerOpen, setPickerOpen] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const canMarkUnread = !isOwn && !isUnread && Boolean(onMarkUnread)
  const stayOpen = pickerOpen || menuOpen || forceVisible

  return (
    <div
      className={cn(
        'pointer-events-none absolute -top-3 right-2 z-20 opacity-0 transition-opacity',
        'group-hover/message:pointer-events-auto group-hover/message:opacity-100',
        'group-focus-within/message:pointer-events-auto group-focus-within/message:opacity-100',
        stayOpen && 'pointer-events-auto opacity-100',
        className,
      )}
    >
      <div className="flex items-center gap-0.5 rounded-lg border border-border bg-popover p-0.5 shadow-md">
        {QUICK_REACTIONS.map((reaction) => (
          <ToolbarIconButton
            key={reaction.id}
            label={reaction.label}
            disabled={disabled}
            className="text-[1.05rem] leading-none text-foreground"
            onClick={() => onReact(reaction.id)}
          >
            <MessageEmoji name={reaction.id} className="text-[1.05rem]" />
          </ToolbarIconButton>
        ))}

        <EmojiPickerPopover
          open={pickerOpen}
          onOpenChange={setPickerOpen}
          onSelect={onReact}
          align="end"
          side="bottom"
        >
          <ToolbarIconButton label="Add reaction" disabled={disabled}>
            <AddReactionIcon />
          </ToolbarIconButton>
        </EmojiPickerPopover>

        <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
          <DropdownMenuTrigger asChild>
            <ToolbarIconButton label="More actions" disabled={disabled}>
              <DotsThreeVerticalIcon strokeWidth={2} className="size-4" />
            </ToolbarIconButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="min-w-52">
            {canMarkUnread ? (
              <>
                <DropdownMenuItem
                  disabled={disabled}
                  onSelect={() => onMarkUnread?.()}
                >
                  <EnvelopeSimpleIcon strokeWidth={2} />
                  Mark unread
                  <DropdownMenuShortcut>U</DropdownMenuShortcut>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
              </>
            ) : null}

            <DropdownMenuItem
              disabled={disabled}
              onSelect={() => {
                void copyText(messageBody)
              }}
            >
              <CopyIcon strokeWidth={2} />
              Copy message
              <DropdownMenuShortcut>⌘C</DropdownMenuShortcut>
            </DropdownMenuItem>

            {isOwn && (onEdit || onDelete) ? (
              <>
                <DropdownMenuSeparator />
                {onEdit ? (
                  <DropdownMenuItem
                    disabled={disabled}
                    onSelect={() => onEdit()}
                  >
                    <PencilSimpleIcon strokeWidth={2} />
                    Edit message
                    <DropdownMenuShortcut>E</DropdownMenuShortcut>
                  </DropdownMenuItem>
                ) : null}
                {onDelete ? (
                  <DropdownMenuItem
                    variant="destructive"
                    disabled={disabled}
                    onSelect={() => onDelete()}
                  >
                    <TrashIcon strokeWidth={2} />
                    Delete message…
                    <DropdownMenuShortcut>⌫</DropdownMenuShortcut>
                  </DropdownMenuItem>
                ) : null}
              </>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}

import { SmileyIcon } from '@phosphor-icons/react'
import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { EmojiPickerPopover } from '@/components/chat/emoji-picker'
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
import { updateStatus, type ApiUser } from '@/lib/api/auth'
import { ApiError } from '@/lib/api/client'
import { resolveEmoji } from '@/lib/emoji'
import { useAuth } from '@/lib/auth'
import { cn } from '@/lib/utils'

type DurationPreset = '30m' | '1h' | '3h' | 'custom' | 'none'

const DURATION_OPTIONS: Array<{ id: DurationPreset; label: string }> = [
  { id: '30m', label: '30 minutes' },
  { id: '1h', label: '1 hour' },
  { id: '3h', label: '3 hours' },
  { id: 'custom', label: 'Custom' },
  { id: 'none', label: "Don't clear" },
]

function toDatetimeLocalValue(date: Date) {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function fromDatetimeLocalValue(value: string): Date | null {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return date
}

function expiresAtForPreset(preset: DurationPreset, customLocal: string): Date | null {
  const now = Date.now()
  switch (preset) {
    case '30m':
      return new Date(now + 30 * 60_000)
    case '1h':
      return new Date(now + 60 * 60_000)
    case '3h':
      return new Date(now + 3 * 60 * 60_000)
    case 'custom':
      return fromDatetimeLocalValue(customLocal)
    case 'none':
      return null
  }
}

function inferPreset(expiresAt?: string | null): DurationPreset {
  if (!expiresAt) return 'none'
  const ms = Date.parse(expiresAt) - Date.now()
  if (!Number.isFinite(ms) || ms <= 0) return 'custom'
  const minutes = Math.round(ms / 60_000)
  if (Math.abs(minutes - 30) <= 2) return '30m'
  if (Math.abs(minutes - 60) <= 3) return '1h'
  if (Math.abs(minutes - 180) <= 5) return '3h'
  return 'custom'
}

export function SetStatusDialog({
  open,
  onOpenChange,
  serverUrl,
  token,
  onSaved,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  serverUrl?: string
  token?: string
  onSaved?: (user: ApiUser) => void
}) {
  const { session, signIn } = useAuth()
  const [emoji, setEmoji] = useState('')
  const [text, setText] = useState('')
  const [duration, setDuration] = useState<DurationPreset>('1h')
  const [customExpiresLocal, setCustomExpiresLocal] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [pickerOpen, setPickerOpen] = useState(false)

  useEffect(() => {
    if (open && session) {
      setEmoji(session.statusEmoji ?? '')
      setText(session.statusText ?? '')
      const preset = inferPreset(session.statusExpiresAt)
      setDuration(preset)
      if (session.statusExpiresAt) {
        const date = new Date(session.statusExpiresAt)
        setCustomExpiresLocal(
          Number.isNaN(date.getTime())
            ? toDatetimeLocalValue(new Date(Date.now() + 60 * 60_000))
            : toDatetimeLocalValue(date),
        )
      } else {
        setCustomExpiresLocal(
          toDatetimeLocalValue(new Date(Date.now() + 60 * 60_000)),
        )
      }
      setSubmitting(false)
      setPickerOpen(false)
    }
  }, [open, session])

  if (!session) return null

  const applyUser = (user: ApiUser) => {
    signIn({
      ...session,
      displayName: user.displayName,
      notificationLevel: user.notificationLevel,
      emailVerified: Boolean(user.emailVerified),
      statusEmoji: user.statusEmoji ?? '',
      statusText: user.statusText ?? '',
      statusExpiresAt: user.statusExpiresAt ?? null,
      hasAvatar: Boolean(user.hasAvatar),
      avatarUpdatedAt: user.avatarUpdatedAt ?? null,
    })
    onSaved?.(user)
  }

  const save = async (nextEmoji: string, nextText: string, clear = false) => {
    if (!serverUrl || !token || submitting) return
    let expiresAt: string | null = null
    if (!clear && (nextEmoji || nextText)) {
      const expires = expiresAtForPreset(duration, customExpiresLocal)
      if (duration === 'custom') {
        if (!expires || expires.getTime() <= Date.now()) {
          toast.error('Choose a custom end time in the future')
          return
        }
      }
      expiresAt = expires ? expires.toISOString() : null
    }
    setSubmitting(true)
    try {
      const result = await updateStatus(serverUrl, token, {
        emoji: nextEmoji,
        text: nextText,
        expiresAt,
      })
      applyUser(result.user)
      onOpenChange(false)
      toast.success(
        nextEmoji || nextText ? 'Status updated' : 'Status cleared',
      )
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : 'Could not update status'
      toast.error(message)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Set a status</DialogTitle>
          <DialogDescription>
            Add an emoji and short note. It appears next to your name in chat.
          </DialogDescription>
        </DialogHeader>
        <form
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault()
            void save(emoji, text.trim())
          }}
        >
          <div className="flex items-end gap-3">
            <div className="grid gap-2">
              <Label htmlFor="status-emoji">Emoji</Label>
              <EmojiPickerPopover
                open={pickerOpen}
                onOpenChange={setPickerOpen}
                align="start"
                side="bottom"
                onSelect={(emojiId) => {
                  const resolved = resolveEmoji(emojiId)
                  if (resolved?.kind === 'unicode') {
                    setEmoji(resolved.native)
                  }
                }}
              >
                <Button
                  id="status-emoji"
                  type="button"
                  variant="outline"
                  className="size-11 px-0 text-xl"
                  aria-label="Choose status emoji"
                >
                  {emoji || <SmileyIcon strokeWidth={2} />}
                </Button>
              </EmojiPickerPopover>
            </div>
            <div className="grid min-w-0 flex-1 gap-2">
              <Label htmlFor="status-text">Status</Label>
              <Input
                id="status-text"
                value={text}
                onChange={(event) => setText(event.target.value)}
                placeholder="In a meeting"
                maxLength={100}
                autoComplete="off"
              />
            </div>
          </div>
          <div className="grid gap-2">
            <Label>Clear after</Label>
            <div className="flex flex-wrap gap-2">
              {DURATION_OPTIONS.map((option) => (
                <Button
                  key={option.id}
                  type="button"
                  size="sm"
                  variant={duration === option.id ? 'default' : 'outline'}
                  className={cn(duration === option.id && 'pointer-events-none')}
                  onClick={() => setDuration(option.id)}
                >
                  {option.label}
                </Button>
              ))}
            </div>
            {duration === 'custom' ? (
              <div className="grid gap-2 pt-1">
                <Label htmlFor="status-expires-custom">Ends</Label>
                <Input
                  id="status-expires-custom"
                  type="datetime-local"
                  value={customExpiresLocal}
                  min={toDatetimeLocalValue(new Date())}
                  onChange={(event) => setCustomExpiresLocal(event.target.value)}
                />
              </div>
            ) : null}
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              disabled={submitting || (!emoji && !text.trim())}
              onClick={() => {
                void save('', '', true)
              }}
            >
              Clear status
            </Button>
            <Button
              type="submit"
              disabled={submitting || (!emoji && !text.trim())}
            >
              {submitting ? 'Saving…' : 'Save'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

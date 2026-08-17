import { CheckIcon, CircleIcon, EyeIcon, EyeSlashIcon } from '@phosphor-icons/react'
import { useId, useState } from 'react'

import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from '@/components/ui/input-group'
import {
  getPasswordRequirements,
  type PasswordPolicy,
} from '@/lib/password-policy'
import { cn } from '@/lib/utils'

type PasswordInputProps = Omit<
  React.ComponentProps<'input'>,
  'type' | 'value' | 'onChange'
> & {
  value: string
  onChange: (value: string) => void
  policy?: PasswordPolicy
  showRequirements?: boolean
}

export function PasswordInput({
  id,
  value,
  onChange,
  policy,
  showRequirements = Boolean(policy),
  className,
  onFocus,
  onBlur,
  ...props
}: PasswordInputProps) {
  const generatedId = useId()
  const inputId = id ?? generatedId
  const requirementsId = `${inputId}-requirements`
  const [visible, setVisible] = useState(false)
  const [focused, setFocused] = useState(false)

  const requirements = policy ? getPasswordRequirements(value, policy) : []
  const showChecklist =
    showRequirements && policy && (focused || value.length > 0)

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      <InputGroup>
        <InputGroupInput
          id={inputId}
          type={visible ? 'text' : 'password'}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          onFocus={(event) => {
            setFocused(true)
            onFocus?.(event)
          }}
          onBlur={(event) => {
            setFocused(false)
            onBlur?.(event)
          }}
          aria-describedby={showChecklist ? requirementsId : undefined}
          {...props}
        />
        <InputGroupAddon align="inline-end">
          <InputGroupButton
            size="icon-xs"
            variant="ghost"
            aria-label={visible ? 'Hide password' : 'Show password'}
            aria-pressed={visible}
            onClick={() => setVisible((current) => !current)}
          >
            {visible ? <EyeSlashIcon strokeWidth={2} /> : <EyeIcon strokeWidth={2} />}
          </InputGroupButton>
        </InputGroupAddon>
      </InputGroup>

      {showChecklist && (
        <ul
          id={requirementsId}
          className="grid gap-1.5 rounded-lg border border-border/70 bg-muted/30 px-3 py-2.5 animate-in fade-in-0 slide-in-from-top-1 duration-200"
        >
          {requirements.map((requirement) => (
            <li
              key={requirement.id}
              className={cn(
                'flex items-center gap-2 text-xs transition-colors',
                requirement.met ? 'text-foreground' : 'text-muted-foreground',
              )}
            >
              {requirement.met ? (
                <CheckIcon strokeWidth={2} className="size-3.5 text-primary" aria-hidden />
              ) : (
                <CircleIcon strokeWidth={2} className="size-3.5" aria-hidden />
              )}
              <span>{requirement.label}</span>
              <span className="sr-only">
                {requirement.met ? 'met' : 'not met'}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

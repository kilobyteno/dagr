export type PasswordPolicy = {
  minLength: number
  requireUppercase: boolean
  requireLowercase: boolean
  requireNumber: boolean
  requireSymbol: boolean
}

export type PasswordRequirement = {
  id: string
  label: string
  met: boolean
}

export function getPasswordRequirements(
  password: string,
  policy: PasswordPolicy,
): PasswordRequirement[] {
  const requirements: PasswordRequirement[] = [
    {
      id: 'length',
      label: `At least ${policy.minLength} characters`,
      met: password.length >= policy.minLength,
    },
  ]

  if (policy.requireUppercase) {
    requirements.push({
      id: 'uppercase',
      label: 'One uppercase letter',
      met: /[A-Z]/.test(password),
    })
  }
  if (policy.requireLowercase) {
    requirements.push({
      id: 'lowercase',
      label: 'One lowercase letter',
      met: /[a-z]/.test(password),
    })
  }
  if (policy.requireNumber) {
    requirements.push({
      id: 'number',
      label: 'One number',
      met: /\d/.test(password),
    })
  }
  if (policy.requireSymbol) {
    requirements.push({
      id: 'symbol',
      label: 'One symbol',
      met: /[^\dA-Za-z]/.test(password),
    })
  }

  return requirements
}

export function validatePassword(
  password: string,
  policy: PasswordPolicy,
): string | null {
  const unmet = getPasswordRequirements(password, policy).find((item) => !item.met)
  if (!unmet) return null

  switch (unmet.id) {
    case 'length':
      return `Password must be at least ${policy.minLength} characters`
    case 'uppercase':
      return 'Password must include an uppercase letter'
    case 'lowercase':
      return 'Password must include a lowercase letter'
    case 'number':
      return 'Password must include a number'
    case 'symbol':
      return 'Password must include a symbol'
    default:
      return 'Password does not meet the server requirements'
  }
}

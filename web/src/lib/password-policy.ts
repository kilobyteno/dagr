export type PasswordPolicy = {
  minLength: number
  requireUppercase: boolean
  requireLowercase: boolean
  requireNumber: boolean
  requireSymbol: boolean
}

export type PasswordRequirement = {
  id: 'length' | 'uppercase' | 'lowercase' | 'number' | 'symbol'
  met: boolean
}

export function getPasswordRequirements(
  password: string,
  policy: PasswordPolicy,
): PasswordRequirement[] {
  const requirements: PasswordRequirement[] = [
    {
      id: 'length',
      met: password.length >= policy.minLength,
    },
  ]

  if (policy.requireUppercase) {
    requirements.push({
      id: 'uppercase',
      met: /[A-Z]/.test(password),
    })
  }
  if (policy.requireLowercase) {
    requirements.push({
      id: 'lowercase',
      met: /[a-z]/.test(password),
    })
  }
  if (policy.requireNumber) {
    requirements.push({
      id: 'number',
      met: /\d/.test(password),
    })
  }
  if (policy.requireSymbol) {
    requirements.push({
      id: 'symbol',
      met: /[^\dA-Za-z]/.test(password),
    })
  }

  return requirements
}

export function validatePassword(
  password: string,
  policy: PasswordPolicy,
): { id: PasswordRequirement['id'] | 'default'; minLength: number } | null {
  const unmet = getPasswordRequirements(password, policy).find((item) => !item.met)
  if (!unmet) return null
  return { id: unmet.id, minLength: policy.minLength }
}

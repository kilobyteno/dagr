/**
 * SemVer 2.0.0 (https://semver.org). Build metadata is parsed and ignored in
 * precedence. A leading v on tags is stripped.
 */

const IDENT = '(?:0|[1-9][0-9]*|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)'
const PRERELEASE = `(?:-(${IDENT}(?:\\.${IDENT})*))`
const BUILD = `(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))`
const CORE = '(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)'

export const SEMVER_PATTERN = new RegExp(`^${CORE}${PRERELEASE}?${BUILD}?$`)

export type SemVer = {
  major: number
  minor: number
  patch: number
  prerelease: string[]
  build: string[]
}

export function stripTagPrefix(value: string) {
  return value.trim().replace(/^v/i, '')
}

export function parseSemVer(value: string): SemVer | null {
  const match = SEMVER_PATTERN.exec(stripTagPrefix(value))
  if (!match) return null
  return {
    major: Number(match[1]),
    minor: Number(match[2]),
    patch: Number(match[3]),
    prerelease: match[4] ? match[4].split('.') : [],
    build: match[5] ? match[5].split('.') : [],
  }
}

export function isSemVer(value: string) {
  return parseSemVer(value) !== null
}

function compareIdentifiers(a: string, b: string) {
  const aNumeric = /^(0|[1-9][0-9]*)$/.test(a)
  const bNumeric = /^(0|[1-9][0-9]*)$/.test(b)
  if (aNumeric && bNumeric) {
    const delta = Number(a) - Number(b)
    if (delta > 0) return 1
    if (delta < 0) return -1
    return 0
  }
  if (aNumeric) return -1
  if (bNumeric) return 1
  if (a > b) return 1
  if (a < b) return -1
  return 0
}

/** Negative if a < b, zero if equal, positive if a > b. Build is ignored. */
export function compareSemVer(a: string, b: string): number | null {
  const left = parseSemVer(a)
  const right = parseSemVer(b)
  if (!left || !right) return null

  if (left.major !== right.major) return left.major > right.major ? 1 : -1
  if (left.minor !== right.minor) return left.minor > right.minor ? 1 : -1
  if (left.patch !== right.patch) return left.patch > right.patch ? 1 : -1

  const leftPre = left.prerelease
  const rightPre = right.prerelease
  if (leftPre.length === 0 && rightPre.length === 0) return 0
  if (leftPre.length === 0) return 1
  if (rightPre.length === 0) return -1

  const length = Math.max(leftPre.length, rightPre.length)
  for (let index = 0; index < length; index += 1) {
    const leftIdent = leftPre[index]
    const rightIdent = rightPre[index]
    if (leftIdent === undefined) return -1
    if (rightIdent === undefined) return 1
    const delta = compareIdentifiers(leftIdent, rightIdent)
    if (delta !== 0) return delta
  }
  return 0
}

export function isNewerVersion(latest: string, current: string): boolean {
  const delta = compareSemVer(latest, current)
  return delta !== null && delta > 0
}

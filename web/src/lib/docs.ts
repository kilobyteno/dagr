/** Public documentation site. */
export const DOCS_BASE_URL = 'https://docs.dagr.no'

export function docsUrl(path = ''): string {
  const normalised = path.trim().replace(/^\//, '')
  if (!normalised) return DOCS_BASE_URL
  return `${DOCS_BASE_URL}/${normalised}`
}

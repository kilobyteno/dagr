export const SITE_URL = (
  process.env.SITE_URL?.trim() || 'https://www.dagr.no'
).replace(/\/$/, '')

export const SITE_NAME = 'Dagr'
export const SITE_TITLE = 'Dagr | Self-hosted team chat'
export const SITE_TAGLINE = 'Team chat you host yourself.'
export const SITE_DESCRIPTION =
  'Privacy-centric, self-hostable team chat. A Slack alternative you run yourself.'

export const DOCS_URL = 'https://docs.page/kilobyteno/dagr'
export const DOCS_COMPARE_URL = `${DOCS_URL}/compare`
export const DOCS_PLANS_URL = `${DOCS_URL}/cloud/plans`
export const DOCS_SELF_HOST_URL = `${DOCS_URL}/hosting/self-hosting`
export const DOCS_QUICKSTART_URL = `${DOCS_URL}/quickstart`

export const GITHUB_REPO = 'kilobyteno/dagr'
export const GITHUB_URL = `https://github.com/${GITHUB_REPO}`

export const DEFAULT_GITHUB_REPO = 'kilobyteno/dagr'
export const RELEASES_PAGE_URL = `https://github.com/${DEFAULT_GITHUB_REPO}/releases`

export type LatestRelease = {
  version: string
  tagName: string
  htmlUrl: string
  macDmgUrl: string | null
  windowsExeUrl: string | null
}

type GitHubAsset = {
  name?: string
  browser_download_url?: string
}

type GitHubRelease = {
  tag_name?: string
  html_url?: string
  assets?: GitHubAsset[]
}

function githubRepo() {
  const raw = process.env.GITHUB_REPO?.trim()
  return raw && raw.includes('/') ? raw : DEFAULT_GITHUB_REPO
}

function pickAsset(assets: GitHubAsset[], suffix: string) {
  const match = assets.find((asset) => {
    const name = asset.name?.toLowerCase() ?? ''
    return name.endsWith(suffix) && !name.endsWith('.blockmap')
  })
  return match?.browser_download_url ?? null
}

function versionFromTag(tagName: string) {
  return tagName.trim().replace(/^v/i, '')
}

export async function fetchLatestRelease(): Promise<LatestRelease | null> {
  const repo = githubRepo()
  const headers: Record<string, string> = {
    Accept: 'application/vnd.github+json',
    'User-Agent': 'dagr-website',
    'X-GitHub-Api-Version': '2022-11-28',
  }
  const token = process.env.GITHUB_TOKEN?.trim()
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(
    `https://api.github.com/repos/${repo}/releases/latest`,
    {
      headers,
      next: { revalidate: 300 },
    },
  )

  if (response.status === 404) return null
  if (!response.ok) {
    throw new Error(`GitHub releases returned ${response.status}`)
  }

  const payload = (await response.json()) as GitHubRelease
  const tagName = payload.tag_name?.trim()
  if (!tagName) return null

  const assets = payload.assets ?? []
  return {
    version: versionFromTag(tagName),
    tagName,
    htmlUrl: payload.html_url?.trim() || `https://github.com/${repo}/releases/latest`,
    macDmgUrl: pickAsset(assets, '.dmg'),
    windowsExeUrl: pickAsset(assets, '.exe'),
  }
}

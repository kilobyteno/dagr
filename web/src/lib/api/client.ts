export type ApiErrorBody = {
  error: {
    code: string
    message: string
  }
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

/** True when the client could not reach the API at all. */
export function isNetworkError(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    (error.status === 0 || error.code === 'network_error')
  )
}

/** True when the API is unreachable or returning a gateway/unavailable status. */
export function isServerUnavailable(error: unknown): boolean {
  if (!(error instanceof ApiError)) return false
  if (isNetworkError(error)) return true
  return (
    error.status === 502 ||
    error.status === 503 ||
    error.status === 504 ||
    error.code === 'server_unavailable'
  )
}

type ApiFetchOptions = {
  method?: string
  token?: string
  body?: unknown
  signal?: AbortSignal
}

function normaliseBase(serverUrl: string): string {
  return serverUrl.trim().replace(/\/$/, '')
}

export async function apiFetch<T>(
  serverUrl: string,
  path: string,
  options: ApiFetchOptions = {},
): Promise<T> {
  const base = normaliseBase(serverUrl)
  const headers: Record<string, string> = {
    Accept: 'application/json',
  }
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
  }
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`
  }

  let response: Response
  try {
    response = await fetch(`${base}${path}`, {
      method: options.method ?? (options.body !== undefined ? 'POST' : 'GET'),
      headers,
      body: options.body !== undefined ? JSON.stringify(options.body) : undefined,
      signal: options.signal,
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Could not reach the Dagr server')
  }

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text) as unknown
    } catch {
      data = null
    }
  }

  if (!response.ok) {
    const err = data as ApiErrorBody | null
    throw new ApiError(
      response.status,
      err?.error?.code ?? 'request_failed',
      err?.error?.message ?? `Request failed (${response.status})`,
    )
  }

  return data as T
}

type ApiUploadOptions = {
  method?: string
  token?: string
  formData: FormData
  signal?: AbortSignal
}

/** Multipart upload helper (does not set Content-Type; the browser sets the boundary). */
export async function apiUpload<T>(
  serverUrl: string,
  path: string,
  options: ApiUploadOptions,
): Promise<T> {
  const base = normaliseBase(serverUrl)
  const headers: Record<string, string> = {
    Accept: 'application/json',
  }
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`
  }

  let response: Response
  try {
    response = await fetch(`${base}${path}`, {
      method: options.method ?? 'PUT',
      headers,
      body: options.formData,
      signal: options.signal,
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Could not reach the Dagr server')
  }

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text) as unknown
    } catch {
      data = null
    }
  }

  if (!response.ok) {
    const err = data as ApiErrorBody | null
    throw new ApiError(
      response.status,
      err?.error?.code ?? 'request_failed',
      err?.error?.message ?? `Request failed (${response.status})`,
    )
  }

  return data as T
}

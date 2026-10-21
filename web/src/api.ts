import type { Snapshot, SourceResponse } from './types'

interface APIErrorPayload {
  error?: { message?: string }
}

async function request<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, init)
  if (!response.ok) {
    let message = `Request failed with status ${response.status}`
    try {
      const payload = (await response.json()) as APIErrorPayload
      message = payload.error?.message || message
    } catch {
      // The HTTP status still provides a useful fallback.
    }
    throw new Error(message)
  }
  return response.json() as Promise<T>
}

export function createScan(path: string): Promise<Snapshot> {
  return request<Snapshot>('/api/v1/scans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ path }),
  })
}

export function readSource(scanId: string, file: string): Promise<SourceResponse> {
  const query = new URLSearchParams({ file })
  return request<SourceResponse>(`/api/v1/scans/${encodeURIComponent(scanId)}/source?${query}`)
}

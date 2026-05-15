import { CreateDropResponse, Drop, WebhookEvent, EventPreview, EventDetail, EventsResponse } from './types'

const API_BASE = process.env.NEXT_PUBLIC_API_URL

function authHeaders(token: string): HeadersInit {
  return { Authorization: `Bearer ${token}` }
}

export async function createDrop(): Promise<CreateDropResponse> {
  const res = await fetch(`${API_BASE}/api/drops`, { method: 'POST' })
  if (!res.ok) throw new Error('Failed to create drop')
  return res.json()
}

export async function getDrop(dropUrl: string, token: string): Promise<Drop> {
  const res = await fetch(`${API_BASE}/api/drops/${dropUrl}`, {
    headers: authHeaders(token),
  })
  if (res.status === 401 || res.status === 404) throw new Error('SESSION_EXPIRED')
  if (!res.ok) throw new Error('Failed to fetch drop')
  return res.json()
}

export async function getEvents(
  dropUrl: string,
  token: string,
  page = 1,
  limit = 20
): Promise<EventsResponse> {
  const res = await fetch(
    `${API_BASE}/api/drops/${dropUrl}/events?page=${page}&limit=${limit}`,
    { headers: authHeaders(token) }
  )
  if (!res.ok) throw new Error('Failed to fetch events')
  return res.json()
}

export async function getEvent(
  dropUrl: string,
  eventId: string,
  token: string
): Promise<EventDetail> {
  const res = await fetch(`${API_BASE}/api/drops/${dropUrl}/events/${eventId}`, {
    headers: authHeaders(token),
  })
  if (!res.ok) throw new Error('Failed to fetch event')
  return res.json()
}

export async function deleteDrop(dropUrl: string, token: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api/drops/${dropUrl}`, {
    method: 'DELETE',
    headers: authHeaders(token),
  })
  if (!res.ok) throw new Error('Failed to delete drop')
}

type SSEFrame =
  | { type: 'connected' }
  | { type: 'webhook_received'; event: WebhookEvent }
  | { type: 'drop_expired' }

export function openEventStream(
  dropUrl: string,
  token: string,
  onEvent: (event: WebhookEvent) => void,
  onExpired?: () => void
): () => void {
  let cancelled = false

  async function connect() {
    try {
      const res = await fetch(`/api/stream/${dropUrl}`, {
        headers: { Authorization: `Bearer ${token}` },
      })

      if (!res.ok || !res.body) {
        // unauthorized or drop gone — surface expiry and stop
        onExpired?.()
        return
      }

      const reader = res.body.pipeThrough(new TextDecoderStream()).getReader()
      let buffer = ''

      while (!cancelled) {
        const { value, done } = await reader.read()
        if (done) break

        buffer += value
        const parts = buffer.split('\n\n')
        buffer = parts.pop() ?? ''

        for (const part of parts) {
          const dataLine = part.split('\n').find(l => l.startsWith('data:'))
          if (!dataLine) continue // keepalive comment line
          try {
            const frame = JSON.parse(dataLine.slice(5).trim()) as SSEFrame
            if (frame.type === 'webhook_received') onEvent(frame.event)
            if (frame.type === 'drop_expired') onExpired?.()
          } catch {
            // ignore malformed frames
          }
        }
      }
    } catch {
      // network error — retry after 3 s unless cancelled
      if (!cancelled) setTimeout(connect, 3000)
    }
  }

  connect()
  return () => { cancelled = true }
}

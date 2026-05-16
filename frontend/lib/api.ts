import { CreateDropResponse, Drop, WebhookEvent, EventPreview, EventDetail, EventsResponse } from './types'

const API_BASE = process.env.NEXT_PUBLIC_API_URL

function authHeaders(token: string): HeadersInit {
  return { Authorization: `Bearer ${token}` }
}

async function throwDropError(res: Response): Promise<never> {
  if (res.status === 404) throw new Error('DROP_NOT_FOUND')
  if (res.status === 401) {
    const body = await res.json()
    if (body.error === 'empty_token') throw new Error('TOKEN_EMPTY')
    throw new Error('TOKEN_EXPIRED')
  }
  throw new Error(`DROP_ERROR_${res.status}`)
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
  if (!res.ok) return throwDropError(res)
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
  onExpired?: () => void,
  onDisconnect?: () => void,
  onConnected?: () => void,
): () => void {
  let cancelled = false

  async function connect() {
    try {
      const res = await fetch(`/api/stream/${dropUrl}`, {
        headers: { Authorization: `Bearer ${token}` },
      })

      if (!res.ok || !res.body) {
        if (!cancelled) {
          // 401 means the token is invalid or the drop is gone — terminal
          res.status === 401 ? onExpired?.() : onDisconnect?.()
        }
        return
      }

      const reader = res.body.pipeThrough(new TextDecoderStream()).getReader()
      let buffer = ''
      let didExpire = false

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
            if (frame.type === 'connected') onConnected?.()
            if (frame.type === 'webhook_received') onEvent(frame.event)
            if (frame.type === 'drop_expired') {
              didExpire = true
              onExpired?.()
            }
          } catch {
            // ignore malformed frames
          }
        }
      }

      // stream closed without an explicit expiry event — recoverable disconnect
      if (!cancelled && !didExpire) onDisconnect?.()
    } catch {
      if (!cancelled) onDisconnect?.()
    }
  }

  connect()
  return () => { cancelled = true }
}

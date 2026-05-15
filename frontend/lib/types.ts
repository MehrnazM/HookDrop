// Returned by POST /api/drops only
export interface CreateDropResponse {
  url_slug: string
  session_token: string
  expires_at: string
}

// Returned by GET /api/drops/:drop_slug
export interface Drop {
  url_slug: string
  created_at: string
  expires_at: string
  event_count: number
}

interface WebhookEventMetadata {
  http_method: string 
  path: string 
  ip_address: string 
  received_at: string
}

interface WebhookEventPayload {
  headers: Record<string, string>
  query_params: Record<string, string>
  body: string | null
}

// SSE event shape — broadcast by processor, never stored with an id
export interface WebhookEvent {
  drop_slug: string
  metadata: WebhookEventMetadata
  payload: WebhookEventPayload
}

// Returned by GET /api/drops/:slug/events (list row)
export interface EventPreview {
  id: string
  http_method: string
  path: string
  received_at: string
  ip_address: string
  headers_preview: Record<string, string>
  body_preview: string
}

// Returned by GET /api/drops/:slug/events/:id (full detail)
export interface EventDetail {
  id: string
  http_method: string
  path: string
  headers: Record<string, string>
  query_params: Record<string, string>
  body: unknown
  received_at: string
  ip_address: string
  response_status: number | null
  response_body: unknown
}

export interface EventsResponse {
  events: EventPreview[]
  total_count: number
  page: number
  limit: number
}

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'

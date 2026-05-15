const SSE_BASE = process.env.NEXT_PUBLIC_SSE_URL ?? 'http://localhost:8082'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ slug: string }> }
) {
  const { slug } = await params
  const auth = request.headers.get('Authorization') ?? ''

  const upstream = await fetch(`${SSE_BASE}/api/drops/${slug}/stream`, {
    headers: { Authorization: auth },
  })

  if (!upstream.ok || !upstream.body) {
    return new Response('Unauthorized', { status: 401 })
  }

  return new Response(upstream.body, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
      'X-Accel-Buffering': 'no',
    },
  })
}

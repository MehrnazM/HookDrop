'use client'

import { useEffect, useRef, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { getDrop, getEvents, getEvent, deleteDrop, openEventStream } from '@/lib/api'
import { Drop, EventPreview, EventDetail } from '@/lib/types'
import RequestList from '@/components/RequestList'
import RequestDetail from '@/components/RequestDetail'
import TTLCountdown from '@/components/TTLCountdown'
import styles from './page.module.css'

const INGESTION_BASE = process.env.NEXT_PUBLIC_INGESTION_URL ?? 'http://localhost:8080'

export default function DropPage() {
  const params = useParams<{ drop_id: string }>()
  const router = useRouter()
  const dropSlug = params?.drop_id ?? ''

  const [drop, setDrop] = useState<Drop | null>(null)
  const [events, setEvents] = useState<EventPreview[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [detail, setDetail] = useState<EventDetail | null>(null)
  const [newIds, setNewIds] = useState<Set<string>>(new Set())
  const [sessionError, setSessionError] = useState(false)
  const [loading, setLoading] = useState(true)
  const [detailLoading, setDetailLoading] = useState(false)
  const [curlCopied, setCurlCopied] = useState(false)
  const [urlCopied, setUrlCopied] = useState(false)

  const tokenRef = useRef<string>('')
  const cancelSSERef = useRef<(() => void) | null>(null)

  const webhookUrl = `${INGESTION_BASE}/drop/${dropSlug}`

  useEffect(() => {
    if (!dropSlug) return

    const token = localStorage.getItem(`token:${dropSlug}`) ?? ''
    if (!token) {
      setSessionError(true)
      setLoading(false)
      return
    }
    tokenRef.current = token

    async function init() {
      try {
        const [dropData, eventsData] = await Promise.all([
          getDrop(dropSlug, token),
          getEvents(dropSlug, token, 1, 20),
        ])
        setDrop(dropData)
        setEvents(eventsData.events)
      } catch (err: unknown) {
        if (err instanceof Error && err.message === 'SESSION_EXPIRED') {
          setSessionError(true)
        }
      } finally {
        setLoading(false)
      }
    }

    init()
  }, [dropSlug])

  useEffect(() => {
    if (loading || sessionError || !dropSlug) return

    const token = tokenRef.current
    const cancel = openEventStream(
      dropSlug,
      token,
      async () => {
        try {
          const fresh = await getEvents(dropSlug, token, 1, 20)
          setEvents(prev => {
            const existingIds = new Set(prev.map(e => e.id))
            const incoming = fresh.events.filter(e => !existingIds.has(e.id))
            if (incoming.length === 0) return prev
            setNewIds(ids => {
              const next = new Set(ids)
              incoming.forEach(e => next.add(e.id))
              setTimeout(() => setNewIds(cur => {
                const cleaned = new Set(cur)
                incoming.forEach(e => cleaned.delete(e.id))
                return cleaned
              }), 600)
              return next
            })
            return [...incoming, ...prev]
          })
        } catch {
          // non-fatal
        }
      },
      () => setSessionError(true),
    )

    cancelSSERef.current = cancel
    return () => cancel()
  }, [loading, sessionError, dropSlug])

  async function handleSelect(id: string) {
    if (id === selectedId) return
    setSelectedId(id)
    setDetail(null)
    setDetailLoading(true)
    try {
      const d = await getEvent(dropSlug, id, tokenRef.current)
      setDetail(d)
    } catch {
      // keep detail null
    } finally {
      setDetailLoading(false)
    }
  }

  async function handleClear() {
    if (!confirm('Delete this drop and all its requests?')) return
    try {
      await deleteDrop(dropSlug, tokenRef.current)
      localStorage.removeItem(`token:${dropSlug}`)
      cancelSSERef.current?.()
      router.push('/')
    } catch {
      // non-fatal
    }
  }

  function handleCopyUrl() {
    navigator.clipboard.writeText(webhookUrl)
    setUrlCopied(true)
    setTimeout(() => setUrlCopied(false), 2000)
  }

  function handleCopyCurl() {
    if (!detail) return
    const headerFlags = Object.entries(detail.headers)
      .map(([k, v]) => `-H "${k}: ${v}"`)
      .join(' \\\n  ')
    const bodyFlag = detail.body ? `-d '${JSON.stringify(detail.body)}'` : ''
    const curl = [
      `curl -X ${detail.http_method}`,
      `  "${webhookUrl}"`,
      headerFlags ? `  ${headerFlags}` : '',
      bodyFlag     ? `  ${bodyFlag}`   : '',
    ].filter(Boolean).join(' \\\n')

    navigator.clipboard.writeText(curl)
    setCurlCopied(true)
    setTimeout(() => setCurlCopied(false), 2000)
  }

  if (loading) {
    return (
      <div className={styles.centered}>
        <p className={styles.muted}>Loading…</p>
      </div>
    )
  }

  if (sessionError) {
    return (
      <div className={styles.centered}>
        <p className={styles.sessionErrorTitle}>Session expired or drop not found.</p>
        <button onClick={() => router.push('/')} className={styles.backBtn}>
          Create a new drop
        </button>
      </div>
    )
  }

  return (
    <div className={styles.page}>
      <div className={styles.topbar}>
        <span className={styles.logo}>
          webhook<span className={styles.logoAccent}>x</span>
        </span>

        <div className={styles.urlBar}>
          <div className={styles.pulse} />
          <span className={styles.urlText}>{webhookUrl}</span>
          <button onClick={handleCopyUrl} className={styles.copyBtn}>
            {urlCopied ? 'Copied!' : 'Copy URL'}
          </button>
        </div>

        {drop && <TTLCountdown expiresAt={drop.expires_at} />}

        <button onClick={handleClear} className={styles.clearBtn}>
          Clear all
        </button>
      </div>

      <div className={styles.main}>
        <div className={styles.listPanel}>
          <div className={styles.panelHeader}>
            <span className={styles.panelTitle}>Requests</span>
            <span className={styles.countBadge}>{events.length}</span>
          </div>
          <div className={styles.colLabels}>
            <span className={styles.colLbl}>Method</span>
            <span className={styles.colLbl}>Path · Time</span>
            <span className={`${styles.colLbl} ${styles.colLblRight}`}>Status</span>
          </div>
          <RequestList
            events={events}
            selectedId={selectedId}
            onSelect={handleSelect}
            newIds={newIds}
          />
        </div>

        <div className={styles.detailPanel}>
          {detailLoading && (
            <div className={styles.centered}>
              <p className={styles.muted}>Loading…</p>
            </div>
          )}
          {!detailLoading && detail && (
            <RequestDetail
              event={detail}
              dropSlug={dropSlug}
              onCopyCurl={handleCopyCurl}
              curlCopied={curlCopied}
            />
          )}
          {!detailLoading && !detail && (
            <div className={styles.centered}>
              <p className={styles.muted}>Select a request to inspect it.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

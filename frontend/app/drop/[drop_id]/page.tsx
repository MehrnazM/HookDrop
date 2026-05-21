'use client'

import { useEffect, useRef, useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { getDrop, getEvents, getEvent, deleteDrop, openEventStream } from '@/lib/api'
import { Drop, EventPreview, EventDetail } from '@/lib/types'
import RequestList from '@/components/RequestList'
import RequestDetail from '@/components/RequestDetail'
import SkeletonDetail from '@/components/SkeletonDetail'
import TTLCountdown from '@/components/TTLCountdown'
import styles from './page.module.css'

const PAGE_LIMIT = 20

export default function DropPage() {
  const params = useParams<{ drop_id: string }>()
  const router = useRouter()
  const dropSlug = params?.drop_id ?? ''

  const [drop, setDrop] = useState<Drop | null>(null)
  const [events, setEvents] = useState<EventPreview[]>([])
  const [totalCount, setTotalCount] = useState(0)
  const [page, setPage] = useState(1)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [detail, setDetail] = useState<EventDetail | null>(null)
  const [newIds, setNewIds] = useState<Set<string>>(new Set())
  const [errorState, setErrorState] = useState<'no_token' | 'expired' | 'not_found' | null>(null)
  const [connectionState, setConnectionState] = useState<'connected' | 'reconnecting' | 'failed'>('connected')
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [curlCopied, setCurlCopied] = useState(false)
  const [urlCopied, setUrlCopied] = useState(false)
  const [latestEventAnnouncement, setLatestEventAnnouncement] = useState('')

  const tokenRef = useRef<string>('')
  const cancelSSERef = useRef<(() => void) | null>(null)

  const webhookUrl = typeof window !== 'undefined'
    ? `${window.location.protocol}//${window.location.host}/webhook/${dropSlug}`
    : `/webhook/${dropSlug}`
  const hasMore = events.length < totalCount

  // ── Initial load ────────────────────────────────────────────────
  useEffect(() => {
    if (!dropSlug) return
    const token = localStorage.getItem(`token:${dropSlug}`) ?? ''
    tokenRef.current = token

    async function init() {
      try {
        const dropData = await getDrop(dropSlug, token)
        const eventsData = await getEvents(dropSlug, token, 1, PAGE_LIMIT)
        setDrop(dropData)
        setEvents(eventsData.events)
        setTotalCount(eventsData.total_count)
        setPage(1)
      } catch (err: unknown) {
        if (err instanceof Error) {
          if (err.message === 'DROP_NOT_FOUND') {
            localStorage.removeItem('webhookx:lastSlug')
            setErrorState('not_found')
          } else if (err.message === 'TOKEN_EMPTY') setErrorState('no_token')
          else if (err.message === 'TOKEN_EXPIRED') {
            localStorage.removeItem('webhookx:lastSlug')
            setErrorState('expired')
          }
        }
      } finally {
        setLoading(false)
      }
    }
    init()
  }, [dropSlug])

  // ── SSE ──────────────────────────────────────────────────────────
  useEffect(() => {
    if (loading || errorState || !dropSlug) return
    const token = tokenRef.current
    let cancelled = false
    let retryCount = 0
    let retryTimerId: ReturnType<typeof setTimeout> | null = null
    const BACKOFF_MS = [1000, 2000, 4000, 8000, 8000]

    async function onEvent() {
      try {
        const fresh = await getEvents(dropSlug, token, 1, PAGE_LIMIT)
        setTotalCount(fresh.total_count)
        setEvents(prev => {
          const existingIds = new Set(prev.map(e => e.id))
          const incoming = fresh.events.filter(e => !existingIds.has(e.id))
          if (incoming.length === 0) return prev
          setLatestEventAnnouncement(`New ${incoming[0].http_method} request received`)
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
      } catch { /* non-fatal */ }
    }

    function onConnected() {
      retryCount = 0
      setConnectionState('connected')
    }

    function onExpired() {
      setErrorState(prev => prev ?? 'expired')
    }

    function onDisconnect() {
      if (cancelled) return
      if (retryCount >= 5) {
        setConnectionState('failed')
        return
      }
      setConnectionState('reconnecting')
      const delay = BACKOFF_MS[Math.min(retryCount, BACKOFF_MS.length - 1)]
      retryCount++
      retryTimerId = setTimeout(() => {
        if (cancelled) return
        cancelSSERef.current = openEventStream(dropSlug, token, onEvent, onExpired, onDisconnect, onConnected)
      }, delay)
    }

    cancelSSERef.current = openEventStream(dropSlug, token, onEvent, onExpired, onDisconnect, onConnected)

    return () => {
      cancelled = true
      cancelSSERef.current?.()
      if (retryTimerId !== null) clearTimeout(retryTimerId)
      setConnectionState('connected')
    }
  }, [loading, errorState, dropSlug])

  // ── Load more ────────────────────────────────────────────────────
  async function handleLoadMore() {
    if (loadingMore || !hasMore) return
    setLoadingMore(true)
    try {
      const nextPage = page + 1
      const data = await getEvents(dropSlug, tokenRef.current, nextPage, PAGE_LIMIT)
      setEvents(prev => [...prev, ...data.events])
      setTotalCount(data.total_count)
      setPage(nextPage)
    } catch { /* non-fatal */ } finally {
      setLoadingMore(false)
    }
  }

  // ── Select event ─────────────────────────────────────────────────
  async function handleSelect(id: string) {
    if (id === selectedId) return
    setSelectedId(id)
    setDetail(null)
    setDetailLoading(true)
    try {
      const d = await getEvent(dropSlug, id, tokenRef.current)
      setDetail(d)
    } catch { /* keep detail null */ } finally {
      setDetailLoading(false)
    }
  }

  function handleBack() {
    setSelectedId(null)
    setDetail(null)
  }

  // ── Clear / delete ───────────────────────────────────────────────
  async function handleClear() {
    if (!confirm('Delete this drop and all its requests?')) return
    try {
      await deleteDrop(dropSlug, tokenRef.current)
      localStorage.removeItem(`token:${dropSlug}`)
      localStorage.removeItem('webhookx:lastSlug')
      cancelSSERef.current?.()
      window.location.href = '/'
    } catch { /* non-fatal */ }
  }

  // ── Copy helpers ─────────────────────────────────────────────────
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

  // ── Error / loading states ───────────────────────────────────────
  if (loading) {
    return (
      <div className={styles.page}>
        <div className={styles.topbarSkeleton}>
          <div className={`shimmer ${styles.skLogo}`} />
          <div className={`shimmer ${styles.skUrlBar}`} />
          <div className={`shimmer ${styles.skBtn}`} />
          <div className={`shimmer ${styles.skBtn}`} />
        </div>
        <div className={styles.main}>
          <div className={styles.listPanel}>
            <div className={styles.panelHeader}>
              <div className={`shimmer ${styles.skTitle}`} />
            </div>
            <div className={styles.colLabels}>
              <span className={styles.colLbl}>Method</span>
              <span className={styles.colLbl}>Path · Time</span>
              <span className={`${styles.colLbl} ${styles.colLblRight}`}>Status</span>
            </div>
            <RequestList
              events={[]} selectedId={null} onSelect={() => {}} newIds={new Set()}
              loading={true} hasMore={false} loadingMore={false} onLoadMore={() => {}}
              webhookUrl=""
            />
          </div>
          <div className={styles.detailPanel}>
            <SkeletonDetail />
          </div>
        </div>
      </div>
    )
  }

  if (errorState) {
    const errorConfig = {
      no_token: {
        title: 'Wrong browser or private window',
        message: 'This Drop can only be viewed in the browser where you created it. Your session token is stored there.',
      },
      expired: {
        title: 'Drop expired',
        message: 'This Drop existed but your 24-hour session has ended. All events have been deleted.',
      },
      not_found: {
        title: 'Drop not found',
        message: 'This URL doesn\'t match any active Drop.',
      },
    }
    const error = errorConfig[errorState]
    return (
      <div className={styles.centered}>
        <p className={styles.sessionErrorTitle}>{error.title}</p>
        <p className={styles.sessionErrorMessage}>{error.message}</p>
        <button onClick={() => { window.location.href = '/' }} className={styles.backBtn}>
          Create a new drop
        </button>
      </div>
    )
  }

  // ── Main layout ──────────────────────────────────────────────────
  const mobileShowDetail = !!selectedId

  return (
    <div className={styles.page}>
      {/* Top bar */}
      <div className={styles.topbar}>
        <span className={styles.logo}>
          webhook<span className={styles.logoAccent}>x</span>
        </span>
        <div className={styles.urlBar}>
          <div className={styles.pulse} aria-hidden="true" />
          <span className={styles.urlText}>{webhookUrl}</span>
          <button onClick={handleCopyUrl} className={styles.copyBtn}>
            {urlCopied ? 'Copied!' : 'Copy URL'}
          </button>
        </div>
        {drop && <TTLCountdown expiresAt={drop.expires_at} />}
        <button onClick={handleClear} className={styles.clearBtn} aria-label="Clear all events">Clear all</button>
      </div>

      {/* Reconnecting banner */}
      {connectionState !== 'connected' && (
        <div className={styles.reconnectBanner}>
          {connectionState === 'reconnecting' ? (
            <>
              <span>&#9888; Connection lost &mdash; reconnecting</span>
              <span className={styles.dot1}>.</span>
              <span className={styles.dot2}>.</span>
              <span className={styles.dot3}>.</span>
            </>
          ) : (
            <>
              <span>Connection lost &mdash; refresh to reconnect.</span>
              <button className={styles.refreshBtn} onClick={() => window.location.reload()}>
                Refresh
              </button>
            </>
          )}
        </div>
      )}

      {/* Two-panel layout */}
      <div className={`${styles.main} ${mobileShowDetail ? styles.mobileDetail : ''}`}>

        {/* Left: request list */}
        <div className={styles.listPanel}>
          <div className={styles.panelHeader}>
            <span className={styles.panelTitle}>Requests</span>
            <span className={styles.countBadge}>{totalCount}</span>
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
            loading={false}
            hasMore={hasMore}
            loadingMore={loadingMore}
            onLoadMore={handleLoadMore}
            webhookUrl={webhookUrl}
          />
        </div>

        {/* Right: request detail */}
        <div className={styles.detailPanel}>
          {detailLoading && <SkeletonDetail />}
          {!detailLoading && detail && (
            <RequestDetail
              event={detail}
              dropSlug={dropSlug}
              onCopyCurl={handleCopyCurl}
              curlCopied={curlCopied}
              onBack={handleBack}
            />
          )}
          {!detailLoading && !detail && (
            <div className={styles.centered}>
              <svg className={styles.emptyDetailIcon} width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="22 12 16 12 14 15 10 15 8 12 2 12"/>
                <path d="M5.45 5.11L2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11z"/>
              </svg>
              <p className={styles.emptyDetailTitle}>No request selected</p>
              <p className={styles.emptyDetailSubtext}>Click any request on the left to inspect its headers, body, and query params.</p>
            </div>
          )}
        </div>

      </div>

      {/* Screen-reader live region for incoming events */}
      <div aria-live="polite" aria-atomic="false" className="sr-only">
        {latestEventAnnouncement}
      </div>
    </div>
  )
}

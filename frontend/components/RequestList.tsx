'use client'

import { useRef, useState } from 'react'
import MethodBadge from './MethodBadge'
import StatusBadge from './StatusBadge'
import SkeletonRow from './SkeletonRow'
import styles from './RequestList.module.css'
import { EventPreview } from '@/lib/types'

type Props = {
  events: EventPreview[]
  selectedId: string | null
  onSelect: (id: string) => void
  newIds: Set<string>
  loading: boolean
  hasMore: boolean
  loadingMore: boolean
  onLoadMore: () => void
  webhookUrl: string
}

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime()
  const s = Math.floor(diff / 1000)
  if (s < 5)  return 'just now'
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ago`
}

export default function RequestList({
  events, selectedId, onSelect, newIds,
  loading, hasMore, loadingMore, onLoadMore, webhookUrl,
}: Props) {
  const [curlCopied, setCurlCopied] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)

  function focusItem(index: number) {
    const items = listRef.current?.querySelectorAll<HTMLElement>('[role="option"]')
    if (!items) return
    const target = items[Math.max(0, Math.min(index, items.length - 1))]
    target?.focus()
  }

  if (loading) {
    return (
      <div className={styles.list}>
        {Array.from({ length: 6 }).map((_, i) => <SkeletonRow key={i} />)}
      </div>
    )
  }

  if (events.length === 0) {
    const curlCommand = `curl -X POST ${webhookUrl} \\\n  -H "Content-Type: application/json" \\\n  -d '{"hello": "webhookx"}'`

    return (
      <div className={styles.empty}>
        <div className={styles.emptyPulse} />
        <p className={styles.emptyHeading}>Waiting for your first request</p>
        <p className={styles.emptySubtext}>
          Send any HTTP request to your drop URL to see it appear here instantly.
        </p>
        <div className={styles.curlBlock}>
          <pre className={styles.curlCode}>{curlCommand}</pre>
          <button
            className={styles.curlCopyBtn}
            onClick={() => {
              navigator.clipboard.writeText(curlCommand)
              setCurlCopied(true)
              setTimeout(() => setCurlCopied(false), 2000)
            }}
          >
            {curlCopied ? 'Copied!' : 'Copy'}
          </button>
        </div>
      </div>
    )
  }

  return (
    <>
      <div
        ref={listRef}
        className={styles.list}
        role="listbox"
        aria-label="Webhook events"
        aria-orientation="vertical"
      >
        {events.map((event, index) => {
          const isActive = event.id === selectedId
          const isNew = newIds.has(event.id)
          // Roving tabIndex: selected item (or first item when nothing selected) is reachable via Tab
          const isTabStop = selectedId ? isActive : index === 0
          const rowClass = [
            styles.row,
            isActive ? styles.active : '',
            isNew ? 'slide-in' : '',
          ].join(' ')

          return (
            <div
              key={event.id}
              role="option"
              aria-selected={isActive}
              tabIndex={isTabStop ? 0 : -1}
              className={rowClass}
              onClick={() => onSelect(event.id)}
              onKeyDown={(e) => {
                if (e.key === 'ArrowDown') {
                  e.preventDefault()
                  focusItem(index + 1)
                } else if (e.key === 'ArrowUp') {
                  e.preventDefault()
                  focusItem(index - 1)
                } else if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  onSelect(event.id)
                }
              }}
            >
              <MethodBadge method={event.http_method} />
              <div className={styles.meta}>
                <div className={styles.path}>{event.path || '/'}</div>
                <div className={styles.time}>{relativeTime(event.received_at)}</div>
              </div>
              <StatusBadge status={200} />
            </div>
          )
        })}
      </div>

      {hasMore && (
        <div className={styles.loadMore}>
          <button
            className={styles.loadMoreBtn}
            onClick={onLoadMore}
            disabled={loadingMore}
          >
            {loadingMore ? 'Loading…' : 'Load more'}
          </button>
        </div>
      )}
    </>
  )
}

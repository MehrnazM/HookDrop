'use client'

import MethodBadge from './MethodBadge'
import StatusBadge from './StatusBadge'
import styles from './RequestList.module.css'
import { EventPreview } from '@/lib/types'

type Props = {
  events: EventPreview[]
  selectedId: string | null
  onSelect: (id: string) => void
  newIds: Set<string>
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

export default function RequestList({ events, selectedId, onSelect, newIds }: Props) {
  if (events.length === 0) {
    return (
      <div className={styles.empty}>
        <p className={styles.emptyText}>Waiting for requests…</p>
        <p className={styles.emptyHint}>Send a webhook to your drop URL to see it here.</p>
      </div>
    )
  }

  return (
    <div className={styles.list}>
      {events.map((event) => {
        const isActive = event.id === selectedId
        const isNew = newIds.has(event.id)
        const rowClass = [
          styles.row,
          isActive ? styles.active : '',
          isNew ? 'slide-in' : '',
        ].join(' ')

        return (
          <div key={event.id} onClick={() => onSelect(event.id)} className={rowClass}>
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
  )
}

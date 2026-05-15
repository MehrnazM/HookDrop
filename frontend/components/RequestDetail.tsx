'use client'

import { useState } from 'react'
import MethodBadge from './MethodBadge'
import StatusBadge from './StatusBadge'
import JsonHighlight from './JsonHighlight'
import styles from './RequestDetail.module.css'
import { EventDetail } from '@/lib/types'

type Tab = 'body' | 'headers' | 'query' | 'response'

type Props = {
  event: EventDetail
  dropSlug: string
  onCopyCurl: () => void
  curlCopied: boolean
  onBack: () => void
}

function KVTable({ data }: { data: Record<string, string> }) {
  const entries = Object.entries(data)
  if (entries.length === 0) {
    return <p className={styles.emptyState}>None</p>
  }
  return (
    <div>
      {entries.map(([k, v]) => (
        <div key={k} className={styles.kvRow}>
          <span className={styles.kvKey}>{k}</span>
          <span className={styles.kvVal}>{v}</span>
        </div>
      ))}
    </div>
  )
}

export default function RequestDetail({ event, onCopyCurl, curlCopied, onBack }: Props) {
  const [tab, setTab] = useState<Tab>('body')

  const ts = new Date(event.received_at).toUTCString()

  return (
    <div className={styles.panel}>
      <div className={styles.header}>
        <button className={styles.backBtn} onClick={onBack}>← Back</button>
        <MethodBadge method={event.http_method} />
        <StatusBadge status={event.response_status ?? 200} />
        <span className={styles.ts}>{ts}</span>
        <button onClick={onCopyCurl} className={styles.actionBtn}>
          {curlCopied ? 'Copied!' : 'Copy as cURL'}
        </button>
      </div>

      <div className={styles.tabs}>
        {(['body', 'headers', 'query', 'response'] as Tab[]).map((t) => {
          const isActive = tab === t
          const tabClass = [
            styles.tab,
            isActive ? (t === 'response' ? styles.tabActiveResp : styles.tabActive) : '',
          ].join(' ')
          return (
            <button key={t} onClick={() => setTab(t)} className={tabClass}>
              {t === 'body' ? 'Request Body'
                : t === 'headers' ? 'Request Headers'
                : t === 'query' ? 'Query Params'
                : 'Our Response'}
            </button>
          )
        })}
      </div>

      <div className={styles.body}>
        {tab === 'body' && <JsonHighlight value={event.body} />}

        {tab === 'headers' && <KVTable data={event.headers} />}

        {tab === 'query' && (
          Object.keys(event.query_params).length === 0
            ? <p className={styles.emptyState}>No query parameters</p>
            : <KVTable data={event.query_params} />
        )}

        {tab === 'response' && (
          <div>
            <div className={styles.responseStatus}>
              <StatusBadge status={event.response_status ?? 200} />
              <span className={styles.responseStatusLabel}>
                {event.response_status ?? 200} OK
              </span>
            </div>
            <p className={styles.emptyState} style={{ marginTop: '12px' }}>No response body</p>
          </div>
        )}
      </div>
    </div>
  )
}

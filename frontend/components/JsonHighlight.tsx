'use client'

import styles from './JsonHighlight.module.css'

type Props = { value: unknown }

function highlight(value: unknown): React.ReactNode {
  if (value === null) {
    return <span className={styles.jMuted}>null</span>
  }
  if (typeof value === 'boolean') {
    return <span className={styles.jBool}>{String(value)}</span>
  }
  if (typeof value === 'number') {
    return <span className={styles.jNum}>{value}</span>
  }
  if (typeof value === 'string') {
    return <span className={styles.jStr}>"{value}"</span>
  }
  if (Array.isArray(value)) {
    if (value.length === 0) return <span className={styles.jMuted}>{'[]'}</span>
    return (
      <>
        <span className={styles.jMuted}>{'['}</span>
        {value.map((item, i) => (
          <div key={i} className={styles.indent}>
            {highlight(item)}{i < value.length - 1 ? <span className={styles.jMuted}>,</span> : null}
          </div>
        ))}
        <div><span className={styles.jMuted}>{']'}</span></div>
      </>
    )
  }
  if (typeof value === 'object') {
    const entries = Object.entries(value as Record<string, unknown>)
    if (entries.length === 0) return <span className={styles.jMuted}>{'{}'}</span>
    return (
      <>
        <span className={styles.jMuted}>{'{'}</span>
        {entries.map(([k, v], i) => (
          <div key={k} className={styles.indent}>
            <span className={styles.jKey}>"{k}"</span>
            <span className={styles.jMuted}>: </span>
            {highlight(v)}
            {i < entries.length - 1 ? <span className={styles.jMuted}>,</span> : null}
          </div>
        ))}
        <div><span className={styles.jMuted}>{'}'}</span></div>
      </>
    )
  }
  return <span className={styles.jMuted}>{String(value)}</span>
}

export default function JsonHighlight({ value }: Props) {
  if (value === null || value === undefined) {
    return <p className={styles.noBody}>No body</p>
  }

  if (typeof value === 'object' && '_raw' in (value as object)) {
    const raw = (value as { _raw: string })._raw
    return (
      <div className={styles.codeBlock}>
        <p className={styles.rawWarn}>⚠ Non-JSON payload</p>
        <span className={styles.rawText}>{raw}</span>
      </div>
    )
  }

  return (
    <div className={styles.codeBlock}>
      {highlight(value)}
    </div>
  )
}

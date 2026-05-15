import styles from './StatusBadge.module.css'

type Props = { status: number | null }

export default function StatusBadge({ status }: Props) {
  if (status === null) return null

  const cls = status >= 200 && status < 300 ? styles.s2xx
            : status >= 400 && status < 500 ? styles.s4xx
            : status >= 500                 ? styles.s5xx
            : styles.def

  return (
    <span className={`${styles.badge} ${cls}`}>
      {status}
    </span>
  )
}

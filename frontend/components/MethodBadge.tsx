import styles from './MethodBadge.module.css'

type Props = { method: string }

const classMap: Record<string, string> = {
  GET: styles.GET, POST: styles.POST, PUT: styles.PUT,
  PATCH: styles.PATCH, DELETE: styles.DELETE,
}

export default function MethodBadge({ method }: Props) {
  const m = method.toUpperCase()
  const label = m === 'DELETE' ? 'DEL' : m
  return (
    <span className={`${styles.badge} ${classMap[m] ?? styles.DEF}`}>
      {label}
    </span>
  )
}

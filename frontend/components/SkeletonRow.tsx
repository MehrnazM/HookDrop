import styles from './SkeletonRow.module.css'

export default function SkeletonRow() {
  return (
    <div className={styles.row}>
      <div className={`shimmer ${styles.method}`} />
      <div className={styles.meta}>
        <div className={`shimmer ${styles.path}`} />
        <div className={`shimmer ${styles.time}`} />
      </div>
      <div className={`shimmer ${styles.status}`} />
    </div>
  )
}

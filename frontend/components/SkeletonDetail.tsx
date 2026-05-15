import styles from './SkeletonDetail.module.css'

const LINE_WIDTHS = ['72%', '55%', '88%', '40%', '65%', '78%', '50%', '83%']

export default function SkeletonDetail() {
  return (
    <div className={styles.panel}>
      <div className={styles.header}>
        <div className={`shimmer ${styles.pill}`} style={{ width: 44 }} />
        <div className={`shimmer ${styles.pill}`} style={{ width: 36 }} />
        <div className={`shimmer ${styles.tsBar}`} />
        <div className={`shimmer ${styles.pill}`} style={{ width: 90 }} />
      </div>

      <div className={styles.tabs}>
        {[100, 110, 90, 95].map((w, i) => (
          <div key={i} className={`shimmer ${styles.tab}`} style={{ width: w }} />
        ))}
      </div>

      <div className={styles.body}>
        {LINE_WIDTHS.map((w, i) => (
          <div key={i} className={`shimmer ${styles.line}`} style={{ width: w }} />
        ))}
      </div>
    </div>
  )
}

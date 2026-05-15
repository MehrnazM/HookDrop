'use client'

import { useEffect, useState } from 'react'
import styles from './TTLCountdown.module.css'

type Props = { expiresAt: string }

function formatTTL(ms: number): string {
  if (ms <= 0) return 'Expired'
  const totalSeconds = Math.floor(ms / 1000)
  const h = Math.floor(totalSeconds / 3600)
  const m = Math.floor((totalSeconds % 3600) / 60)
  if (h > 0) return `Expires in ${h}h ${m}m`
  return `Expires in ${m}m`
}

export default function TTLCountdown({ expiresAt }: Props) {
  const [label, setLabel] = useState('')

  useEffect(() => {
    function tick() {
      setLabel(formatTTL(new Date(expiresAt).getTime() - Date.now()))
    }
    tick()
    const id = setInterval(tick, 60_000)
    return () => clearInterval(id)
  }, [expiresAt])

  return <span className={styles.countdown}>{label}</span>
}

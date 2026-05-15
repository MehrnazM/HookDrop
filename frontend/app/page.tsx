'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { createDrop } from '@/lib/api'
import styles from './page.module.css'

export default function HomePage() {
  const router = useRouter()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleCreate() {
    setLoading(true)
    setError(null)
    try {
      const drop = await createDrop()
      localStorage.setItem(`token:${drop.url_slug}`, drop.session_token)
      router.push(`/drop/${drop.url_slug}`)
    } catch {
      setError('Failed to create a drop. Is the server running?')
      setLoading(false)
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.card}>
        <div className={styles.logo}>
          webhook<span className={styles.logoAccent}>x</span>
        </div>

        <p className={styles.subtitle}>
          Instant webhook inspection.<br />No setup required.
        </p>

        <button onClick={handleCreate} disabled={loading} className={styles.button}>
          {loading ? 'Creating…' : 'Create a Drop'}
        </button>

        {error && <p className={styles.error}>{error}</p>}
      </div>
    </div>
  )
}

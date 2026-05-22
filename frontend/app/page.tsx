'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { createDrop } from '@/lib/api'

// ── Live wire ────────────────────────────────────────────────
type Particle = { id: number; lane: number; phase: 'in' | 'out'; born: number }

// easeOut quad: starts fast, decelerates cleanly into the destination — no stalling before arrival
const easeOut = (t: number) => t * (2 - t)
// easeIn quad: starts slow, accelerates away — clean departure
const easeIn  = (t: number) => t * t

// Quadratic bezier matching lane paths: P0=(112,sy), P1=(290,sy), P2=(455,110)
// Particle must follow the SAME curve the SVG path draws — linear lerp would drift off
const lanePoint = (t: number, sy: number) => {
  const mt = 1 - t
  return {
    x: mt * mt * 112 + 2 * mt * t * 290 + t * t * 455,
    y: mt * mt * sy  + 2 * mt * t * sy  + t * t * 110,
  }
}

const IN_DUR   = 3200   // ms — how long a particle takes to reach HookDrop
const OUT_DUR  = 2600   // ms — how long a particle takes to reach inspector
const SPAWN_MS = 2000   // ms between new particles

function LiveWire() {
  const [particles, setParticles] = useState<Particle[]>([])
  const [now, setNow] = useState(() => Date.now())
  const idRef = useRef(0)

  // RAF loop — drives smooth particle positions without SVG SMIL quirks
  useEffect(() => {
    let raf: number
    const tick = () => { setNow(Date.now()); raf = requestAnimationFrame(tick) }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [])

  // Particle spawner — in → out → removed
  useEffect(() => {
    let alive = true
    const spawn = () => {
      if (!alive) return
      const lane = Math.floor(Math.random() * 3)
      const id = ++idRef.current
      setParticles(p => [...p, { id, lane, phase: 'in', born: Date.now() }])
      setTimeout(() => setParticles(p =>
        p.map(x => x.id === id ? { ...x, phase: 'out' as const, born: Date.now() } : x)
      ), IN_DUR)
      setTimeout(() => setParticles(p => p.filter(x => x.id !== id)), IN_DUR + OUT_DUR + 200)
    }
    const iv = setInterval(spawn, SPAWN_MS)
    spawn()
    return () => { alive = false; clearInterval(iv) }
  }, [])

  const laneY = [0.28, 0.50, 0.72]

  return (
    <div className="livewire">
      <span className="livewire-label lw-src s1">stripe.com</span>
      <span className="livewire-label lw-src s2">github.com</span>
      <span className="livewire-label lw-src s3">shopify.com</span>
      <span className="livewire-label lw-center">HookDrop</span>
      <span className="livewire-label lw-dst">your inspector</span>

      <svg className="livewire-svg" preserveAspectRatio="none" viewBox="0 0 1000 220">
        <defs>
          {/* userSpaceOnUse on BOTH gradients — objectBoundingBox breaks on zero-height
              paths (straight middle lane + horizontal output line) */}
          <linearGradient id="laneGrad" x1="112" y1="0" x2="455" y2="0" gradientUnits="userSpaceOnUse">
            <stop offset="0"   stopColor="rgba(91,110,245,0.10)" />
            <stop offset=".6"  stopColor="rgba(91,110,245,0.65)" />
            <stop offset="1"   stopColor="rgba(91,110,245,0.85)" />
          </linearGradient>
          <linearGradient id="laneGradOut" x1="545" y1="0" x2="862" y2="0" gradientUnits="userSpaceOnUse">
            <stop offset="0"   stopColor="rgba(91,110,245,0.80)" />
            <stop offset="1"   stopColor="rgba(34,197,94,0.55)" />
          </linearGradient>
        </defs>

        {/* Incoming lanes — top/bottom arc in; middle is straight horizontal */}
        {laneY.map((y, i) => (
          <path
            key={'lane' + i}
            d={`M 112 ${y * 220} Q 290 ${y * 220}, 455 110`}
            stroke="url(#laneGrad)"
            strokeWidth="1.5"
            fill="none"
          />
        ))}

        {/* Output lane — right edge of HookDrop → left edge of inspector */}
        <path d="M 545 110 L 862 110" stroke="url(#laneGradOut)" strokeWidth="1.5" fill="none" />

        {/* Positions driven by RAF — bezier for 'in', linear for 'out' */}
        {particles.map(p => {
          const sy = laneY[p.lane] * 220
          const elapsed = now - p.born

          if (p.phase === 'in') {
            // easeOut: decelerates smoothly into HookDrop, never stalls before arrival
            const t = easeOut(Math.min(elapsed / IN_DUR, 1))
            const { x: cx, y: cy } = lanePoint(t, sy)
            const opacity = Math.min(elapsed / 200, 1)
            return <circle key={p.id} r="3.5" cx={cx} cy={cy} opacity={opacity} className="lw-particle" />
          }

          // easeIn: accelerates away from HookDrop toward inspector
          const t = easeIn(Math.min(elapsed / OUT_DUR, 1))
          const cx = 545 + (860 - 545) * t
          const opacity = elapsed > OUT_DUR * 0.75
            ? Math.max(0, 1 - (elapsed - OUT_DUR * 0.75) / (OUT_DUR * 0.25))
            : 1
          return <circle key={p.id} r="3.5" cx={cx} cy={110} opacity={opacity} className="lw-particle out" />
        })}
      </svg>
    </div>
  )
}

// ── Live counter ─────────────────────────────────────────────
function LiveCounter() {
  const [n, setN] = useState(1247892)
  useEffect(() => {
    const iv = setInterval(() => {
      setN(v => v + Math.floor(Math.random() * 4) + 1)
    }, 1800)
    return () => clearInterval(iv)
  }, [])
  return (
    <div className="live-counter">
      <span className="live-dot" />
      <span><span className="num">{n.toLocaleString()}</span> webhooks captured this week</span>
    </div>
  )
}

// ── Demo reel ────────────────────────────────────────────────
function TypedURL() {
  const full = 'hookdrop.io/drop/k9p3mz'
  const [n, setN] = useState(0)
  useEffect(() => {
    setN(0)
    let i = 0
    const iv = setInterval(() => {
      i++
      setN(i)
      if (i >= full.length) clearInterval(iv)
    }, 75)
    return () => clearInterval(iv)
  }, [])
  return (
    <>
      <span>{full.slice(0, n)}</span>
      <span className="typed-cursor" />
    </>
  )
}

function DemoReel() {
  const STEPS = 4
  const STEP_MS = 5000
  const STEP_LABELS = ['Get URL', 'Paste', 'Watch', 'Inspect']
  const [step, setStep] = useState(0)
  const [progress, setProgress] = useState(0)
  const [paused, setPaused] = useState(false)
  const startRef = useRef(Date.now())

  useEffect(() => {
    if (paused) return
    let raf: number
    const tick = () => {
      const elapsed = Date.now() - startRef.current
      const p = Math.min(1, elapsed / STEP_MS)
      setProgress(p)
      if (p >= 1) {
        startRef.current = Date.now()
        setStep(s => (s + 1) % STEPS)
        setProgress(0)
      }
      raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => cancelAnimationFrame(raf)
  }, [paused])

  const jumpTo = (i: number) => {
    setStep(i)
    setProgress(0)
    startRef.current = Date.now()
  }

  const togglePause = () => setPaused(p => !p)

  return (
    <div className="reel">
      <div className="reel-stage">
        <div className={'reel-step' + (step === 0 ? ' active' : '')}>
          <div className="step-num">01 / GET YOUR URL</div>
          <h3>One click. A live URL. That&apos;s it.</h3>
          <div className="step-visual">
            <div className="url-card">
              <span className="live-dot" />
              <span className="url-string"><span className="muted">hookdrop.io/drop/</span>k9p3mz</span>
              <span className="copy-bubble">Copied!</span>
            </div>
          </div>
        </div>

        <div className={'reel-step' + (step === 1 ? ' active' : '')}>
          <div className="step-num">02 / PASTE INTO ANY SERVICE</div>
          <h3>Stripe, GitHub, Shopify, your own backend.</h3>
          <div className="step-visual">
            <div className="service-mock">
              <div className="svc-row">
                <div className="svc-logo">S</div>
                <span>Stripe · Add endpoint</span>
              </div>
              <div className="svc-field-label">Endpoint URL</div>
              <div className="svc-field">
                {step === 1
                  ? <TypedURL key={step} />
                  : <span style={{ color: 'var(--text-muted)' }}>hookdrop.io/drop/k9p3mz</span>
                }
              </div>
            </div>
          </div>
        </div>

        <div className={'reel-step' + (step === 2 ? ' active' : '')}>
          <div className="step-num">03 / WATCH IT ARRIVE</div>
          <h3>The moment a request hits, it&apos;s yours.</h3>
          <div className="step-visual">
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
              {step === 2 && (
                <div className="arriving-event" key={step}>
                  <div className="ev-top">
                    <span className="method-pill">POST</span>
                    <span className="status-pill">200</span>
                    <span className="ev-time">just now</span>
                  </div>
                  <div className="ev-path">/drop/k9p3mz</div>
                  <div className="ev-from">from <span className="src">stripe.com</span> · 421 bytes · 12ms latency</div>
                </div>
              )}
              <div className="check-row">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3">
                  <path d="M5 12l5 5L20 7" />
                </svg>
                Captured · 0 dropped · no config needed
              </div>
            </div>
          </div>
        </div>

        <div className={'reel-step' + (step === 3 ? ' active' : '')}>
          <div className="step-num">04 / INSPECT EVERY DETAIL</div>
          <h3>Headers. Body. Query. All of it.</h3>
          <div className="step-visual">
            <div className="inspect-mini">
              <div className="il">
                <div className="il-item active">
                  <span className="m-post">POST</span>
                  <span className="path">/drop/k9p3mz</span>
                  <span className="s2">200</span>
                </div>
                <div className="il-item">
                  <span className="m-post">POST</span>
                  <span className="path">/drop/k9p3mz</span>
                  <span className="s2">200</span>
                </div>
                <div className="il-item">
                  <span className="m-get">GET</span>
                  <span className="path">/drop/k9p3mz</span>
                  <span className="s2">200</span>
                </div>
              </div>
              <div className="ir">
                <span className="jm">{'{'}</span><br />
                <span className="indent"><span className="jk">&quot;type&quot;</span><span className="jm">:</span> <span className="js">&quot;payment_intent.succeeded&quot;</span><span className="jm">,</span></span><br />
                <span className="indent"><span className="jk">&quot;amount&quot;</span><span className="jm">:</span> <span className="jn">4999</span><span className="jm">,</span></span><br />
                <span className="indent"><span className="jk">&quot;currency&quot;</span><span className="jm">:</span> <span className="js">&quot;usd&quot;</span><span className="jm">,</span></span><br />
                <span className="indent"><span className="jk">&quot;customer&quot;</span><span className="jm">:</span> <span className="js">&quot;cus_Qm9TpXj2&quot;</span><span className="jm">,</span></span><br />
                <span className="indent"><span className="jk">&quot;status&quot;</span><span className="jm">:</span> <span className="js">&quot;succeeded&quot;</span></span><br />
                <span className="jm">{'}'}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="reel-progress">
        <button
          className="reel-play"
          onClick={togglePause}
          aria-label={paused ? 'Play' : 'Pause'}
          title={paused ? 'Play' : 'Pause'}
        >
          {paused ? (
            <svg viewBox="0 0 12 12" fill="currentColor"><polygon points="3,2 10,6 3,10" /></svg>
          ) : (
            <svg viewBox="0 0 12 12" fill="currentColor"><rect x="3" y="2" width="2.5" height="8" /><rect x="6.5" y="2" width="2.5" height="8" /></svg>
          )}
        </button>
        {STEP_LABELS.map((label, i) => (
          <button
            key={i}
            className={'reel-bar' + (i === step ? ' current' : '') + (i < step ? ' complete' : '')}
            onClick={() => jumpTo(i)}
          >
            <span className="reel-bar-label">
              <span className="num">0{i + 1}</span> {label}
            </span>
            <span className="reel-bar-track">
              <span className="fill" style={{
                width: i === step ? `${progress * 100}%` : (i < step ? '100%' : '0%')
              }} />
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}

// ── Reveal-on-scroll ─────────────────────────────────────────
function Reveal({ children, delay = 0 }: { children: React.ReactNode; delay?: number }) {
  const ref = useRef<HTMLDivElement>(null)
  const [shown, setShown] = useState(false)
  useEffect(() => {
    if (!ref.current) return
    const rect = ref.current.getBoundingClientRect()
    const inView = rect.top < window.innerHeight + 100 && rect.bottom > -100
    if (inView) {
      const t = setTimeout(() => setShown(true), Math.min(delay, 60))
      return () => clearTimeout(t)
    }
    const io = new IntersectionObserver(entries => {
      entries.forEach(e => {
        if (e.isIntersecting) {
          setTimeout(() => setShown(true), delay)
          io.disconnect()
        }
      })
    }, { threshold: 0, rootMargin: '0px 0px 100px 0px' })
    io.observe(ref.current)
    const fallback = setTimeout(() => setShown(true), 200)
    return () => { io.disconnect(); clearTimeout(fallback) }
  }, [delay])
  return (
    <div ref={ref} className={'reveal' + (shown ? ' in' : '')}>
      {children}
    </div>
  )
}

// ── Playground ───────────────────────────────────────────────
type Source = {
  id: string; name: string; initial: string; method: string
  path: string; body: Record<string, unknown>
}

const SOURCES: Source[] = [
  { id: 'stripe', name: 'Stripe', initial: 'S', method: 'POST', path: '/drop/k9p3mz/stripe', body: { id: 'evt_3PqXnK2eZvKYlo2C1Bvc7Mq8', type: 'payment_intent.succeeded', data: { object: { amount: 4999, currency: 'usd', status: 'succeeded', customer: 'cus_QkWj9aB12c' } } } },
  { id: 'github', name: 'GitHub', initial: 'G', method: 'POST', path: '/drop/k9p3mz/github', body: { action: 'opened', number: 142, pull_request: { title: 'Fix auth bug in session middleware', user: { login: 'octocat' }, head: { ref: 'fix/session-auth' } } } },
  { id: 'shopify', name: 'Shopify', initial: 'S', method: 'POST', path: '/drop/k9p3mz/shopify', body: { topic: 'orders/paid', order_id: 8821, total_price: '49.00', currency: 'USD', customer: { email: 'jane@example.com' } } },
  { id: 'discord', name: 'Discord', initial: 'D', method: 'POST', path: '/drop/k9p3mz/discord', body: { type: 'MESSAGE_CREATE', content: 'Deploy completed ✓', channel_id: '1023948503', author: { username: 'deploybot' } } },
  { id: 'slack',   name: 'Slack',   initial: 'S', method: 'POST', path: '/drop/k9p3mz/slack',   body: { event: { type: 'app_mention', user: 'U05ABCD123', text: '<@U07XYZ> ship it', channel: 'C03DEV001' }, event_id: 'Ev09BC4M2X' } },
  { id: 'twilio',  name: 'Twilio',  initial: 'T', method: 'POST', path: '/drop/k9p3mz/twilio',  body: { MessageStatus: 'delivered', MessageSid: 'SM7e0c2f1d4a5b6e9a1', To: '+15550009999', From: '+15551112222' } },
  { id: 'linear',  name: 'Linear',  initial: 'L', method: 'POST', path: '/drop/k9p3mz/linear',  body: { action: 'create', type: 'Issue', data: { title: 'Webhook timeout on retry', state: 'In Progress', priority: 2 } } },
  { id: 'vercel',  name: 'Vercel',  initial: 'V', method: 'POST', path: '/drop/k9p3mz/vercel',  body: { type: 'deployment.succeeded', deployment: { url: 'hookdrop-7gx4.vercel.app', meta: { branch: 'main', commit: 'a3f9bc7' } } } },
]

function highlightVal(v: unknown, depth = 0): React.ReactNode {
  if (v === null) return <span className="jm">null</span>
  if (typeof v === 'boolean') return <span className="jb">{String(v)}</span>
  if (typeof v === 'number') return <span className="jn">{v}</span>
  if (typeof v === 'string') return <span className="js">&quot;{v}&quot;</span>
  if (Array.isArray(v)) {
    if (v.length === 0) return <span className="jm">[]</span>
    return (
      <>
        <span className="jm">[</span>
        {v.map((x, i) => (
          <div key={i} className="indent">{highlightVal(x, depth + 1)}{i < v.length - 1 ? <span className="jm">,</span> : null}</div>
        ))}
        <div><span className="jm">]</span></div>
      </>
    )
  }
  if (typeof v === 'object' && v !== null) {
    const entries = Object.entries(v as Record<string, unknown>)
    if (entries.length === 0) return <span className="jm">{'{}'}</span>
    return (
      <>
        <span className="jm">{'{'}</span>
        {entries.map(([k, val], i) => (
          <div key={k} className="indent">
            <span className="jk">&quot;{k}&quot;</span><span className="jm">: </span>
            {highlightVal(val, depth + 1)}
            {i < entries.length - 1 ? <span className="jm">,</span> : null}
          </div>
        ))}
        <div><span className="jm">{'}'}</span></div>
      </>
    )
  }
  return <span className="jm">{String(v)}</span>
}

type PlaygroundEvent = {
  id: number; srcId: string; srcName: string
  method: string; path: string; body: Record<string, unknown>; ts: number
}

function Playground() {
  const [activeId, setActiveId] = useState('stripe')
  const [events, setEvents] = useState<PlaygroundEvent[]>([])
  const counter = useRef(0)

  useEffect(() => {
    const src = SOURCES.find(s => s.id === activeId)
    if (!src) return
    const ev: PlaygroundEvent = {
      id: ++counter.current,
      srcId: src.id, srcName: src.name,
      method: src.method, path: src.path, body: src.body,
      ts: Date.now(),
    }
    setEvents(prev => [ev, ...prev].slice(0, 5))
  }, [activeId])

  const selected = events[0]

  return (
    <div className="pg-shell">
      <div className="pg-sources">
        {SOURCES.map(s => (
          <button
            key={s.id}
            className={'pg-source' + (activeId === s.id ? ' active' : '')}
            onClick={() => setActiveId(s.id)}
          >
            <span className="dot" />
            {s.name}
          </button>
        ))}
      </div>
      <div className="pg-inspector">
        <div className="pg-list">
          <div className="pg-list-header">
            <span className="lbl">Requests</span>
            <span className="ct">{events.length}</span>
          </div>
          {events.map((ev, i) => (
            <div key={ev.id} className={'pg-event' + (i === 0 ? ' active' : '')}>
              <span className={`mp ${ev.method}`}>{ev.method}</span>
              <div>
                <div className="path">{ev.srcName.toLowerCase()}-webhook</div>
                <div className="t">{i === 0 ? 'just now' : `${i * 5}s ago`}</div>
              </div>
              <span className="s">200</span>
            </div>
          ))}
        </div>
        <div className="pg-detail">
          {selected && (
            <>
              <div className="pg-detail-head">
                <span className="mp">{selected.method}</span>
                <span className="s">200</span>
                <span className="ts">{new Date(selected.ts).toUTCString()}</span>
              </div>
              <div className="pg-detail-body">
                <div className="code-block">{highlightVal(selected.body)}</div>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

// ── Page ─────────────────────────────────────────────────────
export default function HomePage() {
  const router = useRouter()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Reset stuck loading state when page is restored from bfcache (browser back)
  useEffect(() => {
    const reset = () => setLoading(false)
    window.addEventListener('pageshow', reset)
    return () => window.removeEventListener('pageshow', reset)
  }, [])

  const handleCreate = useCallback(async () => {
    if (loading) return
    setLoading(true)
    setError(null)
    try {
      const drop = await createDrop()
      localStorage.setItem(`token:${drop.url_slug}`, drop.session_token)
      localStorage.setItem('hookdrop:lastSlug', drop.url_slug)
      router.push(`/drop/${drop.url_slug}`)
    } catch {
      setError('Failed to create a drop. Is the server running?')
      setLoading(false)
    }
  }, [loading, router])

  const handleOpenDashboard = useCallback(async () => {
    if (loading) return
    const slug = localStorage.getItem('hookdrop:lastSlug')
    if (slug) {
      router.push(`/drop/${slug}`)
      return
    }
    // No active drop — create one
    setLoading(true)
    setError(null)
    try {
      const drop = await createDrop()
      localStorage.setItem(`token:${drop.url_slug}`, drop.session_token)
      localStorage.setItem('hookdrop:lastSlug', drop.url_slug)
      router.push(`/drop/${drop.url_slug}`)
    } catch {
      setError('Failed to create a drop. Is the server running?')
      setLoading(false)
    }
  }, [loading, router])

  const CreateBtn = ({ style }: { style?: React.CSSProperties }) => (
    <button className="btn-primary" onClick={handleCreate} disabled={loading} style={style}>
      {loading ? (<><span className="spinner" /> Creating your Drop…</>) : <>Get a Drop URL →</>}
    </button>
  )

  return (
    <main style={{ position: 'relative', zIndex: 1 }}>
      <nav className="lp-nav">
        <div className="nav-inner">
          <div className="brand">Webhook<span className="x">X</span></div>
          <div className="nav-right">
            <span className="beta-pill"><span className="beta-dot" /> beta</span>
            {/* <LiveCounter /> */}
            <a className="nav-link" href="#how">How it works</a>
            <a className="nav-link" href="#playground">Playground</a>
            <a className="nav-link" href="#specs">Specs</a>
            <button className="btn-ghost" onClick={handleOpenDashboard}>Open Dashboard →</button>
          </div>
        </div>
      </nav>

      {/* Hero */}
      <section className="hero">
        <div className="container">
          <span className="label-tag">Free · No signup · Expires in 24h</span>
          <h1>Drop URL.<br />Catch payloads.<br /><span className="accent">Move on.</span></h1>
          <p className="lede">
            HookDrop is a zero-config webhook receiver. Get a live URL in one click, point any service at it, and watch every request land in real time.
          </p>
          <div className="hero-ctas">
            <CreateBtn />
            <a className="btn-secondary" href="#how">Watch the 20-second demo ↓</a>
          </div>
          <div className="hero-fineprint">No account · No CLI · No tunnel · No setup</div>
          {error && (
            <p style={{ fontFamily: 'var(--mono)', fontSize: '12px', color: 'var(--red)', marginTop: '14px' }}>
              {error}
            </p>
          )}
        </div>
      </section>

      {/* Live wire */}
      <section className="livewire-section">
        <div className="container">
          <LiveWire />
        </div>
      </section>

      {/* How it works */}
      <section className="block" id="how">
        <div className="container">
          <Reveal>
            <div className="section-head">
              <span className="label-tag">How it works</span>
              <h2>Four steps. <span className="light">Twenty seconds.</span></h2>
              <p>Click a step to jump straight to it, or hit pause to read along.</p>
            </div>
          </Reveal>
          <Reveal delay={150}>
            <DemoReel />
          </Reveal>
        </div>
      </section>

      {/* Before / after */}
      <section className="block">
        <div className="container">
          <Reveal>
            <div className="section-head">
              <span className="label-tag">The setup you don&apos;t need</span>
              <h2>You&apos;ve done this dance. <span className="light">Stop dancing.</span></h2>
            </div>
          </Reveal>
          <Reveal delay={100}>
            <div className="compare">
              <div className="compare-card before">
                <h3>Before</h3>
                <p className="tagline">Webhook debugging, the old way.</p>
                <ol className="compare-steps">
                  <li><span className="n">01</span>Spin up <code>express</code> on a free port.</li>
                  <li><span className="n">02</span>Wire <code>app.post(&apos;/hook&apos;, …)</code> and a hundred <code>console.log</code>s.</li>
                  <li><span className="n">03</span>Run <code>ngrok http 3000</code>, copy the random URL.</li>
                  <li><span className="n">04</span>Paste in Stripe. Save. Realize you typo&apos;d.</li>
                  <li><span className="n">05</span>Ship a redeploy because logs were missing a field.</li>
                  <li><span className="n">06</span>Tunnel dies. Restart. Repeat.</li>
                </ol>
                <div className="compare-foot">
                  <span className="foot-line">Minutes of setup</span> before you see a single request
                </div>
              </div>
              <div className="compare-card after">
                <h3>With HookDrop</h3>
                <p className="tagline">Just a URL. And a screen.</p>
                <ol className="compare-steps">
                  <li><span className="n">01</span>Click <code>Get a Drop URL</code>.</li>
                  <li><span className="n">02</span>Paste it into Stripe.</li>
                  <li><span className="n">03</span>Watch the request arrive — every header, every byte.</li>
                </ol>
                <div className="compare-foot">
                  <span className="foot-line">Seconds</span> from idea to inspecting the payload
                </div>
              </div>
            </div>
          </Reveal>
        </div>
      </section>

      {/* Playground */}
      <section className="block" id="playground">
        <div className="container">
          <Reveal>
            <div className="section-head">
              <span className="label-tag">Preview · static payloads</span>
              <h2>A peek inside the inspector. <span className="light">Before you wire anything up.</span></h2>
              <p>Canned payloads shaped like the real thing from services you&apos;ll point at HookDrop. Click around to see how the inspector handles each one — then grab a Drop URL above for the live version.</p>
            </div>
          </Reveal>
          <Reveal delay={150}>
            <Playground />
          </Reveal>
        </div>
      </section>

      {/* Specs */}
      <section className="block" id="specs">
        <div className="container-narrow">
          <Reveal>
            <div className="section-head">
              <span className="label-tag">What you actually get</span>
              <h2>Specs you don&apos;t have to read twice.</h2>
            </div>
          </Reveal>
          <Reveal delay={100}>
            <div className="specs">
              <div className="spec-row">
                <div className="lhs">
                  <span className="key">latency.p99</span>
                  <span className="desc">From sender → your screen. Benchmarks land with v1.</span>
                </div>
                <span className="val val-pending">measured at beta</span>
              </div>
              <div className="spec-row">
                <div className="lhs">
                  <span className="key">retention</span>
                  <span className="desc">Every Drop expires automatically. No cleanup.</span>
                </div>
                <span className="val">24<span className="unit">h</span></span>
              </div>
              <div className="spec-row">
                <div className="lhs">
                  <span className="key">payload.max</span>
                  <span className="desc">Covers every real webhook from every major service.</span>
                </div>
                <span className="val">1<span className="unit">MB</span></span>
              </div>
              <div className="spec-row">
                <div className="lhs">
                  <span className="key">requests / drop</span>
                  <span className="desc">Generous limits, real protection. Resets when the Drop expires.</span>
                </div>
                <span className="val">10,000</span>
              </div>
              <div className="spec-row">
                <div className="lhs">
                  <span className="key">methods</span>
                  <span className="desc">POST, GET, PUT, PATCH, DELETE — anything HTTP.</span>
                </div>
                <span className="val">ALL</span>
              </div>
              <div className="spec-row">
                <div className="lhs">
                  <span className="key">price</span>
                  <span className="desc">While in beta.</span>
                </div>
                <span className="val">$0</span>
              </div>
            </div>
          </Reveal>
        </div>
      </section>

      {/* Final CTA */}
      <section className="final-cta">
        <div className="container">
          <Reveal>
            <h2>Drop<span className="dot">.</span> Inspect<span className="dot">.</span> Done<span className="dot">.</span></h2>
            <p>Get a live URL. Paste it anywhere. We&apos;ll catch what comes back.</p>
            <CreateBtn style={{ fontSize: '17px', padding: '16px 28px' }} />
            <div className="hero-fineprint" style={{ marginTop: '18px' }}>Free · No signup · Expires in 24h</div>
          </Reveal>
        </div>
      </section>

      {/* Footer */}
      <footer className="lp-footer">
        <div className="footer-inner">
          <div>HookDrop · Free while in beta · Built for developers</div>
          <div>
            <a href="#">Docs</a>
            <a href="#">GitHub</a>
            <a href="#">Status</a>
          </div>
        </div>
      </footer>
    </main>
  )
}

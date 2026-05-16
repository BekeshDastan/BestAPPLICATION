import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import { Mail, Loader2 } from 'lucide-react'
import api from '../../lib/api'

const COOLDOWN = 60

export default function ForgotPasswordPage() {
  const [email, setEmail]     = useState('')
  const [error, setError]     = useState('')
  const [loading, setLoading] = useState(false)
  const [sent, setSent]       = useState(false)
  const [timer, setTimer]     = useState(0)
  const intervalRef = useRef(null)

  function startCooldown() {
    setTimer(COOLDOWN)
    intervalRef.current = setInterval(() => {
      setTimer((t) => {
        if (t <= 1) { clearInterval(intervalRef.current); return 0 }
        return t - 1
      })
    }, 1000)
  }

  useEffect(() => () => clearInterval(intervalRef.current), [])

  async function handleSubmit(ev) {
    ev.preventDefault()
    if (!email) { setError('Email is required'); return }
    setLoading(true)
    setError('')
    try {
      await api.post('/auth/forgot-password', { email })
      setSent(true)
      startCooldown()
    } catch (err) {
      setError(err.response?.data?.error ?? 'Something went wrong')
    } finally {
      setLoading(false)
    }
  }

  async function handleResend() {
    if (timer > 0) return
    setLoading(true)
    try {
      await api.post('/auth/forgot-password', { email })
      startCooldown()
    } finally {
      setLoading(false)
    }
  }

  /* ── Sent state ── */
  if (sent) {
    return (
      <div className="card p-8 text-center animate-fade-in">
        <div
          className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
          style={{ background: 'rgba(124,58,237,0.15)' }}
        >
          <Mail size={28} style={{ color: 'var(--accent)' }} />
        </div>
        <h2 className="text-xl font-bold text-hi mb-2">Check your inbox</h2>
        <p className="text-sm text-lo">
          We sent a reset link to
        </p>
        <p className="text-sm font-semibold text-hi mt-0.5 mb-6">{email}</p>

        <button
          onClick={handleResend}
          disabled={timer > 0 || loading}
          className="btn-ghost w-full text-sm"
        >
          {timer > 0 ? `Resend in ${timer}s` : loading ? 'Sending…' : 'Resend email'}
        </button>

        <p className="mt-4 text-sm text-lo">
          <Link to="/login" style={{ color: 'var(--accent)' }} className="font-medium">
            Back to sign in
          </Link>
        </p>
      </div>
    )
  }

  /* ── Form ── */
  return (
    <div className="card p-8 animate-fade-in">
      <h1 className="text-2xl font-bold text-hi mb-1">Forgot password?</h1>
      <p className="text-sm text-lo mb-6">
        Enter your email and we'll send you a reset link.
      </p>

      <form onSubmit={handleSubmit} noValidate className="space-y-4">
        <div>
          <label className="block text-xs font-medium text-lo mb-1.5">Email</label>
          <input
            type="email"
            value={email}
            onChange={(e) => { setEmail(e.target.value); setError('') }}
            placeholder="you@example.com"
            autoComplete="email"
            className={`input-base ${error ? 'error' : ''}`}
          />
          {error && (
            <p className="mt-1 text-xs" style={{ color: 'var(--danger)' }}>{error}</p>
          )}
        </div>

        <button type="submit" disabled={loading} className="btn-primary w-full">
          {loading ? <Loader2 size={16} className="animate-spin" /> : null}
          {loading ? 'Sending…' : 'Send reset link'}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-lo">
        Remember your password?{' '}
        <Link to="/login" className="font-medium" style={{ color: 'var(--accent)' }}>
          Sign in
        </Link>
      </p>
    </div>
  )
}

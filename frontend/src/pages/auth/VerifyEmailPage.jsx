import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { CheckCircle2, XCircle, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'

export default function VerifyEmailPage() {
  const [params] = useSearchParams()
  const token = params.get('token')

  const [state, setState] = useState('loading') // loading | success | error
  const [resending, setResending] = useState(false)
  const [email, setEmail] = useState(params.get('email') ?? '')

  useEffect(() => {
    if (!token) { setState('error'); return }
    api.post('/auth/verify-email', { token })
      .then(() => setState('success'))
      .catch(() => setState('error'))
  }, [token])

  async function handleResend() {
    if (!email.trim()) {
      toast.error('Enter your email to resend')
      return
    }
    setResending(true)
    try {
      await api.post('/auth/resend-verification', { email: email.trim() })
      toast.success('Verification email sent')
    } catch {
      toast.error('Failed to resend')
    } finally {
      setResending(false)
    }
  }

  /* ── Loading skeleton ── */
  if (state === 'loading') {
    return (
      <div className="card p-8 flex flex-col items-center gap-4 animate-fade-in">
        <div className="skeleton w-16 h-16 rounded-full" />
        <div className="skeleton w-40 h-5 rounded-btn" />
        <div className="skeleton w-56 h-4 rounded-btn" />
      </div>
    )
  }

  /* ── Success ── */
  if (state === 'success') {
    return (
      <div className="card p-8 text-center animate-fade-in">
        <div
          className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
          style={{ background: 'rgba(34,197,94,0.12)' }}
        >
          <CheckCircle2 size={30} style={{ color: 'var(--online)' }} />
        </div>
        <h2 className="text-xl font-bold text-hi mb-2">Email verified!</h2>
        <p className="text-sm text-lo mb-6">
          Your account is now active. Sign in to get started.
        </p>
        <Link to="/login" className="btn-primary w-full inline-flex">
          Sign in →
        </Link>
      </div>
    )
  }

  /* ── Error ── */
  return (
    <div className="card p-8 text-center animate-fade-in">
      <div
        className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
        style={{ background: 'rgba(239,68,68,0.12)' }}
      >
        <XCircle size={30} style={{ color: 'var(--danger)' }} />
      </div>
      <h2 className="text-xl font-bold text-hi mb-2">Link expired or invalid</h2>
      <p className="text-sm text-lo mb-6">
        This verification link is no longer valid. Enter your email to get a new one.
      </p>
      <input
        type="email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="you@example.com"
        className="input-base mb-3"
        autoComplete="email"
      />
      <button
        onClick={handleResend}
        disabled={resending || !email.trim()}
        className="btn-primary w-full disabled:opacity-40"
      >
        {resending ? <Loader2 size={16} className="animate-spin" /> : null}
        {resending ? 'Sending…' : 'Resend verification email'}
      </button>
      <p className="mt-4 text-sm text-lo">
        <Link to="/login" style={{ color: 'var(--accent)' }} className="font-medium">
          Back to sign in
        </Link>
      </p>
    </div>
  )
}

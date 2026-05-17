import { useState, useEffect } from 'react'
import { Link, useSearchParams, useNavigate } from 'react-router-dom'
import { Eye, EyeOff, Loader2, CheckCircle2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import { getPasswordStrength } from '../../lib/utils'

export default function ResetPasswordPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const token = params.get('token')

  const [form, setForm]         = useState({ password: '', confirm: '' })
  const [showPw, setShowPw]     = useState(false)
  const [errors, setErrors]     = useState({})
  const [loading, setLoading]   = useState(false)
  const [success, setSuccess]   = useState(false)
  const [countdown, setCountdown] = useState(null)

  const strength = form.password ? getPasswordStrength(form.password) : null

  /* Countdown redirect after success */
  useEffect(() => {
    if (countdown === null) return
    if (countdown === 0) { navigate('/login'); return }
    const t = setTimeout(() => setCountdown((c) => c - 1), 1000)
    return () => clearTimeout(t)
  }, [countdown, navigate])

  function set(field, value) {
    setForm((f) => ({ ...f, [field]: value }))
    setErrors((e) => ({ ...e, [field]: '' }))
  }

  function validate() {
    const e = {}
    if (!token)              e.form     = 'Invalid or missing reset token'
    if (!form.password)      e.password = 'Password is required'
    else if (form.password.length < 8) e.password = 'Minimum 8 characters'
    if (form.password !== form.confirm) e.confirm = 'Passwords do not match'
    setErrors(e)
    return !Object.keys(e).length
  }

  async function handleSubmit(ev) {
    ev.preventDefault()
    if (!validate()) return
    setLoading(true)
    try {
      await api.post('/auth/reset-password', {
        token,
        new_password: form.password,
      })
      setSuccess(true)
      setCountdown(2)
      toast.success('Password updated!')
    } catch (err) {
      setErrors({ form: err.response?.data?.error ?? 'Reset failed. Link may have expired.' })
    } finally {
      setLoading(false)
    }
  }

  /* ── Success ── */
  if (success) {
    return (
      <div className="card p-8 text-center animate-fade-in">
        <div
          className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
          style={{ background: 'rgba(34,197,94,0.12)' }}
        >
          <CheckCircle2 size={30} style={{ color: 'var(--online)' }} />
        </div>
        <h2 className="text-xl font-bold text-hi mb-2">Password updated!</h2>
        <p className="text-sm text-lo">
          Redirecting to sign in{countdown !== null ? ` in ${countdown}s` : '…'}
        </p>
      </div>
    )
  }

  /* ── Form ── */
  return (
    <div className="card p-8 animate-fade-in">
      <h1 className="text-2xl font-bold text-hi mb-1">Set new password</h1>
      <p className="text-sm text-lo mb-6">Choose a strong password.</p>

      {errors.form && (
        <div
          className="mb-4 px-4 py-3 rounded-btn text-sm"
          style={{
            background: 'rgba(239,68,68,0.1)',
            border: '1px solid rgba(239,68,68,0.3)',
            color: 'var(--danger)',
          }}
        >
          {errors.form}
        </div>
      )}

      <form onSubmit={handleSubmit} noValidate className="space-y-4">
        {/* New password */}
        <div>
          <label className="block text-xs font-medium text-lo mb-1.5">New Password</label>
          <div className="relative">
            <input
              type={showPw ? 'text' : 'password'}
              value={form.password}
              onChange={(e) => set('password', e.target.value)}
              placeholder="••••••••"
              autoComplete="new-password"
              className={`input-base pr-10 ${errors.password ? 'error' : ''}`}
            />
            <button
              type="button"
              onClick={() => setShowPw((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 text-lo"
            >
              {showPw ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>

          {/* Strength meter */}
          {form.password && strength && (
            <div className="mt-2">
              <div
                className="h-1 rounded-full overflow-hidden"
                style={{ background: 'var(--surface-high)' }}
              >
                <div
                  className="h-full rounded-full transition-all duration-300"
                  style={{ width: strength.width, background: strength.color }}
                />
              </div>
              <p className="text-[11px] mt-1" style={{ color: strength.color }}>
                {strength.level}
              </p>
            </div>
          )}

          {errors.password && (
            <p className="mt-1 text-xs" style={{ color: 'var(--danger)' }}>{errors.password}</p>
          )}
        </div>

        {/* Confirm */}
        <div>
          <label className="block text-xs font-medium text-lo mb-1.5">Confirm Password</label>
          <input
            type="password"
            value={form.confirm}
            onChange={(e) => set('confirm', e.target.value)}
            placeholder="••••••••"
            autoComplete="new-password"
            className={`input-base ${errors.confirm ? 'error' : ''}`}
          />
          {errors.confirm && (
            <p className="mt-1 text-xs" style={{ color: 'var(--danger)' }}>{errors.confirm}</p>
          )}
        </div>

        <button type="submit" disabled={loading} className="btn-primary w-full">
          {loading ? <Loader2 size={16} className="animate-spin" /> : null}
          {loading ? 'Saving…' : 'Set new password'}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-lo">
        <Link to="/login" className="font-medium" style={{ color: 'var(--accent)' }}>
          Back to sign in
        </Link>
      </p>
    </div>
  )
}

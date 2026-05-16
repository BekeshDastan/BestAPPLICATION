import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Eye, EyeOff, Loader2, Mail } from 'lucide-react'
import api from '../../lib/api'

export default function RegisterPage() {
  const [form, setForm] = useState({
    full_name: '', username: '', email: '', password: '', confirm: '',
  })
  const [showPw, setShowPw]   = useState(false)
  const [errors, setErrors]   = useState({})
  const [loading, setLoading] = useState(false)
  const [sent, setSent]       = useState(false)
  const [resending, setResending] = useState(false)

  function set(field, value) {
    setForm((f) => ({ ...f, [field]: value }))
    setErrors((e) => ({ ...e, [field]: '' }))
  }

  function validate() {
    const e = {}
    if (!form.full_name.trim()) e.full_name = 'Full name is required'
    if (!form.username.trim())  e.username  = 'Username is required'
    else if (!/^[a-z0-9_.]{3,30}$/i.test(form.username))
      e.username = 'Letters, numbers, _ and . only (3-30 chars)'
    if (!form.email) e.email = 'Email is required'
    if (!form.password) e.password = 'Password is required'
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
      await api.post('/auth/register', {
        full_name: form.full_name,
        username:  form.username,
        email:     form.email,
        password:  form.password,
      })
      setSent(true)
    } catch (err) {
      const msg = err.response?.data?.error ?? 'Registration failed'
      setErrors({ form: msg })
    } finally {
      setLoading(false)
    }
  }

  async function handleResend() {
    setResending(true)
    try {
      await api.post('/auth/resend-verification', { email: form.email })
    } finally {
      setResending(false)
    }
  }

  /* ── Success state ── */
  if (sent) {
    return (
      <div className="card p-8 text-center animate-fade-in">
        <div
          className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-4"
          style={{ background: 'rgba(124,58,237,0.15)' }}
        >
          <Mail size={28} style={{ color: 'var(--accent)' }} />
        </div>
        <h2 className="text-xl font-bold text-hi mb-2">Check your email</h2>
        <p className="text-sm text-lo mb-1">
          We sent a verification link to
        </p>
        <p className="text-sm font-medium text-hi mb-6">{form.email}</p>
        <button
          onClick={handleResend}
          disabled={resending}
          className="btn-ghost w-full text-sm"
        >
          {resending ? <Loader2 size={14} className="animate-spin" /> : null}
          {resending ? 'Sending…' : 'Resend verification email'}
        </button>
        <p className="mt-4 text-sm text-lo">
          Already verified?{' '}
          <Link to="/login" style={{ color: 'var(--accent)' }} className="font-medium">
            Sign in
          </Link>
        </p>
      </div>
    )
  }

  /* ── Form ── */
  return (
    <div className="card p-8 animate-fade-in">
      <h1 className="text-2xl font-bold text-hi mb-1">Create account</h1>
      <p className="text-sm text-lo mb-6">Join Social today</p>

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
        {[
          { field: 'full_name', label: 'Full Name',  type: 'text',  placeholder: 'Your name', auto: 'name' },
          { field: 'username',  label: 'Username',   type: 'text',  placeholder: 'yourhandle', auto: 'username' },
          { field: 'email',     label: 'Email',      type: 'email', placeholder: 'you@example.com', auto: 'email' },
        ].map(({ field, label, type, placeholder, auto }) => (
          <div key={field}>
            <label className="block text-xs font-medium text-lo mb-1.5">{label}</label>
            <input
              type={type}
              value={form[field]}
              onChange={(e) => set(field, e.target.value)}
              placeholder={placeholder}
              autoComplete={auto}
              className={`input-base ${errors[field] ? 'error' : ''}`}
            />
            {errors[field] && (
              <p className="mt-1 text-xs" style={{ color: 'var(--danger)' }}>{errors[field]}</p>
            )}
          </div>
        ))}

        {/* Password */}
        <div>
          <label className="block text-xs font-medium text-lo mb-1.5">Password</label>
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

        <button type="submit" disabled={loading} className="btn-primary w-full mt-2">
          {loading ? <Loader2 size={16} className="animate-spin" /> : null}
          {loading ? 'Creating account…' : 'Create account'}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-lo">
        Already have an account?{' '}
        <Link to="/login" className="font-medium" style={{ color: 'var(--accent)' }}>
          Sign in
        </Link>
      </p>
    </div>
  )
}

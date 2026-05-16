import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { Eye, EyeOff, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api, { setRefreshToken } from '../../lib/api'
import useAuthStore from '../../store/authStore'

export default function LoginPage() {
  const navigate = useNavigate()
  const { login } = useAuthStore()

  const [form, setForm] = useState({ email: '', password: '' })
  const [showPw, setShowPw] = useState(false)
  const [errors, setErrors] = useState({})
  const [loading, setLoading] = useState(false)

  function set(field, value) {
    setForm((f) => ({ ...f, [field]: value }))
    setErrors((e) => ({ ...e, [field]: '' }))
  }

  function validate() {
    const e = {}
    if (!form.email)    e.email    = 'Email is required'
    if (!form.password) e.password = 'Password is required'
    setErrors(e)
    return !Object.keys(e).length
  }

  async function handleSubmit(ev) {
    ev.preventDefault()
    if (!validate()) return
    setLoading(true)
    try {
      const { data } = await api.post('/auth/login', {
        email:    form.email,
        password: form.password,
      })
      if (data.refresh_token) setRefreshToken(data.refresh_token)
      const me = await api.get('/users/me', {
        headers: { Authorization: `Bearer ${data.access_token}` },
      })
      login(data.access_token, me.data)
      navigate('/')
    } catch (err) {
      const msg = err.response?.data?.error ?? 'Invalid credentials'
      setErrors({ form: msg })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card p-8 animate-fade-in">
      <h1 className="text-2xl font-bold text-hi mb-1">Welcome back</h1>
      <p className="text-sm text-lo mb-6">Sign in to continue</p>

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
        {/* Email */}
        <div>
          <label className="block text-xs font-medium text-lo mb-1.5">Email</label>
          <input
            type="email"
            value={form.email}
            onChange={(e) => set('email', e.target.value)}
            placeholder="you@example.com"
            className={`input-base ${errors.email ? 'error' : ''}`}
            autoComplete="email"
          />
          {errors.email && (
            <p className="mt-1 text-xs" style={{ color: 'var(--danger)' }}>{errors.email}</p>
          )}
        </div>

        {/* Password */}
        <div>
          <div className="flex items-center justify-between mb-1.5">
            <label className="text-xs font-medium text-lo">Password</label>
            <Link
              to="/forgot-password"
              className="text-xs transition-colors"
              style={{ color: 'var(--accent)' }}
            >
              Forgot password?
            </Link>
          </div>
          <div className="relative">
            <input
              type={showPw ? 'text' : 'password'}
              value={form.password}
              onChange={(e) => set('password', e.target.value)}
              placeholder="••••••••"
              className={`input-base pr-10 ${errors.password ? 'error' : ''}`}
              autoComplete="current-password"
            />
            <button
              type="button"
              onClick={() => setShowPw((v) => !v)}
              className="absolute right-3 top-1/2 -translate-y-1/2 transition-colors"
              style={{ color: 'var(--text-2)' }}
            >
              {showPw ? <EyeOff size={16} /> : <Eye size={16} />}
            </button>
          </div>
          {errors.password && (
            <p className="mt-1 text-xs" style={{ color: 'var(--danger)' }}>{errors.password}</p>
          )}
        </div>

        <button
          type="submit"
          disabled={loading}
          className="btn-primary w-full mt-2"
        >
          {loading ? <Loader2 size={16} className="animate-spin" /> : null}
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>

      <p className="mt-6 text-center text-sm text-lo">
        Don't have an account?{' '}
        <Link
          to="/register"
          className="font-medium transition-colors"
          style={{ color: 'var(--accent)' }}
        >
          Create account
        </Link>
      </p>
    </div>
  )
}

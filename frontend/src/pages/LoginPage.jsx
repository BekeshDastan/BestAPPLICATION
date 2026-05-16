import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../AuthContext'

export default function LoginPage() {
  const { login } = useAuth()
  const nav = useNavigate()
  const [form, setForm] = useState({ email: '', password: '' })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handle = e => setForm(f => ({ ...f, [e.target.name]: e.target.value }))

  const submit = async e => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try { await login(form.email, form.password); nav('/') }
    catch (err) { setError(err.response?.data?.error || 'Login failed') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.page}>
      <div style={s.card}>
        <h1 style={s.logo}>Social</h1>
        <p style={s.sub}>Sign in to see photos and videos from your friends.</p>
        {error && <div style={s.err}>{error}</div>}
        <form onSubmit={submit} style={s.form}>
          <input name="email" type="email" placeholder="Email" value={form.email} onChange={handle} style={s.input} required />
          <input name="password" type="password" placeholder="Password" value={form.password} onChange={handle} style={s.input} required />
          <button type="submit" disabled={loading} style={s.btn}>{loading ? 'Signing in…' : 'Log In'}</button>
        </form>
        <Link to="/forgot-password" style={s.forgot}>Forgot password?</Link>
        <div style={s.divider}><span>OR</span></div>
        <p style={s.foot}>Don't have an account? <Link to="/register" style={s.link}>Sign up</Link></p>
      </div>
    </div>
  )
}

const s = {
  page: { minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#fafafa' },
  card: { background: '#fff', border: '1px solid #dbdbdb', padding: '40px 40px 24px', width: 360, borderRadius: 4 },
  logo: { textAlign: 'center', fontSize: 36, fontWeight: 700, marginBottom: 8 },
  sub: { textAlign: 'center', color: '#8e8e8e', fontSize: 14, fontWeight: 600, marginBottom: 20 },
  form: { display: 'flex', flexDirection: 'column', gap: 8 },
  input: { padding: '9px 10px', background: '#fafafa', border: '1px solid #dbdbdb', borderRadius: 4, fontSize: 14 },
  btn: { padding: '8px', background: '#0095f6', color: '#fff', border: 'none', borderRadius: 4, fontWeight: 700, fontSize: 14, marginTop: 4 },
  err: { background: '#fff3f3', color: '#e53935', fontSize: 13, padding: '8px 12px', borderRadius: 4, marginBottom: 8, textAlign: 'center' },
  forgot: { display: 'block', textAlign: 'center', marginTop: 14, fontSize: 13, color: '#0095f6' },
  divider: { textAlign: 'center', color: '#8e8e8e', fontSize: 13, margin: '16px 0', borderTop: '1px solid #dbdbdb', paddingTop: 16 },
  foot: { textAlign: 'center', fontSize: 14, margin: 0 },
  link: { color: '#0095f6', fontWeight: 600 },
}

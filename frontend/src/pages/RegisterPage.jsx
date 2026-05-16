import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../AuthContext'

export default function RegisterPage() {
  const { register } = useAuth()
  const nav = useNavigate()
  const [form, setForm] = useState({ email: '', username: '', password: '', full_name: '' })
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handle = e => setForm(f => ({ ...f, [e.target.name]: e.target.value }))

  const submit = async e => {
    e.preventDefault()
    setError('')
    setLoading(true)
    try { await register(form.email, form.username, form.password, form.full_name); nav('/') }
    catch (err) { setError(err.response?.data?.error || 'Registration failed') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.page}>
      <div style={s.card}>
        <h1 style={s.logo}>Social</h1>
        <p style={s.sub}>Sign up to see photos and videos from your friends.</p>
        {error && <div style={s.err}>{error}</div>}
        <form onSubmit={submit} style={s.form}>
          <input name="email" type="email" placeholder="Email" value={form.email} onChange={handle} style={s.input} required />
          <input name="full_name" placeholder="Full Name" value={form.full_name} onChange={handle} style={s.input} />
          <input name="username" placeholder="Username" value={form.username} onChange={handle} style={s.input} required />
          <input name="password" type="password" placeholder="Password" value={form.password} onChange={handle} style={s.input} required />
          <button type="submit" disabled={loading} style={s.btn}>{loading ? 'Signing up…' : 'Sign Up'}</button>
        </form>
        <p style={s.terms}>By signing up, you agree to our Terms of Service.</p>
        <div style={s.foot}>Have an account? <Link to="/login" style={s.link}>Log in</Link></div>
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
  terms: { fontSize: 12, color: '#8e8e8e', textAlign: 'center', marginTop: 16 },
  foot: { textAlign: 'center', fontSize: 14, marginTop: 8 },
  link: { color: '#0095f6', fontWeight: 600 },
}

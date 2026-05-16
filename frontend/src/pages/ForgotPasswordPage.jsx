import { useState } from 'react'
import { Link } from 'react-router-dom'
import { authApi } from '../api'

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('')
  const [token, setToken] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [step, setStep] = useState('request') // request | reset
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(false)

  const sendReset = async e => {
    e.preventDefault()
    setErr(''); setMsg(''); setLoading(true)
    try {
      await authApi.forgotPassword(email)
      setMsg('Reset email sent! Check your inbox. Then enter the token below.')
      setStep('reset')
    } catch (e) { setErr(e.response?.data?.error || 'Failed to send reset email') }
    finally { setLoading(false) }
  }

  const doReset = async e => {
    e.preventDefault()
    setErr(''); setLoading(true)
    try {
      await authApi.resetPassword(token, newPwd)
      setMsg('Password reset! You can now log in.')
      setStep('done')
    } catch (e) { setErr(e.response?.data?.error || 'Reset failed') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.page}>
      <div style={s.card}>
        <div style={s.lock}>🔒</div>
        <h2 style={s.title}>Trouble logging in?</h2>
        <p style={s.sub}>Enter your email and we'll send you a reset link.</p>
        {msg && <div style={s.success}>{msg}</div>}
        {err && <div style={s.err}>{err}</div>}
        {step === 'request' && (
          <form onSubmit={sendReset} style={s.form}>
            <input type="email" placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} style={s.input} required />
            <button type="submit" disabled={loading} style={s.btn}>{loading ? 'Sending…' : 'Send Reset Email'}</button>
          </form>
        )}
        {step === 'reset' && (
          <form onSubmit={doReset} style={s.form}>
            <input placeholder="Reset token from email" value={token} onChange={e => setToken(e.target.value)} style={s.input} required />
            <input type="password" placeholder="New password" value={newPwd} onChange={e => setNewPwd(e.target.value)} style={s.input} required />
            <button type="submit" disabled={loading} style={s.btn}>{loading ? 'Resetting…' : 'Reset Password'}</button>
          </form>
        )}
        {step === 'done' && <Link to="/login" style={s.backLink}>← Back to Log In</Link>}
        <div style={s.divider} />
        <Link to="/login" style={s.back}>Back to Log In</Link>
      </div>
    </div>
  )
}

const s = {
  page: { minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#fafafa' },
  card: { background: '#fff', border: '1px solid #dbdbdb', padding: '32px 40px', width: 360, borderRadius: 4, textAlign: 'center' },
  lock: { fontSize: 48, marginBottom: 8 },
  title: { margin: '0 0 8px', fontSize: 18, fontWeight: 600 },
  sub: { color: '#8e8e8e', fontSize: 14, margin: '0 0 20px' },
  form: { display: 'flex', flexDirection: 'column', gap: 8 },
  input: { padding: '9px 10px', background: '#fafafa', border: '1px solid #dbdbdb', borderRadius: 4, fontSize: 14 },
  btn: { padding: '8px', background: '#0095f6', color: '#fff', border: 'none', borderRadius: 4, fontWeight: 700, fontSize: 14 },
  success: { background: '#e8f5e9', color: '#2e7d32', fontSize: 13, padding: '8px 12px', borderRadius: 4, marginBottom: 8 },
  err: { background: '#fff3f3', color: '#e53935', fontSize: 13, padding: '8px 12px', borderRadius: 4, marginBottom: 8 },
  divider: { borderTop: '1px solid #dbdbdb', margin: '20px 0' },
  back: { color: '#0095f6', fontWeight: 600, fontSize: 14 },
  backLink: { display: 'block', marginTop: 16, color: '#0095f6', fontWeight: 600 },
}

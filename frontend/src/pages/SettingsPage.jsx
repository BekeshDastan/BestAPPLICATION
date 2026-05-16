import { useState } from 'react'
import { useAuth } from '../AuthContext'
import { userApi } from '../api'

export default function SettingsPage() {
  const { user, refreshUser } = useAuth()
  const [form, setForm] = useState({
    full_name: user?.full_name || '',
    bio: user?.bio || '',
    is_private: user?.is_private || false,
  })
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(false)

  const handle = e => {
    const { name, value, type, checked } = e.target
    setForm(f => ({ ...f, [name]: type === 'checkbox' ? checked : value }))
  }

  const save = async e => {
    e.preventDefault()
    setMsg(''); setErr(''); setLoading(true)
    try {
      await userApi.updateMe(form)
      await refreshUser()
      setMsg('Profile updated!')
    } catch (e) { setErr(e.response?.data?.error || 'Update failed') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.page}>
      <h2 style={s.title}>Edit Profile</h2>
      <div style={s.card}>
        <div style={s.userRow}>
          <div style={s.avatar}>{user?.username?.[0]?.toUpperCase() || '?'}</div>
          <div>
            <div style={s.username}>{user?.username}</div>
            <div style={s.email}>{user?.email}</div>
          </div>
        </div>
        {msg && <div style={s.success}>{msg}</div>}
        {err && <div style={s.err}>{err}</div>}
        <form onSubmit={save} style={s.form}>
          <label style={s.label}>
            <span>Full Name</span>
            <input name="full_name" value={form.full_name} onChange={handle} style={s.input} placeholder="Your full name" />
          </label>
          <label style={s.label}>
            <span>Bio</span>
            <textarea name="bio" value={form.bio} onChange={handle} style={s.textarea} rows={3} placeholder="Tell people about yourself" />
          </label>
          <label style={s.checkLabel}>
            <input type="checkbox" name="is_private" checked={form.is_private} onChange={handle} style={{ marginRight: 8 }} />
            Private account
            <span style={s.checkNote}>Only followers can see your posts</span>
          </label>
          <button type="submit" disabled={loading} style={s.btn}>{loading ? 'Saving…' : 'Submit'}</button>
        </form>
      </div>

      <div style={s.card}>
        <h3 style={s.sectionTitle}>Account Info</h3>
        <div style={s.infoRow}><span style={s.infoLabel}>Username</span><span>{user?.username}</span></div>
        <div style={s.infoRow}><span style={s.infoLabel}>Email</span><span>{user?.email}</span></div>
        <div style={s.infoRow}><span style={s.infoLabel}>Account Type</span><span>{user?.is_private ? 'Private' : 'Public'}</span></div>
        <div style={s.infoRow}><span style={s.infoLabel}>Verified</span><span>{user?.is_verified ? '✓ Verified' : 'Not verified'}</span></div>
        <div style={s.infoRow}><span style={s.infoLabel}>User ID</span><span style={{ fontSize: 12, fontFamily: 'monospace', color: '#8e8e8e' }}>{user?.id}</span></div>
      </div>
    </div>
  )
}

const s = {
  page: { maxWidth: 640, margin: '0 auto', padding: '24px 16px', display: 'flex', flexDirection: 'column', gap: 20 },
  title: { margin: 0, fontWeight: 300, fontSize: 24 },
  card: { background: '#fff', border: '1px solid #dbdbdb', borderRadius: 8, padding: '24px' },
  userRow: { display: 'flex', alignItems: 'center', gap: 16, marginBottom: 20 },
  avatar: { width: 56, height: 56, borderRadius: '50%', background: '#333', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 22 },
  username: { fontWeight: 600, fontSize: 16 },
  email: { fontSize: 13, color: '#8e8e8e' },
  form: { display: 'flex', flexDirection: 'column', gap: 16 },
  label: { display: 'flex', flexDirection: 'column', gap: 6, fontSize: 14, fontWeight: 600 },
  input: { padding: '9px 12px', border: '1px solid #dbdbdb', borderRadius: 6, fontSize: 14, fontWeight: 400 },
  textarea: { padding: '9px 12px', border: '1px solid #dbdbdb', borderRadius: 6, fontSize: 14, resize: 'vertical', fontWeight: 400 },
  checkLabel: { display: 'flex', alignItems: 'center', fontSize: 14, fontWeight: 600, cursor: 'pointer' },
  checkNote: { fontWeight: 400, color: '#8e8e8e', fontSize: 13, marginLeft: 8 },
  btn: { padding: '10px', background: '#0095f6', color: '#fff', border: 'none', borderRadius: 6, fontWeight: 700, fontSize: 14, alignSelf: 'flex-start', paddingLeft: 24, paddingRight: 24 },
  success: { background: '#e8f5e9', color: '#2e7d32', fontSize: 13, padding: '8px 12px', borderRadius: 4, marginBottom: 12 },
  err: { background: '#fff3f3', color: '#e53935', fontSize: 13, padding: '8px 12px', borderRadius: 4, marginBottom: 12 },
  sectionTitle: { margin: '0 0 16px', fontWeight: 600, fontSize: 16 },
  infoRow: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 0', borderBottom: '1px solid #fafafa', fontSize: 14 },
  infoLabel: { color: '#8e8e8e', fontWeight: 600, minWidth: 100 },
}

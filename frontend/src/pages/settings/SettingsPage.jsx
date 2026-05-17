import { useState, useRef, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Camera, Eye, EyeOff, Loader2, Trash2,
  Monitor, Smartphone, AlertTriangle,
} from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import Avatar from '../../components/shared/Avatar'
import { getPasswordStrength, formatRelativeTime } from '../../lib/utils'

// "Devices" tab hidden — backend has no real session/device tracking yet.
const TABS = ['Profile', 'Security', 'Privacy', 'Notifications']

export default function SettingsPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') ?? 'Profile'
  function setTab(t) { setSearchParams({ tab: t }) }

  return (
    <div className="max-w-4xl mx-auto px-4 py-6">
      <h1 className="text-2xl font-bold text-hi mb-6">Settings</h1>

      <div className="flex gap-6">
        {/* Left sub-nav */}
        <nav className="w-44 shrink-0 space-y-0.5">
          {TABS.map((t) => (
            <button
              key={t}
              onClick={() => setTab(t)}
              className="w-full text-left px-3 py-2.5 rounded-btn text-sm font-medium transition-colors"
              style={{
                background: tab === t ? 'var(--surface-high)' : 'transparent',
                color:      tab === t ? 'var(--text-1)' : 'var(--text-2)',
              }}
            >
              {t}
            </button>
          ))}
        </nav>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {tab === 'Profile'       && <ProfileTab />}
          {tab === 'Security'      && <SecurityTab />}
          {tab === 'Privacy'       && <PrivacyTab />}
          {tab === 'Notifications' && <NotificationsTab />}
          {tab === 'Devices'       && <DevicesTab />}
        </div>
      </div>
    </div>
  )
}

/* ─── Profile Tab ────────────────────────────────────────────────── */
function ProfileTab() {
  const { user, setUser, logout } = useAuthStore()
  const navigate = useNavigate()

  const [form, setForm] = useState({
    full_name: user?.full_name ?? '',
    username:  user?.username  ?? '',
    bio:       user?.bio       ?? '',
    email:     user?.email     ?? '',
  })
  const [saving,          setSaving]          = useState(false)
  const [uploadingAvatar, setUploadingAvatar] = useState(false)
  const [showDelete,      setShowDelete]      = useState(false)
  const [deleting,        setDeleting]        = useState(false)
  const fileRef = useRef(null)

  function field(key) {
    return (e) => setForm((p) => ({ ...p, [key]: e.target.value }))
  }

  async function uploadAvatar(file) {
    setUploadingAvatar(true)
    try {
      const { data: urlData } = await api.get('/media/upload-url', { params: { type: 'avatar' } })
      await fetch(urlData.upload_url, {
        method: 'PUT', body: file, headers: { 'Content-Type': file.type },
      })
      const { data } = await api.put('/users/avatar', { avatar_url: urlData.media_url })
      setUser({ ...user, avatar_url: data.avatar_url ?? urlData.media_url })
      toast.success('Photo updated!')
    } catch { toast.error('Failed to upload photo') }
    finally { setUploadingAvatar(false) }
  }

  async function saveProfile() {
    setSaving(true)
    try {
      const { data } = await api.put('/users/me', form)
      setUser({ ...user, ...data })
      toast.success('Profile saved!')
    } catch (err) {
      toast.error(err?.response?.data?.message ?? 'Failed to save profile')
    } finally { setSaving(false) }
  }

  async function deleteAccount() {
    setDeleting(true)
    try {
      await api.delete('/users/me')
      logout()
      navigate('/login')
      toast.success('Account deleted')
    } catch { toast.error('Failed to delete account') }
    finally { setDeleting(false) }
  }

  return (
    <div className="space-y-5">
      <div className="card p-6">
        <h2 className="font-semibold text-hi mb-5">Profile Info</h2>

        {/* Avatar */}
        <div className="flex items-center gap-5 mb-6">
          <div className="relative shrink-0">
            <Avatar src={user?.avatar_url} name={user?.full_name ?? user?.username} size={88} />
            {uploadingAvatar && (
              <div className="absolute inset-0 rounded-full bg-black/60 flex items-center justify-center">
                <Loader2 size={22} className="animate-spin text-white" />
              </div>
            )}
          </div>
          <div>
            <button
              onClick={() => fileRef.current?.click()}
              disabled={uploadingAvatar}
              className="btn-primary text-sm px-4 py-2 flex items-center gap-2 disabled:opacity-40"
            >
              <Camera size={14} />
              {uploadingAvatar ? 'Uploading...' : 'Upload new photo'}
            </button>
            <p className="text-xs text-lo mt-1.5">JPG, PNG or GIF. Max 5 MB.</p>
            <input
              ref={fileRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => { const f = e.target.files[0]; if (f) uploadAvatar(f) }}
            />
          </div>
        </div>

        {/* Fields */}
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <Field label="Full Name">
              <input type="text" value={form.full_name} onChange={field('full_name')} className="input-base text-sm" />
            </Field>
            <Field label="Username">
              <input type="text" value={form.username} onChange={field('username')} className="input-base text-sm" />
            </Field>
          </div>
          <Field label={`Bio (${form.bio.length}/150)`}>
            <textarea
              value={form.bio}
              onChange={field('bio')}
              maxLength={150}
              rows={3}
              placeholder="Write something about yourself..."
              className="input-base text-sm resize-none"
            />
          </Field>
          <Field label="Email">
            <input type="email" value={form.email} onChange={field('email')} className="input-base text-sm" />
          </Field>
        </div>

        <div className="flex justify-end mt-5">
          <button
            onClick={saveProfile}
            disabled={saving}
            className="btn-primary px-5 py-2 text-sm disabled:opacity-40 flex items-center gap-2"
          >
            {saving && <Loader2 size={14} className="animate-spin" />}
            {saving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>

      {/* Danger zone */}
      <div className="card p-6" style={{ borderColor: 'rgba(239,68,68,0.35)' }}>
        <h2 className="font-semibold mb-1" style={{ color: 'var(--danger)' }}>Danger Zone</h2>
        <p className="text-sm text-lo mb-4">
          Permanently delete your account and all associated data.
        </p>
        <button
          onClick={() => setShowDelete(true)}
          className="flex items-center gap-2 px-4 py-2 rounded-btn text-sm font-medium transition-colors"
          style={{ background: 'rgba(239,68,68,0.12)', color: 'var(--danger)' }}
          onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.2)')}
          onMouseLeave={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.12)')}
        >
          <Trash2 size={14} /> Delete Account
        </button>
      </div>

      {showDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center modal-backdrop">
          <div className="card p-6 w-80 text-center animate-fade-in">
            <AlertTriangle
              size={36}
              className="mx-auto mb-3"
              style={{ color: 'var(--danger)' }}
            />
            <p className="font-semibold text-hi mb-1">Delete your account?</p>
            <p className="text-sm text-lo mb-5">
              This will permanently erase all your posts, stories, and data. Cannot be undone.
            </p>
            <div className="flex gap-3">
              <button
                onClick={() => setShowDelete(false)}
                className="flex-1 py-2 rounded-btn border text-sm"
                style={{ borderColor: 'var(--border)' }}
              >
                Cancel
              </button>
              <button
                onClick={deleteAccount}
                disabled={deleting}
                className="flex-1 py-2 rounded-btn text-white text-sm disabled:opacity-60 flex items-center justify-center gap-1.5"
                style={{ background: 'var(--danger)' }}
              >
                {deleting && <Loader2 size={13} className="animate-spin" />}
                {deleting ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

/* ─── Security Tab ───────────────────────────────────────────────── */
function SecurityTab() {
  const [form, setForm] = useState({
    current_password: '',
    new_password:     '',
    confirm_password: '',
  })
  const [show,   setShow]   = useState({ current: false, new: false, confirm: false })
  const [saving, setSaving] = useState(false)

  const strength = getPasswordStrength(form.new_password)

  async function changePassword() {
    if (form.new_password !== form.confirm_password) {
      toast.error('Passwords do not match')
      return
    }
    setSaving(true)
    try {
      await api.post('/auth/change-password', {
        current_password: form.current_password,
        new_password:     form.new_password,
      })
      toast.success('Password changed!')
      setForm({ current_password: '', new_password: '', confirm_password: '' })
    } catch (err) {
      toast.error(err?.response?.data?.message ?? 'Failed to change password')
    } finally { setSaving(false) }
  }

  return (
    <div className="card p-6">
      <h2 className="font-semibold text-hi mb-5">Change Password</h2>
      <div className="space-y-4 max-w-sm">
        <Field label="Current Password">
          <PasswordInput
            value={form.current_password}
            onChange={(v) => setForm((p) => ({ ...p, current_password: v }))}
            show={show.current}
            onToggle={() => setShow((p) => ({ ...p, current: !p.current }))}
          />
        </Field>

        <Field label="New Password">
          <PasswordInput
            value={form.new_password}
            onChange={(v) => setForm((p) => ({ ...p, new_password: v }))}
            show={show.new}
            onToggle={() => setShow((p) => ({ ...p, new: !p.new }))}
          />
          {form.new_password && (
            <div className="mt-2">
              <div className="h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--surface-high)' }}>
                <div
                  className="h-full rounded-full transition-all duration-300"
                  style={{ width: strength.width, background: strength.color }}
                />
              </div>
              <p className="text-xs mt-1" style={{ color: strength.color }}>{strength.level}</p>
            </div>
          )}
        </Field>

        <Field label="Confirm New Password">
          <PasswordInput
            value={form.confirm_password}
            onChange={(v) => setForm((p) => ({ ...p, confirm_password: v }))}
            show={show.confirm}
            onToggle={() => setShow((p) => ({ ...p, confirm: !p.confirm }))}
          />
          {form.confirm_password && form.new_password !== form.confirm_password && (
            <p className="text-xs mt-1" style={{ color: 'var(--danger)' }}>
              Passwords do not match
            </p>
          )}
        </Field>

        <button
          onClick={changePassword}
          disabled={saving || !form.current_password || !form.new_password}
          className="w-full btn-primary py-2 text-sm disabled:opacity-40 flex items-center justify-center gap-2"
        >
          {saving && <Loader2 size={14} className="animate-spin" />}
          {saving ? 'Saving...' : 'Change Password'}
        </button>
      </div>
    </div>
  )
}

/* ─── Privacy Tab ────────────────────────────────────────────────── */
function PrivacyTab() {
  const { user, setUser } = useAuthStore()
  const [isPrivate, setIsPrivate] = useState(user?.is_private ?? false)
  const [saving,    setSaving]    = useState(false)

  async function toggle() {
    const next = !isPrivate
    setSaving(true)
    try {
      await api.put('/users/me', { is_private: next })
      setIsPrivate(next)
      setUser({ ...user, is_private: next })
      toast.success(next ? 'Account set to private' : 'Account set to public')
    } catch { toast.error('Failed to update') }
    finally { setSaving(false) }
  }

  return (
    <div className="card p-6">
      <h2 className="font-semibold text-hi mb-5">Privacy</h2>
      <div
        className="flex items-start justify-between gap-4 py-4 border-b"
        style={{ borderColor: 'var(--border)' }}
      >
        <div>
          <p className="font-medium text-hi text-sm">Private Account</p>
          <p className="text-xs text-lo mt-1 max-w-xs">
            Only approved followers will see your posts, stories, and highlights.
          </p>
        </div>
        <Toggle checked={isPrivate} onChange={toggle} disabled={saving} />
      </div>
    </div>
  )
}

/* ─── Notifications Tab ──────────────────────────────────────────── */
const NOTIF_EVENTS = [
  { key: 'new_follower',   label: 'New follower' },
  { key: 'post_liked',     label: 'Post liked' },
  { key: 'post_commented', label: 'Post commented' },
  { key: 'story_viewed',   label: 'Story viewed' },
  { key: 'new_message',    label: 'New message' },
]

function NotificationsTab() {
  const [settings, setSettings] = useState(null)
  const [saving,   setSaving]   = useState(false)

  useEffect(() => {
    api.get('/notification-settings')
      .then(({ data }) => setSettings(data ?? {}))
      .catch(() => setSettings({}))
  }, [])

  function toggleSetting(key, channel) {
    setSettings((prev) => ({
      ...prev,
      [key]: { ...(prev[key] ?? {}), [channel]: !(prev[key]?.[channel] ?? false) },
    }))
  }

  async function save() {
    setSaving(true)
    try {
      await api.put('/notification-settings', settings)
      toast.success('Notification settings saved!')
    } catch { toast.error('Failed to save') }
    finally { setSaving(false) }
  }

  if (!settings) {
    return (
      <div className="card p-6 space-y-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex items-center justify-between py-2">
            <div className="skeleton h-4 w-32 rounded" />
            <div className="flex gap-10">
              <div className="skeleton w-10 h-5 rounded-full" />
              <div className="skeleton w-10 h-5 rounded-full" />
            </div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className="card p-6">
      <h2 className="font-semibold text-hi mb-5">Notification Preferences</h2>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr>
              <th className="text-left text-lo font-medium pb-3 text-xs uppercase tracking-wider pr-8">
                Event
              </th>
              <th className="text-center text-lo font-medium pb-3 text-xs uppercase tracking-wider w-24">
                Push
              </th>
              <th className="text-center text-lo font-medium pb-3 text-xs uppercase tracking-wider w-24">
                Email
              </th>
            </tr>
          </thead>
          <tbody>
            {NOTIF_EVENTS.map(({ key, label }) => (
              <tr
                key={key}
                className="border-t"
                style={{ borderColor: 'var(--border)' }}
              >
                <td className="py-3.5 text-hi pr-8">{label}</td>
                <td className="py-3.5 text-center">
                  <Toggle
                    checked={settings[key]?.push ?? false}
                    onChange={() => toggleSetting(key, 'push')}
                    small
                  />
                </td>
                <td className="py-3.5 text-center">
                  <Toggle
                    checked={settings[key]?.email ?? false}
                    onChange={() => toggleSetting(key, 'email')}
                    small
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="flex justify-end mt-5">
        <button
          onClick={save}
          disabled={saving}
          className="btn-primary text-sm px-5 py-2 flex items-center gap-2 disabled:opacity-40"
        >
          {saving && <Loader2 size={14} className="animate-spin" />}
          {saving ? 'Saving...' : 'Save Settings'}
        </button>
      </div>
    </div>
  )
}

/* ─── Devices Tab ────────────────────────────────────────────────── */
function DevicesTab() {
  const [devices,  setDevices]  = useState([])
  const [loading,  setLoading]  = useState(true)
  const [revoking, setRevoking] = useState(null)

  useEffect(() => {
    api.get('/devices')
      .then(({ data }) => setDevices(data.devices ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  async function revoke(id) {
    setRevoking(id)
    try {
      await api.delete(`/devices/${id}`)
      setDevices((prev) => prev.filter((d) => d.id !== id))
      toast.success('Device revoked')
    } catch { toast.error('Failed to revoke device') }
    finally { setRevoking(null) }
  }

  function DeviceIcon({ os }) {
    const mobile = /android|ios|iphone|ipad/i.test(os ?? '')
    return mobile
      ? <Smartphone size={18} style={{ color: 'var(--text-2)' }} />
      : <Monitor size={18} style={{ color: 'var(--text-2)' }} />
  }

  return (
    <div className="card overflow-hidden">
      <div className="px-5 py-4 border-b" style={{ borderColor: 'var(--border)' }}>
        <h2 className="font-semibold text-hi">Active Devices</h2>
        <p className="text-xs text-lo mt-0.5">Sessions currently with access to your account</p>
      </div>

      {loading ? (
        Array.from({ length: 3 }).map((_, i) => (
          <div
            key={i}
            className="flex items-center gap-3 px-5 py-4"
            style={{ borderBottom: i < 2 ? '1px solid var(--border)' : 'none' }}
          >
            <div className="skeleton w-9 h-9 rounded-btn shrink-0" />
            <div className="flex-1 space-y-1.5">
              <div className="skeleton h-3.5 w-36 rounded" />
              <div className="skeleton h-2.5 w-24 rounded" />
            </div>
            <div className="skeleton h-8 w-16 rounded-btn" />
          </div>
        ))
      ) : devices.length === 0 ? (
        <p className="text-center text-lo text-sm py-10">No devices found.</p>
      ) : devices.map((d, idx) => (
        <div
          key={d.id}
          className="flex items-center gap-3 px-5 py-4"
          style={{ borderBottom: idx < devices.length - 1 ? '1px solid var(--border)' : 'none' }}
        >
          <div
            className="w-9 h-9 rounded-btn flex items-center justify-center shrink-0"
            style={{ background: 'var(--surface-high)' }}
          >
            <DeviceIcon os={d.os ?? d.platform} />
          </div>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-hi truncate">
              {d.name ?? d.device_name ?? 'Unknown Device'}
            </p>
            <p className="text-xs text-lo">
              {d.os ?? d.platform ?? 'Unknown OS'}
              {(d.last_active ?? d.updated_at) && (
                <> · Last active {formatRelativeTime(d.last_active ?? d.updated_at)}</>
              )}
            </p>
          </div>
          <button
            onClick={() => revoke(d.id)}
            disabled={revoking === d.id}
            className="text-sm font-medium px-3 py-1.5 rounded-btn transition-colors disabled:opacity-50 flex items-center gap-1.5"
            style={{ color: 'var(--danger)', background: 'rgba(239,68,68,0.1)' }}
            onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.2)')}
            onMouseLeave={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.1)')}
          >
            {revoking === d.id ? <><Loader2 size={13} className="animate-spin" />Revoking…</> : 'Revoke'}
          </button>
        </div>
      ))}
    </div>
  )
}

/* ─── Shared ─────────────────────────────────────────────────────── */
function Field({ label, children }) {
  return (
    <div className="space-y-1.5">
      <label className="block text-xs font-medium text-lo">{label}</label>
      {children}
    </div>
  )
}

function PasswordInput({ value, onChange, show, onToggle }) {
  return (
    <div className="relative">
      <input
        type={show ? 'text' : 'password'}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="input-base text-sm pr-10"
        autoComplete="off"
      />
      <button
        type="button"
        onClick={onToggle}
        className="absolute right-3 top-1/2 -translate-y-1/2 text-lo hover:text-hi transition-colors"
      >
        {show ? <EyeOff size={16} /> : <Eye size={16} />}
      </button>
    </div>
  )
}

function Toggle({ checked, onChange, disabled, small }) {
  const width  = small ? 'w-9'   : 'w-11'
  const height = small ? 'h-5'   : 'h-6'
  const dot    = small ? 'w-3.5 h-3.5' : 'w-4 h-4'
  const tx     = small ? 'translate-x-4' : 'translate-x-5'

  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={onChange}
      disabled={disabled}
      className={`relative inline-flex items-center ${height} ${width} rounded-full transition-colors duration-200 focus:outline-none disabled:opacity-50`}
      style={{
        background:  checked ? 'var(--accent)' : 'var(--surface-high)',
        border:      '1px solid var(--border)',
      }}
    >
      <span
        className={`inline-block ${dot} rounded-full bg-white shadow-sm transition-transform duration-200 ${checked ? tx : 'translate-x-0.5'}`}
      />
    </button>
  )
}

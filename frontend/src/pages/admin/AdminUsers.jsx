import { useState, useEffect, useCallback } from 'react'
import { Search, X, ChevronLeft, ChevronRight, Lock, CheckCircle2, Eye, Ban, Trash2, ExternalLink } from 'lucide-react'
import { toast } from 'sonner'
import { Link } from 'react-router-dom'
import api from '../../lib/api'
import Avatar from '../../components/shared/Avatar'
import { formatRelativeTime, formatCount } from '../../lib/utils'
import { ADMIN_ACCENT } from './AdminLayout'

const PAGE = 20

export default function AdminUsers() {
  const [users,   setUsers]   = useState([])
  const [total,   setTotal]   = useState(0)
  const [page,    setPage]    = useState(1)
  const [loading, setLoading] = useState(true)

  const [search,   setSearch]   = useState('')
  const [verified, setVerified] = useState('All')
  const [privacy,  setPrivacy]  = useState('All')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo,   setDateTo]   = useState('')

  const [drawer,   setDrawer]   = useState(null)
  const [confirm,  setConfirm]  = useState(null) // { type: 'suspend'|'delete', user }

  const load = useCallback(async (p = page) => {
    setLoading(true)
    try {
      const params = { page: p, limit: PAGE }
      if (search.trim())      params.q        = search.trim()
      if (verified !== 'All') params.verified  = verified === 'Yes'
      if (privacy  !== 'All') params.is_private = privacy === 'Private'
      if (dateFrom)           params.from      = dateFrom
      if (dateTo)             params.to        = dateTo

      const { data } = await api.get('/admin/users', { params })
      setUsers(data.users ?? [])
      setTotal(data.total ?? 0)
    } catch { toast.error('Failed to load users') }
    finally { setLoading(false) }
  }, [page, search, verified, privacy, dateFrom, dateTo]) // eslint-disable-line

  useEffect(() => { load(page) }, [page]) // eslint-disable-line
  function applyFilters() { setPage(1); load(1) }

  async function suspendUser(user) {
    try {
      await api.put(`/admin/users/${user.id}/suspend`)
      toast.success(`@${user.username} suspended`)
      load(page)
      setDrawer(null)
    } catch { toast.error('Failed to suspend') }
    setConfirm(null)
  }

  async function deleteUser(user) {
    try {
      await api.delete(`/admin/users/${user.id}`)
      toast.success(`@${user.username} deleted`)
      load(page)
      setDrawer(null)
    } catch { toast.error('Failed to delete') }
    setConfirm(null)
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE))

  return (
    <div className="p-6">
      <h1 className="text-xl font-bold text-white mb-5">Users</h1>

      {/* Filter bar */}
      <div
        className="rounded-card p-4 mb-4 flex flex-wrap items-end gap-3"
        style={{ background: '#161616', border: '1px solid #1F1F1F' }}
      >
        <div className="flex-1 min-w-48 relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" style={{ color: '#52525B' }} />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
            placeholder="Search username / email / name…"
            className="w-full pl-9 pr-3 py-2 text-sm rounded-btn outline-none transition-colors"
            style={{
              background: '#1A1A1A', border: '1px solid #2E2E2E',
              color: '#fff', caretColor: ADMIN_ACCENT,
            }}
            onFocus={(e) => (e.target.style.borderColor = ADMIN_ACCENT)}
            onBlur={(e)  => (e.target.style.borderColor = '#2E2E2E')}
          />
        </div>
        <AdminSelect label="Verified" value={verified} onChange={setVerified} options={['All', 'Yes', 'No']} />
        <AdminSelect label="Privacy"  value={privacy}  onChange={setPrivacy}  options={['All', 'Public', 'Private']} />
        <div className="flex items-end gap-2">
          <AdminDateInput label="From" value={dateFrom} onChange={setDateFrom} />
          <AdminDateInput label="To"   value={dateTo}   onChange={setDateTo} />
        </div>
        <button onClick={applyFilters} className="admin-btn-primary px-4 py-2 text-sm rounded-btn" style={{ background: ADMIN_ACCENT, color: '#fff' }}>
          Apply
        </button>
        <button
          onClick={() => { setSearch(''); setVerified('All'); setPrivacy('All'); setDateFrom(''); setDateTo(''); setPage(1); }}
          className="text-xs px-3 py-2 rounded-btn transition-colors"
          style={{ color: '#71717A', background: '#1A1A1A' }}
        >
          Reset
        </button>
      </div>

      {/* Table */}
      <div className="rounded-card overflow-hidden" style={{ background: '#161616', border: '1px solid #1F1F1F' }}>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: '1px solid #1F1F1F' }}>
                {['User', 'Email', 'Full Name', 'Badges', 'Joined', 'Actions'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-semibold" style={{ color: '#52525B' }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading
                ? Array.from({ length: 8 }).map((_, i) => <RowSkeleton key={i} cols={6} />)
                : users.map((u, idx) => (
                  <tr
                    key={u.id}
                    className="cursor-pointer transition-colors"
                    style={{ borderBottom: idx < users.length - 1 ? '1px solid #1F1F1F' : 'none' }}
                    onClick={() => setDrawer(u)}
                    onMouseEnter={(e) => (e.currentTarget.style.background = '#1A1A1A')}
                    onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2.5">
                        <Avatar src={u.avatar_url} name={u.username} size={32} />
                        <span className="text-white font-medium">@{u.username}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3" style={{ color: '#A1A1AA' }}>{u.email}</td>
                    <td className="px-4 py-3" style={{ color: '#A1A1AA' }}>{u.full_name ?? '—'}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        {u.is_verified  && <Badge color={ADMIN_ACCENT} icon={<CheckCircle2 size={10} />} label="Verified" />}
                        {u.is_private   && <Badge color="#71717A"      icon={<Lock size={10} />}          label="Private" />}
                        {u.is_suspended && <Badge color="#EF4444"      icon={<Ban size={10} />}           label="Suspended" />}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-xs" style={{ color: '#52525B' }}>
                      {u.created_at ? new Date(u.created_at).toLocaleDateString() : '—'}
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
                        <ActionBtn icon={<Eye size={13} />} onClick={() => setDrawer(u)} title="View" />
                        <ActionBtn
                          icon={<Ban size={13} />}
                          onClick={() => setConfirm({ type: 'suspend', user: u })}
                          title={u.is_suspended ? 'Unsuspend' : 'Suspend'}
                          color="#F59E0B"
                        />
                        <ActionBtn
                          icon={<Trash2 size={13} />}
                          onClick={() => setConfirm({ type: 'delete', user: u })}
                          title="Delete"
                          color="#EF4444"
                        />
                      </div>
                    </td>
                  </tr>
                ))
              }
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {!loading && (
          <div
            className="flex items-center justify-between px-4 py-3 border-t"
            style={{ borderColor: '#1F1F1F' }}
          >
            <p className="text-xs" style={{ color: '#52525B' }}>
              {total} users · Page {page} of {totalPages}
            </p>
            <Pagination current={page} total={totalPages} onChange={setPage} />
          </div>
        )}
      </div>

      {/* User detail drawer */}
      {drawer && (
        <UserDrawer
          user={drawer}
          onClose={() => setDrawer(null)}
          onSuspend={() => setConfirm({ type: 'suspend', user: drawer })}
          onDelete={() => setConfirm({ type: 'delete', user: drawer })}
        />
      )}

      {/* Confirm modal */}
      {confirm && (
        <ConfirmModal
          type={confirm.type}
          user={confirm.user}
          onCancel={() => setConfirm(null)}
          onConfirm={() => confirm.type === 'delete' ? deleteUser(confirm.user) : suspendUser(confirm.user)}
        />
      )}
    </div>
  )
}

function UserDrawer({ user, onClose, onSuspend, onDelete }) {
  return (
    <div className="fixed inset-0 z-40 flex" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="ml-auto flex flex-col h-full animate-slide-in-right overflow-y-auto" style={{ width: 400, background: '#161616', borderLeft: '1px solid #1F1F1F' }}>
        <div className="flex items-center justify-between px-5 py-4 border-b sticky top-0" style={{ borderColor: '#1F1F1F', background: '#161616' }}>
          <h3 className="font-semibold text-white">User Detail</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white p-1"><X size={18} /></button>
        </div>
        <div className="p-5 flex-1">
          <div className="flex flex-col items-center gap-3 mb-6">
            <Avatar src={user.avatar_url} name={user.full_name ?? user.username} size={80} />
            <div className="text-center">
              <p className="font-bold text-white text-lg">@{user.username}</p>
              <p className="text-sm" style={{ color: '#71717A' }}>{user.email}</p>
            </div>
            <div className="flex gap-2">
              {user.is_verified  && <Badge color={ADMIN_ACCENT} icon={<CheckCircle2 size={10} />} label="Verified" />}
              {user.is_private   && <Badge color="#71717A"      icon={<Lock size={10} />}          label="Private" />}
              {user.is_suspended && <Badge color="#EF4444"      icon={<Ban size={10} />}           label="Suspended" />}
            </div>
          </div>

          <div className="grid grid-cols-3 gap-3 mb-5">
            {[
              { label: 'Posts',     value: user.posts_count ?? 0 },
              { label: 'Followers', value: user.followers_count ?? 0 },
              { label: 'Following', value: user.following_count ?? 0 },
            ].map(({ label, value }) => (
              <div key={label} className="text-center rounded-btn py-3" style={{ background: '#1A1A1A' }}>
                <p className="text-lg font-bold text-white">{formatCount(value)}</p>
                <p className="text-xs" style={{ color: '#71717A' }}>{label}</p>
              </div>
            ))}
          </div>

          {user.bio && (
            <div className="mb-4 p-3 rounded-btn text-sm" style={{ background: '#1A1A1A', color: '#A1A1AA' }}>
              {user.bio}
            </div>
          )}

          <div className="space-y-2 text-xs mb-6">
            <DetailRow label="Full Name" value={user.full_name ?? '—'} />
            <DetailRow label="Joined"    value={user.created_at ? new Date(user.created_at).toLocaleDateString() : '—'} />
            <DetailRow label="Last Active" value={user.last_active ? formatRelativeTime(user.last_active) : '—'} />
            <DetailRow label="Status"    value={user.is_suspended ? 'Suspended' : 'Active'} highlight={user.is_suspended ? '#EF4444' : ADMIN_ACCENT} />
          </div>

          <div className="space-y-2">
            <Link
              to={`/profile/${user.id}`}
              target="_blank"
              className="w-full flex items-center justify-center gap-2 py-2 rounded-btn text-sm transition-colors"
              style={{ border: '1px solid #2E2E2E', color: '#A1A1AA' }}
              onMouseEnter={(e) => (e.currentTarget.style.color = '#fff')}
              onMouseLeave={(e) => (e.currentTarget.style.color = '#A1A1AA')}
            >
              <ExternalLink size={14} /> View Public Profile
            </Link>
            <button
              onClick={onSuspend}
              className="w-full py-2 rounded-btn text-sm transition-colors"
              style={{ background: 'rgba(245,158,11,0.1)', color: '#F59E0B' }}
            >
              {user.is_suspended ? 'Unsuspend Account' : 'Suspend Account'}
            </button>
            <button
              onClick={onDelete}
              className="w-full py-2 rounded-btn text-sm transition-colors"
              style={{ background: 'rgba(239,68,68,0.1)', color: '#EF4444' }}
            >
              Delete Account
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function ConfirmModal({ type, user, onCancel, onConfirm }) {
  const [typed, setTyped] = useState('')
  const isDelete = type === 'delete'
  const valid = isDelete ? typed === user.username : true

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.8)' }}>
      <div className="rounded-card p-6 w-80 animate-fade-in" style={{ background: '#1A1A1A', border: '1px solid #2E2E2E' }}>
        <p className="font-semibold text-white mb-2">
          {isDelete ? 'Delete user?' : `${user.is_suspended ? 'Unsuspend' : 'Suspend'} @${user.username}?`}
        </p>
        <p className="text-sm mb-4" style={{ color: '#71717A' }}>
          {isDelete ? `This will permanently delete @${user.username} and all their data.` : 'This will restrict the user\'s access to the platform.'}
        </p>
        {isDelete && (
          <div className="mb-4">
            <p className="text-xs mb-1.5" style={{ color: '#71717A' }}>Type <strong className="text-white">{user.username}</strong> to confirm:</p>
            <input
              type="text"
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              className="w-full px-3 py-2 text-sm rounded-btn outline-none"
              style={{ background: '#111', border: '1px solid #2E2E2E', color: '#fff' }}
              placeholder={user.username}
              autoFocus
            />
          </div>
        )}
        <div className="flex gap-3">
          <button onClick={onCancel} className="flex-1 py-2 rounded-btn text-sm" style={{ border: '1px solid #2E2E2E', color: '#A1A1AA' }}>
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={!valid}
            className="flex-1 py-2 rounded-btn text-sm text-white disabled:opacity-40"
            style={{ background: isDelete ? '#EF4444' : '#F59E0B' }}
          >
            {isDelete ? 'Delete' : (user.is_suspended ? 'Unsuspend' : 'Suspend')}
          </button>
        </div>
      </div>
    </div>
  )
}

/* ─── Shared admin UI primitives ─────────────────────────────────── */
function Badge({ color, icon, label }) {
  return (
    <span
      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-medium"
      style={{ background: `${color}20`, color }}
    >
      {icon}{label}
    </span>
  )
}

function ActionBtn({ icon, onClick, title, color }) {
  return (
    <button
      onClick={onClick}
      title={title}
      className="w-6 h-6 rounded flex items-center justify-center transition-colors"
      style={{ color: color ?? '#71717A', background: '#1F1F1F' }}
      onMouseEnter={(e) => (e.currentTarget.style.color = color ?? '#fff')}
      onMouseLeave={(e) => (e.currentTarget.style.color = color ?? '#71717A')}
    >
      {icon}
    </button>
  )
}

function DetailRow({ label, value, highlight }) {
  return (
    <div className="flex items-center justify-between py-1.5 border-b" style={{ borderColor: '#1F1F1F' }}>
      <span style={{ color: '#52525B' }}>{label}</span>
      <span style={{ color: highlight ?? '#A1A1AA' }}>{value}</span>
    </div>
  )
}

function Pagination({ current, total, onChange }) {
  const pages = []
  for (let i = Math.max(1, current - 2); i <= Math.min(total, current + 2); i++) pages.push(i)

  return (
    <div className="flex items-center gap-1">
      <PageBtn disabled={current === 1} onClick={() => onChange(current - 1)} icon={<ChevronLeft size={14} />} />
      {pages[0] > 1 && <PageBtn label="1" onClick={() => onChange(1)} />}
      {pages[0] > 2 && <span className="px-1" style={{ color: '#52525B' }}>…</span>}
      {pages.map((p) => (
        <PageBtn key={p} label={p} active={p === current} onClick={() => onChange(p)} />
      ))}
      {pages[pages.length - 1] < total - 1 && <span className="px-1" style={{ color: '#52525B' }}>…</span>}
      {pages[pages.length - 1] < total && <PageBtn label={total} onClick={() => onChange(total)} />}
      <PageBtn disabled={current === total} onClick={() => onChange(current + 1)} icon={<ChevronRight size={14} />} />
    </div>
  )
}

function PageBtn({ label, icon, active, disabled, onClick }) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className="min-w-[28px] h-7 px-1.5 rounded text-xs font-medium transition-colors disabled:opacity-30"
      style={{
        background: active ? ADMIN_ACCENT : '#1A1A1A',
        color:      active ? '#fff' : '#71717A',
      }}
    >
      {icon ?? label}
    </button>
  )
}

function AdminSelect({ label, value, onChange, options }) {
  return (
    <div>
      <p className="text-[10px] mb-1" style={{ color: '#52525B' }}>{label}</p>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="px-3 py-2 text-sm rounded-btn outline-none cursor-pointer"
        style={{ background: '#1A1A1A', border: '1px solid #2E2E2E', color: '#A1A1AA' }}
      >
        {options.map((o) => <option key={o} value={o}>{o}</option>)}
      </select>
    </div>
  )
}

function AdminDateInput({ label, value, onChange }) {
  return (
    <div>
      <p className="text-[10px] mb-1" style={{ color: '#52525B' }}>{label}</p>
      <input
        type="date"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="px-3 py-2 text-sm rounded-btn outline-none"
        style={{ background: '#1A1A1A', border: '1px solid #2E2E2E', color: '#A1A1AA' }}
      />
    </div>
  )
}

function RowSkeleton({ cols }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} className="px-4 py-3.5">
          <div className="skeleton h-3.5 rounded" style={{ width: i === 0 ? 120 : i === cols - 1 ? 80 : 100 }} />
        </td>
      ))}
    </tr>
  )
}

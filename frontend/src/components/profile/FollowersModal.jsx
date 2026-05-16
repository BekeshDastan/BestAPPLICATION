import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { X, Search, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import Avatar from '../shared/Avatar'

function FollowBtn({ user }) {
  const { user: me } = useAuthStore()
  const [following, setFollowing] = useState(user.is_following ?? false)
  const [loading,   setLoading]   = useState(false)
  if (user.id === me?.id) return null

  async function toggle() {
    setLoading(true)
    try {
      if (following) await api.delete(`/users/${user.id}/follow`)
      else           await api.post(`/users/${user.id}/follow`)
      setFollowing((v) => !v)
    } catch { toast.error('Action failed') }
    finally { setLoading(false) }
  }

  return (
    <button
      onClick={toggle}
      disabled={loading}
      className="text-xs font-semibold px-3 py-1.5 rounded-btn transition-all disabled:opacity-40"
      style={{
        background: following ? 'transparent' : 'var(--accent)',
        color:      following ? 'var(--text-2)' : '#fff',
        border:     following ? '1px solid var(--border)' : 'none',
        boxShadow:  following ? 'none' : 'var(--shadow-accent)',
      }}
    >
      {loading
        ? <Loader2 size={12} className="animate-spin" />
        : following ? 'Following' : 'Follow'}
    </button>
  )
}

function RowSkeleton() {
  return (
    <div className="flex items-center gap-3 px-4 py-3">
      <div className="skeleton w-10 h-10 rounded-full shrink-0" />
      <div className="flex-1 space-y-1.5">
        <div className="skeleton w-28 h-3 rounded" />
        <div className="skeleton w-20 h-2.5 rounded" />
      </div>
      <div className="skeleton w-16 h-7 rounded-btn" />
    </div>
  )
}

export default function FollowersModal({ userId, type, onClose }) {
  const [all,     setAll]     = useState([])
  const [query,   setQuery]   = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const endpoint = type === 'followers'
      ? `/users/${userId}/followers`
      : `/users/${userId}/following`
    api.get(endpoint, { params: { limit: 200 } })
      .then(({ data }) => setAll(data.users ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [userId, type])

  useEffect(() => {
    const handler = (e) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const filtered = query.trim()
    ? all.filter((u) =>
        u.username?.toLowerCase().includes(query.toLowerCase()) ||
        u.full_name?.toLowerCase().includes(query.toLowerCase()),
      )
    : all

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 modal-backdrop"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="card w-full max-w-sm max-h-[80vh] flex flex-col overflow-hidden animate-fade-in"
      >
        {/* Header */}
        <div
          className="flex items-center justify-between px-4 py-3 border-b shrink-0"
          style={{ borderColor: 'var(--border)' }}
        >
          <h3 className="font-semibold text-hi capitalize">{type}</h3>
          <button
            onClick={onClose}
            className="p-1 rounded-btn text-lo hover:text-hi transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        {/* Search */}
        <div
          className="px-3 py-2 border-b shrink-0"
          style={{ borderColor: 'var(--border)' }}
        >
          <div className="relative">
            <Search
              size={14}
              className="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none"
              style={{ color: 'var(--text-2)' }}
            />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search…"
              className="input-base pl-9 py-1.5 text-sm"
            />
          </div>
        </div>

        {/* List */}
        <div className="flex-1 overflow-y-auto">
          {loading
            ? Array.from({ length: 5 }).map((_, i) => <RowSkeleton key={i} />)
            : filtered.length === 0
              ? (
                <p className="text-sm text-lo text-center py-10">
                  {query ? 'No results' : 'Nobody yet.'}
                </p>
              )
              : filtered.map((u) => (
                  <div
                    key={u.id}
                    className="flex items-center gap-3 px-4 py-2.5 hover:bg-elevated transition-colors"
                  >
                    <Link to={`/profile/${u.id}`} onClick={onClose} className="shrink-0">
                      <Avatar
                        src={u.avatar_url}
                        name={u.full_name ?? u.username}
                        size={40}
                      />
                    </Link>
                    <div className="flex-1 min-w-0">
                      <Link
                        to={`/profile/${u.id}`}
                        onClick={onClose}
                        className="text-sm font-semibold text-hi hover:underline block truncate"
                      >
                        {u.full_name ?? u.username}
                      </Link>
                      <span className="text-xs text-lo">@{u.username}</span>
                    </div>
                    <FollowBtn user={u} />
                  </div>
                ))
          }
        </div>
      </div>
    </div>
  )
}

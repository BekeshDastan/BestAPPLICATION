import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../shared/Avatar'
import { formatRelativeTime } from '../../lib/utils'

function SuggestedUser({ user, onFollowed }) {
  const [following, setFollowing] = useState(false)
  const [loading,   setLoading]   = useState(false)

  async function toggle() {
    setLoading(true)
    try {
      if (following) {
        await api.delete(`/users/${user.id}/follow`)
      } else {
        await api.post(`/users/${user.id}/follow`)
      }
      setFollowing((v) => !v)
      if (!following) onFollowed?.(user.id)
    } catch {
      toast.error('Action failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex items-center gap-3">
      <Link to={`/profile/${user.id}`} className="shrink-0">
        <Avatar
          src={user.avatar_url}
          name={user.full_name ?? user.username}
          size={36}
        />
      </Link>
      <div className="flex-1 min-w-0">
        <Link
          to={`/profile/${user.id}`}
          className="text-xs font-semibold text-hi hover:underline block truncate"
        >
          {user.full_name ?? user.username}
        </Link>
        <span className="text-[11px] text-lo">@{user.username}</span>
      </div>
      <button
        onClick={toggle}
        disabled={loading}
        className="text-xs font-semibold shrink-0 transition-colors disabled:opacity-40"
        style={{ color: following ? 'var(--text-2)' : 'var(--accent)' }}
      >
        {loading
          ? <Loader2 size={12} className="animate-spin" />
          : following ? 'Following' : 'Follow'
        }
      </button>
    </div>
  )
}

function PanelSection({ title, children }) {
  return (
    <div className="card p-4">
      <p className="text-xs font-semibold text-lo uppercase tracking-wider mb-3">
        {title}
      </p>
      {children}
    </div>
  )
}

export default function RightPanel() {
  const [suggestions, setSuggestions]   = useState([])
  const [activity,    setActivity]      = useState([])
  const [loadingSug,  setLoadingSug]    = useState(true)
  const [loadingAct,  setLoadingAct]    = useState(true)

  useEffect(() => {
    // Suggestions — try /users/suggestions endpoint (may not exist, fallback gracefully)
    api
      .get('/users/suggestions', { params: { limit: 5 } })
      .then(({ data }) => setSuggestions(data.users ?? []))
      .catch(() => setSuggestions([]))
      .finally(() => setLoadingSug(false))

    // Recent activity from notifications
    api
      .get('/notifications', { params: { limit: 5 } })
      .then(({ data }) => setActivity(data.notifications ?? []))
      .catch(() => setActivity([]))
      .finally(() => setLoadingAct(false))
  }, [])

  return (
    <aside className="hidden lg:flex flex-col w-72 shrink-0 gap-4 pt-6 pr-4 pb-8">
      {/* Suggestions */}
      <PanelSection title="Suggested for you">
        {loadingSug ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-2">
                <div className="skeleton w-9 h-9 rounded-full" />
                <div className="flex-1 space-y-1.5">
                  <div className="skeleton w-24 h-2.5 rounded" />
                  <div className="skeleton w-16 h-2 rounded" />
                </div>
                <div className="skeleton w-12 h-5 rounded" />
              </div>
            ))}
          </div>
        ) : suggestions.length === 0 ? (
          <p className="text-xs text-lo">No suggestions right now.</p>
        ) : (
          <div className="space-y-3">
            {suggestions.map((u) => (
              <SuggestedUser
                key={u.id}
                user={u}
                onFollowed={(id) =>
                  setSuggestions((s) => s.filter((x) => x.id !== id))
                }
              />
            ))}
          </div>
        )}
      </PanelSection>

      {/* Recent activity */}
      <PanelSection title="Recent activity">
        {loadingAct ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="skeleton w-full h-8 rounded" />
            ))}
          </div>
        ) : activity.length === 0 ? (
          <p className="text-xs text-lo">No recent activity.</p>
        ) : (
          <div className="space-y-2.5">
            {activity.map((n) => (
              <div key={n.id} className="flex items-start gap-2">
                <div
                  className="w-1.5 h-1.5 rounded-full shrink-0 mt-1.5"
                  style={{ background: 'var(--accent)' }}
                />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-hi leading-snug line-clamp-2">
                    {n.message ?? n.type}
                  </p>
                  <span className="text-[10px] text-lo">
                    {formatRelativeTime(n.created_at)}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </PanelSection>
    </aside>
  )
}

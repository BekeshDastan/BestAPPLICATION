import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Trash2, CheckCheck, ChevronDown } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../../components/shared/Avatar'
import { formatRelativeTime } from '../../lib/utils'

const FILTERS = ['All', 'Likes', 'Comments', 'Follows', 'Mentions', 'Stories']
const PAGE = 20

const TYPE_META = {
  like:            { icon: '❤️', label: 'liked your post' },
  post_like:       { icon: '❤️', label: 'liked your post' },
  comment:         { icon: '💬', label: 'commented on your post' },
  post_comment:    { icon: '💬', label: 'commented on your post' },
  follow:          { icon: '👤', label: 'started following you' },
  mention:         { icon: '@',  label: 'mentioned you' },
  story_reply:     { icon: '📖', label: 'replied to your story' },
  story_view:      { icon: '📖', label: 'viewed your story' },
  message:         { icon: '💌', label: 'sent you a message' },
}

function getMeta(type) {
  return TYPE_META[type] ?? { icon: '🔔', label: type ?? 'notification' }
}

function getNavTarget(notif) {
  const t = notif.type ?? ''
  if (t.includes('follow'))  return `/profile/${notif.actor_id}`
  if (t.includes('story'))   return `/stories?user=${notif.actor_id}`
  if (t === 'message')       return '/chat'
  if (notif.post_id)         return `/profile/${notif.actor_id}`
  return `/profile/${notif.actor_id}`
}

function filterParam(f) {
  const map = { Likes: 'like', Comments: 'comment', Follows: 'follow', Mentions: 'mention', Stories: 'story' }
  return map[f] ?? null
}

export default function NotificationsPage() {
  const navigate = useNavigate()

  const [notifs,      setNotifs]      = useState([])
  const [cursor,      setCursor]      = useState(null)
  const [hasMore,     setHasMore]     = useState(true)
  const [loading,     setLoading]     = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [filter,      setFilter]      = useState('All')
  const [showFilter,  setShowFilter]  = useState(false)

  const sentinelRef = useRef(null)
  const filterRef   = useRef(null)

  /* ── Close filter dropdown on outside click ──────────────────── */
  useEffect(() => {
    function handler(e) {
      if (!filterRef.current?.contains(e.target)) setShowFilter(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  /* ── Load ────────────────────────────────────────────────────── */
  const load = useCallback(async (c = null, f = filter) => {
    if (c) setLoadingMore(true)
    else   setLoading(true)
    try {
      const params = { limit: PAGE }
      if (c) params.cursor = c
      const fp = filterParam(f)
      if (fp) params.type = fp
      const { data } = await api.get('/notifications', { params })
      const fresh = data.notifications ?? []
      setNotifs((prev) => (c ? [...prev, ...fresh] : fresh))
      setCursor(data.next_cursor ?? null)
      setHasMore(!!data.next_cursor && fresh.length === PAGE)
    } catch {
      setHasMore(false)
    } finally {
      c ? setLoadingMore(false) : setLoading(false)
    }
  }, [filter]) // eslint-disable-line

  useEffect(() => {
    setNotifs([])
    setCursor(null)
    setHasMore(true)
    load(null, filter)
  }, [filter]) // eslint-disable-line

  /* ── Infinite scroll ─────────────────────────────────────────── */
  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting && hasMore && !loadingMore) load(cursor) },
      { threshold: 0.8 },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [cursor, hasMore, loadingMore, load])

  /* ── Actions ─────────────────────────────────────────────────── */
  async function markRead(id) {
    try {
      await api.put(`/notifications/${id}/read`)
      setNotifs((prev) => prev.map((n) => n.id === id ? { ...n, is_read: true } : n))
    } catch {}
  }

  async function markAllRead() {
    try {
      await api.put('/notifications/read-all')
      setNotifs((prev) => prev.map((n) => ({ ...n, is_read: true })))
      toast.success('All marked as read')
    } catch { toast.error('Failed to mark all as read') }
  }

  async function deleteNotif(id, e) {
    e.stopPropagation()
    try {
      await api.delete(`/notifications/${id}`)
      setNotifs((prev) => prev.filter((n) => n.id !== id))
    } catch { toast.error('Failed to delete') }
  }

  function handleClick(notif) {
    if (!notif.is_read) markRead(notif.id)
    navigate(getNavTarget(notif))
  }

  const unread = notifs.filter((n) => !n.is_read).length

  return (
    <div className="max-w-2xl mx-auto px-4 py-6">
      {/* Header */}
      <div className="flex items-start justify-between mb-5">
        <div>
          <h1 className="text-2xl font-bold text-hi">Notifications</h1>
          {unread > 0 && <p className="text-sm text-lo mt-0.5">{unread} unread</p>}
        </div>
        <div className="flex items-center gap-2 mt-1">
          <button
            onClick={markAllRead}
            className="flex items-center gap-1.5 text-sm font-medium px-3 py-1.5 rounded-btn hover:bg-elevated transition-colors"
            style={{ color: 'var(--accent)' }}
          >
            <CheckCheck size={15} /> Mark all read
          </button>

          {/* Filter dropdown */}
          <div className="relative" ref={filterRef}>
            <button
              onClick={() => setShowFilter((v) => !v)}
              className="flex items-center gap-1.5 text-sm px-3 py-1.5 rounded-btn border hover:bg-elevated transition-colors"
              style={{ borderColor: 'var(--border)', color: 'var(--text-2)' }}
            >
              {filter} <ChevronDown size={13} />
            </button>
            {showFilter && (
              <div
                className="absolute right-0 top-full mt-1 z-20 rounded-card overflow-hidden animate-fade-in"
                style={{
                  background: 'var(--surface)',
                  border: '1px solid var(--border)',
                  boxShadow: 'var(--shadow-card)',
                  minWidth: 140,
                }}
              >
                {FILTERS.map((f) => (
                  <button
                    key={f}
                    onClick={() => { setFilter(f); setShowFilter(false) }}
                    className="w-full text-left px-3 py-2 text-sm hover:bg-elevated transition-colors"
                    style={{ color: f === filter ? 'var(--accent)' : 'var(--text-1)' }}
                  >
                    {f}
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Filter chips */}
      <div className="flex gap-2 mb-4 overflow-x-auto pb-1" style={{ scrollbarWidth: 'none' }}>
        {FILTERS.map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className="shrink-0 px-3 py-1.5 rounded-full text-xs font-medium transition-all"
            style={{
              background: f === filter ? 'var(--accent)' : 'var(--surface-high)',
              color:      f === filter ? '#fff' : 'var(--text-2)',
            }}
          >
            {f}
          </button>
        ))}
      </div>

      {/* List */}
      <div className="card overflow-hidden">
        {loading ? (
          Array.from({ length: 6 }).map((_, i) => <NotifSkeleton key={i} last={i === 5} />)
        ) : notifs.length === 0 ? (
          <div className="py-16 text-center">
            <p className="text-4xl mb-3">🔔</p>
            <p className="font-semibold text-hi mb-1">No notifications yet</p>
            <p className="text-sm text-lo">
              When someone interacts with your content, it'll show here.
            </p>
          </div>
        ) : notifs.map((n, idx) => (
          <NotifRow
            key={n.id}
            notif={n}
            isLast={idx === notifs.length - 1}
            onClick={() => handleClick(n)}
            onDelete={(e) => deleteNotif(n.id, e)}
          />
        ))}
      </div>

      <div ref={sentinelRef} className="h-4 mt-2" />

      {loadingMore && (
        <div className="card overflow-hidden mt-0">
          {Array.from({ length: 3 }).map((_, i) => <NotifSkeleton key={i} last={i === 2} />)}
        </div>
      )}

      {!hasMore && notifs.length > 0 && (
        <p className="text-center text-xs text-lo py-6">All caught up ✓</p>
      )}
    </div>
  )
}

/* ─── NotifRow ───────────────────────────────────────────────────── */
function NotifRow({ notif, isLast, onClick, onDelete }) {
  const [hovered, setHovered] = useState(false)
  const meta = getMeta(notif.type)

  return (
    <div
      className="flex items-start gap-3 px-4 py-3.5 cursor-pointer transition-colors relative"
      style={{
        background:   notif.is_read ? 'transparent' : 'var(--accent-glow)',
        borderBottom: isLast ? 'none' : '1px solid var(--border)',
      }}
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Avatar + type badge */}
      <div className="relative shrink-0">
        <Avatar
          src={notif.actor?.avatar_url}
          name={notif.actor?.username ?? notif.actor_id}
          size={44}
        />
        <span
          className="absolute -bottom-0.5 -right-0.5 w-5 h-5 rounded-full flex items-center justify-center text-[11px] leading-none"
          style={{ background: 'var(--surface)', border: '1.5px solid var(--bg)' }}
        >
          {meta.icon}
        </span>
      </div>

      {/* Text */}
      <div className="flex-1 min-w-0 pr-6">
        <p className="text-sm text-hi leading-snug">
          <span className="font-semibold">@{notif.actor?.username ?? 'someone'}</span>{' '}
          <span className="text-lo">{notif.message ?? meta.label}</span>
          {notif.content_preview && (
            <span className="font-medium text-hi ml-1">"{notif.content_preview}"</span>
          )}
        </p>
        <p className="text-xs text-lo mt-1">{formatRelativeTime(notif.created_at)}</p>
      </div>

      {/* Unread dot */}
      {!notif.is_read && (
        <span
          className="absolute right-4 top-1/2 -translate-y-1/2 w-2 h-2 rounded-full"
          style={{ background: 'var(--accent)' }}
        />
      )}

      {/* Delete on hover */}
      {hovered && (
        <button
          onClick={onDelete}
          className="absolute right-4 top-1/2 -translate-y-1/2 p-1.5 rounded-btn transition-colors"
          style={{ background: 'var(--surface-high)', color: 'var(--text-2)' }}
          onMouseEnter={(e) => (e.currentTarget.style.color = 'var(--danger)')}
          onMouseLeave={(e) => (e.currentTarget.style.color = 'var(--text-2)')}
        >
          <Trash2 size={13} />
        </button>
      )}
    </div>
  )
}

function NotifSkeleton({ last }) {
  return (
    <div
      className="flex items-start gap-3 px-4 py-3.5"
      style={{ borderBottom: last ? 'none' : '1px solid var(--border)' }}
    >
      <div className="skeleton w-11 h-11 rounded-full shrink-0" />
      <div className="flex-1 space-y-1.5">
        <div className="skeleton h-3.5 w-64 rounded" />
        <div className="skeleton h-2.5 w-16 rounded" />
      </div>
    </div>
  )
}

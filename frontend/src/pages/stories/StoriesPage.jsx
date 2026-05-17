import { useEffect, useRef, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { X, Trash2, Eye, Send, ChevronLeft, ChevronRight } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import Avatar from '../../components/shared/Avatar'
import { formatRelativeTime } from '../../lib/utils'

const STORY_DURATION = 5000
const TICK = 50
const EMOJIS = ['❤️', '😂', '😮', '😢', '👏', '🔥']

function groupByUser(stories) {
  const map = new Map()
  for (const s of stories) {
    if (!map.has(s.user_id)) {
      map.set(s.user_id, {
        user_id:    s.user_id,
        username:   s.author?.username   ?? s.username   ?? s.user_id.slice(0, 8),
        avatar_url: s.author?.avatar_url ?? s.avatar_url ?? null,
        full_name:  s.author?.full_name  ?? s.full_name  ?? null,
        stories:    [],
      })
    }
    map.get(s.user_id).stories.push(s)
  }
  return [...map.values()]
}

export default function StoriesPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { user: me } = useAuthStore()

  const [sets,      setSets]      = useState([])
  const [setIdx,    setSetIdx]    = useState(0)
  const [storyIdx,  setStoryIdx]  = useState(0)
  const [stories,   setStories]   = useState([])
  const [loading,   setLoading]   = useState(true)
  const [paused,    setPaused]    = useState(false)
  const [progress,  setProgress]  = useState(0)

  const [reply,        setReply]        = useState('')
  const [replying,     setReplying]     = useState(false)
  const [confirmDel,   setConfirmDel]   = useState(false)
  const [showViewers,  setShowViewers]  = useState(false)
  const [viewers,      setViewers]      = useState([])
  const [analytics,    setAnalytics]    = useState(null)

  const progressRef  = useRef(null)
  const startTimeRef = useRef(null)
  const holdRef      = useRef(false)

  /* ── Load story sets ─────────────────────────────────────────── */
  useEffect(() => {
    api.get('/stories/following', { params: { limit: 100 } })
      .then(({ data }) => {
        const grouped = groupByUser(data.stories ?? [])
        setSets(grouped)

        const userId = searchParams.get('user')
        const n      = parseInt(searchParams.get('index') ?? '0', 10)
        if (userId) {
          const i = grouped.findIndex((g) => g.user_id === userId)
          if (i !== -1) { setSetIdx(i); setStoryIdx(isNaN(n) ? 0 : n) }
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  /* ── Load stories for current set ───────────────────────────── */
  useEffect(() => {
    const set = sets[setIdx]
    if (!set) return
    if (set.stories?.length) { setStories(set.stories); return }

    api.get(`/stories/user/${set.user_id}`)
      .then(({ data }) => {
        const s = data.stories ?? []
        setStories(s)
        setSets((prev) =>
          prev.map((p, i) => (i === setIdx ? { ...p, stories: s } : p))
        )
      })
      .catch(() => setStories([]))
  }, [setIdx, sets.length]) // eslint-disable-line react-hooks/exhaustive-deps

  const currentSet   = sets[setIdx]
  const currentStory = stories[storyIdx]
  const isOwn        = currentSet?.user_id === me?.id

  /* ── Auto-advance ────────────────────────────────────────────── */
  useEffect(() => {
    if (loading || paused || !stories.length || !currentStory) return
    startTimeRef.current = Date.now() - (progress / 100) * STORY_DURATION
    clearInterval(progressRef.current)

    progressRef.current = setInterval(() => {
      if (holdRef.current) return
      const elapsed = Date.now() - startTimeRef.current
      const pct = Math.min((elapsed / STORY_DURATION) * 100, 100)
      setProgress(pct)
      if (pct >= 100) { clearInterval(progressRef.current); goNext() }
    }, TICK)

    return () => clearInterval(progressRef.current)
  }, [storyIdx, setIdx, paused, loading, stories.length]) // eslint-disable-line

  /* ── Mark viewed ─────────────────────────────────────────────── */
  useEffect(() => {
    if (!currentStory) return
    api.post(`/stories/${currentStory.id}/view`).catch(() => {})
  }, [currentStory?.id])

  /* ── ESC / scroll lock ───────────────────────────────────────── */
  useEffect(() => {
    const h = (e) => { if (e.key === 'Escape') close() }
    document.addEventListener('keydown', h)
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', h)
      document.body.style.overflow = prev
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  function close() { clearInterval(progressRef.current); navigate(-1) }

  function goNext() {
    setProgress(0)
    if (storyIdx + 1 < stories.length) {
      setStoryIdx((i) => i + 1)
    } else if (setIdx + 1 < sets.length) {
      setStories([])
      setSetIdx((i) => i + 1)
      setStoryIdx(0)
    } else {
      close()
    }
  }

  function goPrev() {
    setProgress(0)
    if (storyIdx > 0) {
      setStoryIdx((i) => i - 1)
    } else if (setIdx > 0) {
      setStories([])
      setSetIdx((i) => i - 1)
      setStoryIdx(0)
    }
  }

  function handleTap(e) {
    const rect = e.currentTarget.getBoundingClientRect()
    const pct  = (e.clientX - rect.left) / rect.width
    if (pct < 0.3) goPrev()
    else if (pct > 0.7) goNext()
  }

  function holdStart() { holdRef.current = true;  setPaused(true)  }
  function holdEnd()   { holdRef.current = false; setPaused(false) }

  async function sendReply() {
    if (!reply.trim() || !currentStory) return
    setReplying(true)
    try {
      await api.post(`/stories/${currentStory.id}/reply`, { text: reply })
      setReply('')
      toast.success('Reply sent!')
    } catch { toast.error('Failed to send reply') }
    finally { setReplying(false) }
  }

  async function reactEmoji(emoji) {
    if (!currentStory) return
    try { await api.post(`/stories/${currentStory.id}/reaction`, { emoji }) }
    catch {}
  }

  async function deleteStory() {
    if (!currentStory) return
    try {
      await api.delete(`/stories/${currentStory.id}`)
      toast.success('Story deleted')
      goNext()
    } catch { toast.error('Failed to delete') }
    setConfirmDel(false)
  }

  async function loadViewers() {
    if (!currentStory) return
    try {
      const [vRes, aRes] = await Promise.all([
        api.get(`/stories/${currentStory.id}/viewers`),
        api.get(`/stories/${currentStory.id}/analytics`),
      ])
      setViewers(vRes.data.viewers ?? [])
      setAnalytics(aRes.data)
    } catch {}
    setShowViewers(true)
  }

  function switchSet(dir) {
    setStories([])
    setSetIdx((i) => i + dir)
    setStoryIdx(0)
    setProgress(0)
  }

  return (
    <div className="fixed inset-0 z-50 bg-black flex items-center justify-center">
      {/* Prev user */}
      {setIdx > 0 && (
        <button
          onClick={() => switchSet(-1)}
          className="absolute left-4 top-1/2 -translate-y-1/2 z-20 flex flex-col items-center gap-1 group"
        >
          <div className="w-12 h-12 rounded-full border-2 border-white/40 overflow-hidden">
            <Avatar src={sets[setIdx - 1]?.avatar_url} name={sets[setIdx - 1]?.username} size={48} />
          </div>
          <ChevronLeft size={20} className="text-white/60 group-hover:text-white transition-colors" />
        </button>
      )}

      {/* Next user */}
      {setIdx < sets.length - 1 && (
        <button
          onClick={() => switchSet(1)}
          className="absolute right-4 top-1/2 -translate-y-1/2 z-20 flex flex-col items-center gap-1 group"
        >
          <div className="w-12 h-12 rounded-full border-2 border-white/40 overflow-hidden">
            <Avatar src={sets[setIdx + 1]?.avatar_url} name={sets[setIdx + 1]?.username} size={48} />
          </div>
          <ChevronRight size={20} className="text-white/60 group-hover:text-white transition-colors" />
        </button>
      )}

      {/* Story card */}
      <div
        className="relative flex flex-col overflow-hidden"
        style={{
          width: '100%', maxWidth: 420,
          height: '100%', maxHeight: '100dvh',
          background: '#111',
        }}
      >
        {/* Progress bars */}
        <div className="absolute top-0 inset-x-0 flex gap-1 p-2 z-10">
          {(stories.length ? stories : [null]).map((_, i) => (
            <div key={i} className="flex-1 h-0.5 bg-white/30 rounded-full overflow-hidden">
              <div
                className="h-full bg-white rounded-full"
                style={{ width: i < storyIdx ? '100%' : i === storyIdx ? `${progress}%` : '0%' }}
              />
            </div>
          ))}
        </div>

        {/* Header */}
        <div className="absolute top-5 inset-x-0 flex items-center gap-3 px-4 z-10 pt-1">
          <Avatar src={currentSet?.avatar_url} name={currentSet?.username} size={32} />
          <div className="flex-1 min-w-0">
            <p className="text-white text-sm font-semibold">@{currentSet?.username}</p>
            {currentStory && (
              <p className="text-white/60 text-xs">{formatRelativeTime(currentStory.created_at)}</p>
            )}
          </div>
          {isOwn && (
            <button onClick={loadViewers} className="text-white/70 hover:text-white p-1">
              <Eye size={16} />
            </button>
          )}
          {isOwn && (
            <button
              onClick={() => setConfirmDel(true)}
              className="text-white/70 hover:text-red-400 transition-colors p-1"
            >
              <Trash2 size={16} />
            </button>
          )}
          <button onClick={close} className="text-white/70 hover:text-white p-1">
            <X size={18} />
          </button>
        </div>

        {/* Media tap zone */}
        <div
          className="flex-1 flex items-center justify-center cursor-pointer select-none relative"
          onClick={handleTap}
          onMouseDown={holdStart}
          onMouseUp={holdEnd}
          onTouchStart={holdStart}
          onTouchEnd={holdEnd}
        >
          {loading ? (
            <div className="skeleton w-full h-full" />
          ) : !currentStory ? (
            <p className="text-white/50 text-sm">No stories.</p>
          ) : currentStory.media_type === 'video' ||
              /\.(mp4|webm|ogg)$/i.test(currentStory.media_url ?? '') ? (
            <video
              key={currentStory.id}
              src={currentStory.media_url}
              className="w-full h-full object-cover"
              autoPlay playsInline muted={paused}
            />
          ) : (
            <img
              key={currentStory.id}
              src={currentStory.media_url}
              alt=""
              className="w-full h-full object-cover"
            />
          )}

          {/* Caption */}
          {currentStory?.caption && (
            <div
              className="absolute bottom-0 inset-x-0 px-4 py-8 pointer-events-none"
              style={{ background: 'linear-gradient(to top, rgba(0,0,0,0.8) 0%, transparent 100%)' }}
            >
              <p className="text-white text-sm leading-relaxed">{currentStory.caption}</p>
            </div>
          )}
        </div>

        {/* Bottom bar */}
        <div
          className="shrink-0 px-3 py-2.5 flex flex-col gap-2"
          style={{ background: 'rgba(0,0,0,0.85)' }}
        >
          {/* Reaction row */}
          <div className="flex items-center gap-2">
            {isOwn && (
              <button
                onClick={loadViewers}
                className="flex items-center gap-1 text-white/70 text-xs hover:text-white transition-colors"
              >
                <Eye size={14} />
                <span>{currentStory?.view_count ?? 0}</span>
              </button>
            )}
            <div className="flex-1 flex items-center justify-center gap-1">
              {EMOJIS.map((e) => (
                <button
                  key={e}
                  onClick={() => reactEmoji(e)}
                  className="text-xl hover:scale-125 transition-transform active:scale-110"
                >
                  {e}
                </button>
              ))}
            </div>
          </div>

          {/* Reply input (others' stories only) */}
          {!isOwn && (
            <div className="flex items-center gap-2">
              <input
                type="text"
                value={reply}
                onChange={(e) => setReply(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && sendReply()}
                placeholder="Reply..."
                className="flex-1 bg-white/10 border border-white/20 rounded-full px-3 py-1.5 text-white text-sm placeholder-white/40 outline-none focus:border-white/50 transition-colors"
              />
              <button
                onClick={sendReply}
                disabled={!reply.trim() || replying}
                className="text-white/70 hover:text-white disabled:opacity-40 p-1 transition-colors"
              >
                <Send size={18} />
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Delete confirm */}
      {confirmDel && (
        <div className="fixed inset-0 z-60 flex items-center justify-center bg-black/70">
          <div className="card p-6 w-72 text-center animate-fade-in">
            <p className="font-semibold text-hi mb-2">Delete story?</p>
            <p className="text-sm text-lo mb-5">This cannot be undone.</p>
            <div className="flex gap-3">
              <button
                onClick={() => setConfirmDel(false)}
                className="flex-1 py-2 rounded-btn border text-sm"
                style={{ borderColor: 'var(--border)' }}
              >
                Cancel
              </button>
              <button
                onClick={deleteStory}
                className="flex-1 py-2 rounded-btn bg-red-600 text-white text-sm hover:bg-red-700 transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Viewers panel */}
      {showViewers && (
        <ViewersPanel
          story={currentStory}
          viewers={viewers}
          analytics={analytics}
          onClose={() => setShowViewers(false)}
        />
      )}
    </div>
  )
}

function ViewersPanel({ viewers, analytics, onClose }) {
  return (
    <div
      className="fixed inset-0 z-60 flex items-end justify-center"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="w-full max-w-sm rounded-t-2xl overflow-hidden animate-slide-up"
        style={{ background: 'var(--surface)', maxHeight: '70vh', display: 'flex', flexDirection: 'column' }}
      >
        <div
          className="flex items-center justify-between px-4 py-3 border-b shrink-0"
          style={{ borderColor: 'var(--border)' }}
        >
          <h3 className="font-semibold text-hi">Viewers</h3>
          <button onClick={onClose} className="text-lo hover:text-hi p-1"><X size={18} /></button>
        </div>

        {analytics && (
          <div
            className="flex items-center gap-5 px-4 py-3 border-b shrink-0"
            style={{ borderColor: 'var(--border)' }}
          >
            <div className="text-center">
              <p className="text-xl font-bold text-hi">{analytics.total_views ?? 0}</p>
              <p className="text-xs text-lo">Views</p>
            </div>
            {analytics.reactions &&
              Object.entries(analytics.reactions).map(([emoji, count]) => (
                <div key={emoji} className="text-center">
                  <p className="text-lg">{emoji}</p>
                  <p className="text-xs text-lo">{count}</p>
                </div>
              ))}
          </div>
        )}

        <div className="overflow-y-auto flex-1">
          {viewers.length === 0 ? (
            <p className="text-center text-lo text-sm py-8">No viewers yet.</p>
          ) : viewers.map((v) => (
            <div key={v.id ?? v.user_id} className="flex items-center gap-3 px-4 py-2.5">
              <Avatar src={v.avatar_url} name={v.username} size={36} />
              <div>
                <p className="text-sm font-semibold text-hi">@{v.username}</p>
                <p className="text-xs text-lo">{formatRelativeTime(v.viewed_at)}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

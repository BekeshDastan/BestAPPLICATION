import { useEffect, useRef, useState } from 'react'
import { X, ChevronLeft, ChevronRight, Pause, Play } from 'lucide-react'
import api from '../../lib/api'
import Avatar from '../shared/Avatar'
import { formatRelativeTime } from '../../lib/utils'

const STORY_DURATION = 5000 // ms per story

export default function HighlightViewerModal({ highlight, onClose }) {
  const [stories, setStories] = useState([])
  const [idx,     setIdx]     = useState(0)
  const [loading, setLoading] = useState(true)
  const [paused,  setPaused]  = useState(false)
  const [progress, setProgress] = useState(0) // 0-100

  const intervalRef  = useRef(null)
  const progressRef  = useRef(null)
  const startTimeRef = useRef(null)

  /* ── Fetch stories ───────────────────────────────────────────────── */
  useEffect(() => {
    api
      .get(`/stories/user/${highlight.user_id}`, { params: { limit: 50 } })
      .then(({ data }) => setStories(data.stories ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [highlight.user_id])

  /* ── Auto-advance ────────────────────────────────────────────────── */
  useEffect(() => {
    if (loading || paused || !stories.length) return
    startTimeRef.current = Date.now() - (progress / 100) * STORY_DURATION
    clearInterval(progressRef.current)

    progressRef.current = setInterval(() => {
      const elapsed = Date.now() - startTimeRef.current
      const pct = Math.min((elapsed / STORY_DURATION) * 100, 100)
      setProgress(pct)
      if (pct >= 100) {
        clearInterval(progressRef.current)
        goNext()
      }
    }, 50)

    return () => clearInterval(progressRef.current)
  }, [idx, paused, loading, stories.length]) // eslint-disable-line

  /* ── ESC close ───────────────────────────────────────────────────── */
  useEffect(() => {
    const h = (e) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', h)
    return () => document.removeEventListener('keydown', h)
  }, [onClose])

  /* ── Lock scroll ─────────────────────────────────────────────────── */
  useEffect(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = prev }
  }, [])

  function goNext() {
    setProgress(0)
    setIdx((i) => {
      if (i + 1 >= stories.length) { onClose(); return i }
      return i + 1
    })
  }

  function goPrev() {
    setProgress(0)
    setIdx((i) => Math.max(0, i - 1))
  }

  function handleTap(e) {
    const rect = e.currentTarget.getBoundingClientRect()
    const x = e.clientX - rect.left
    if (x < rect.width / 2) goPrev()
    else goNext()
  }

  const story = stories[idx]

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/95 animate-fade-in">
      <div
        className="relative w-full max-w-sm h-full max-h-[90dvh] flex flex-col rounded-card overflow-hidden"
        style={{ background: '#000' }}
      >
        {/* ── Progress bars ──────────────────────────────────── */}
        <div className="absolute top-0 inset-x-0 flex gap-1 p-2 z-10">
          {stories.map((_, i) => (
            <div
              key={i}
              className="flex-1 h-0.5 rounded-full overflow-hidden"
              style={{ background: 'rgba(255,255,255,0.3)' }}
            >
              <div
                className="h-full rounded-full transition-none"
                style={{
                  background: '#fff',
                  width:
                    i < idx ? '100%'
                    : i === idx ? `${progress}%`
                    : '0%',
                  transition: i === idx && !paused ? 'none' : undefined,
                }}
              />
            </div>
          ))}
        </div>

        {/* ── Header ─────────────────────────────────────────── */}
        <div className="absolute top-5 inset-x-0 flex items-center gap-3 px-4 z-10 pt-1">
          <Avatar
            src={highlight.cover_url}
            name={highlight.title}
            size={32}
          />
          <div className="flex-1 min-w-0">
            <p className="text-white text-sm font-semibold">{highlight.title}</p>
            {story && (
              <p className="text-white/60 text-xs">
                {formatRelativeTime(story.created_at)}
              </p>
            )}
          </div>
          <button
            onClick={() => setPaused((v) => !v)}
            className="text-white/70 hover:text-white transition-colors p-1"
          >
            {paused ? <Play size={16} /> : <Pause size={16} />}
          </button>
          <button
            onClick={onClose}
            className="text-white/70 hover:text-white transition-colors p-1"
          >
            <X size={18} />
          </button>
        </div>

        {/* ── Media ──────────────────────────────────────────── */}
        <div
          className="flex-1 flex items-center justify-center cursor-pointer select-none"
          onClick={handleTap}
        >
          {loading ? (
            <div className="skeleton w-full h-full" />
          ) : !story ? (
            <p className="text-white/50 text-sm">No stories in this highlight.</p>
          ) : story.media_type === 'video' ||
              /\.(mp4|webm|ogg)$/i.test(story.media_url ?? '') ? (
            <video
              key={story.id}
              src={story.media_url}
              className="w-full h-full object-cover"
              autoPlay
              muted={paused}
              playsInline
            />
          ) : (
            <img
              key={story.id}
              src={story.media_url}
              alt=""
              className="w-full h-full object-cover"
            />
          )}
        </div>

        {/* ── Caption ────────────────────────────────────────── */}
        {story?.caption && (
          <div
            className="absolute bottom-0 inset-x-0 px-4 py-5 z-10"
            style={{
              background: 'linear-gradient(to top, rgba(0,0,0,0.8) 0%, transparent 100%)',
            }}
          >
            <p className="text-white text-sm">{story.caption}</p>
          </div>
        )}

        {/* ── Nav arrows (desktop) ───────────────────────────── */}
        {idx > 0 && (
          <button
            onClick={(e) => { e.stopPropagation(); goPrev() }}
            className="absolute left-2 top-1/2 -translate-y-1/2 z-10 w-8 h-8 rounded-full bg-black/40 flex items-center justify-center text-white hover:bg-black/70 transition-colors"
          >
            <ChevronLeft size={18} />
          </button>
        )}
        {idx < stories.length - 1 && (
          <button
            onClick={(e) => { e.stopPropagation(); goNext() }}
            className="absolute right-2 top-1/2 -translate-y-1/2 z-10 w-8 h-8 rounded-full bg-black/40 flex items-center justify-center text-white hover:bg-black/70 transition-colors"
          >
            <ChevronRight size={18} />
          </button>
        )}
      </div>
    </div>
  )
}

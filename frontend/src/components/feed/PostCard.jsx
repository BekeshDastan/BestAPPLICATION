import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Heart, MessageCircle, Bookmark, Share2,
  MoreHorizontal, Trash2, Edit2, Flag,
} from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import { formatRelativeTime, formatCount } from '../../lib/utils'
import Avatar from '../shared/Avatar'
import CommentsSection from './CommentsSection'
import ReportModal from './ReportModal'

/* ── Hashtag / mention renderer ──────────────────────────────────────── */
function Caption({ text, expanded, onExpand }) {
  if (!text) return null
  const parts = text.split(/(#[\p{L}\p{N}_]+)/gu)
  const rendered = parts.map((chunk, i) =>
    chunk.startsWith('#') ? (
      <Link
        key={i}
        to={`/hashtag/${chunk.slice(1)}`}
        className="hover:underline"
        style={{ color: 'var(--accent)' }}
      >
        {chunk}
      </Link>
    ) : (
      chunk
    ),
  )
  const lines = text.split('\n').length
  return (
    <div className="px-4 pt-3 pb-1">
      <p
        className={`text-sm text-hi leading-relaxed ${
          !expanded && lines > 3 ? 'line-clamp-3' : ''
        }`}
      >
        {rendered}
      </p>
      {lines > 3 && !expanded && (
        <button
          onClick={onExpand}
          className="text-xs text-lo hover:text-hi mt-0.5 transition-colors"
        >
          more
        </button>
      )}
    </div>
  )
}

/* ── Skeleton ────────────────────────────────────────────────────────── */
export function PostSkeleton() {
  return (
    <div className="card overflow-hidden">
      <div className="flex items-center gap-3 px-4 py-3">
        <div className="skeleton w-9 h-9 rounded-full" />
        <div className="flex-1 space-y-1.5">
          <div className="skeleton w-32 h-3 rounded" />
          <div className="skeleton w-20 h-2.5 rounded" />
        </div>
      </div>
      <div className="skeleton aspect-video" />
      <div className="px-4 py-3 space-y-2">
        <div className="skeleton w-3/4 h-3 rounded" />
        <div className="skeleton w-1/2 h-3 rounded" />
      </div>
      <div className="px-3 py-2 flex gap-2">
        <div className="skeleton w-16 h-8 rounded-btn" />
        <div className="skeleton w-16 h-8 rounded-btn" />
        <div className="ml-auto flex gap-2">
          <div className="skeleton w-8 h-8 rounded-btn" />
          <div className="skeleton w-8 h-8 rounded-btn" />
        </div>
      </div>
    </div>
  )
}

/* ── PostCard ────────────────────────────────────────────────────────── */
export default function PostCard({ post, lastRef, onDelete }) {
  const { user: me } = useAuthStore()

  const [liked,      setLiked]      = useState(post.is_liked    ?? false)
  const [likesCount, setLikesCount] = useState(post.likes_count ?? 0)
  const [saved,      setSaved]      = useState(post.is_saved    ?? false)
  const [likeAnim,   setLikeAnim]   = useState(false)
  const [showComments, setShowComments] = useState(false)
  const [captionExp,   setCaptionExp]   = useState(false)
  const [menuOpen,     setMenuOpen]     = useState(false)
  const [showReport,   setShowReport]   = useState(false)

  const author = post.author ?? { id: post.author_id }
  const isOwn  = me?.id === (author.id ?? post.author_id)

  /* ── Like (optimistic) ─────────────────────────────────────────────── */
  async function toggleLike() {
    const next  = !liked
    const delta = next ? 1 : -1
    setLiked(next)
    setLikesCount((c) => c + delta)
    if (next) { setLikeAnim(true); setTimeout(() => setLikeAnim(false), 300) }
    try {
      if (next) await api.post(`/posts/${post.id}/like`)
      else      await api.delete(`/posts/${post.id}/like`)
    } catch {
      setLiked(!next)
      setLikesCount((c) => c - delta)
    }
  }

  /* ── Save (optimistic) ─────────────────────────────────────────────── */
  async function toggleSave() {
    const next = !saved
    setSaved(next)
    try {
      if (next) await api.post(`/posts/${post.id}/save`)
      else      await api.delete(`/posts/${post.id}/save`)
    } catch {
      setSaved(!next)
    }
  }

  /* ── Share ─────────────────────────────────────────────────────────── */
  async function handleShare() {
    const url = `${window.location.origin}/posts/${post.id}`
    try {
      await navigator.clipboard.writeText(url)
      toast.success('Link copied!')
    } catch {
      toast.error('Could not copy link')
    }
  }

  /* ── Delete ────────────────────────────────────────────────────────── */
  async function handleDelete() {
    if (!window.confirm('Delete this post?')) return
    try {
      await api.delete(`/posts/${post.id}`)
      onDelete?.(post.id)
      toast.success('Post deleted')
    } catch {
      toast.error('Failed to delete post')
    }
    setMenuOpen(false)
  }

  return (
    <article ref={lastRef} className="card overflow-hidden animate-fade-in">
      {/* ── Header ───────────────────────────────────────────────────── */}
      <div className="flex items-center gap-3 px-4 py-3">
        <Link to={`/profile/${author.id}`} className="shrink-0">
          <Avatar
            src={author.avatar_url}
            name={author.full_name ?? author.username}
            size={36}
          />
        </Link>
        <div className="flex-1 min-w-0">
          <Link
            to={`/profile/${author.id}`}
            className="text-sm font-semibold text-hi hover:underline block truncate"
          >
            {author.full_name ?? author.username ?? author.id?.slice(0, 8)}
          </Link>
          <div className="flex items-center gap-1">
            {author.username && (
              <>
                <span className="text-xs text-lo">@{author.username}</span>
                <span className="text-xs text-lo">·</span>
              </>
            )}
            <span className="text-xs text-lo">
              {formatRelativeTime(post.created_at)}
            </span>
          </div>
        </div>

        {/* ⋯ menu */}
        <div className="relative shrink-0">
          <button
            onClick={() => setMenuOpen((v) => !v)}
            className="p-1.5 rounded-btn text-lo hover:text-hi hover:bg-elevated transition-colors"
          >
            <MoreHorizontal size={18} />
          </button>

          {menuOpen && (
            <>
              <div
                className="fixed inset-0 z-10"
                onClick={() => setMenuOpen(false)}
              />
              <div
                className="absolute right-0 top-9 z-20 w-44 rounded-card border py-1 shadow-card"
                style={{
                  background:   'var(--surface)',
                  borderColor:  'var(--border)',
                }}
              >
                {isOwn ? (
                  <>
                    <button
                      className="flex items-center gap-2 w-full px-4 py-2 text-sm text-hi hover:bg-elevated transition-colors"
                      onClick={() => setMenuOpen(false)}
                    >
                      <Edit2 size={14} /> Edit caption
                    </button>
                    <button
                      className="flex items-center gap-2 w-full px-4 py-2 text-sm hover:bg-elevated transition-colors"
                      style={{ color: 'var(--danger)' }}
                      onClick={handleDelete}
                    >
                      <Trash2 size={14} /> Delete post
                    </button>
                  </>
                ) : (
                  <button
                    className="flex items-center gap-2 w-full px-4 py-2 text-sm text-hi hover:bg-elevated transition-colors"
                    onClick={() => { setMenuOpen(false); setShowReport(true) }}
                  >
                    <Flag size={14} /> Report
                  </button>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {/* ── Media ────────────────────────────────────────────────────── */}
      {post.media_urls?.length > 0 && (
        <div
          className="aspect-video overflow-hidden"
          style={{ background: 'var(--surface-high)' }}
        >
          {/\.(mp4|webm|ogg)$/i.test(post.media_urls[0]) ? (
            <video
              src={post.media_urls[0]}
              className="w-full h-full object-cover"
              controls
            />
          ) : (
            <img
              src={post.media_urls[0]}
              alt="post media"
              className="w-full h-full object-cover"
            />
          )}
        </div>
      )}

      {/* ── Caption ──────────────────────────────────────────────────── */}
      <Caption
        text={post.caption}
        expanded={captionExp}
        onExpand={() => setCaptionExp(true)}
      />

      {/* ── Tags ─────────────────────────────────────────────────────── */}
      {post.tags?.length > 0 && (
        <div className="flex flex-wrap gap-1.5 px-4 pb-1 pt-0.5">
          {post.tags.map((t) => (
            <Link
              key={t}
              to={`/hashtag/${t}`}
              className="text-[11px] px-2 py-0.5 rounded-full transition-colors"
              style={{
                background:  'var(--accent-glow)',
                color:       'var(--accent)',
              }}
            >
              #{t}
            </Link>
          ))}
        </div>
      )}

      {/* ── Action bar ───────────────────────────────────────────────── */}
      <div className="flex items-center gap-0.5 px-2 py-1.5">
        {/* Like */}
        <button
          onClick={toggleLike}
          className={`flex items-center gap-1.5 px-3 py-2 rounded-btn text-sm font-medium transition-all
            ${likeAnim ? 'scale-110' : 'scale-100'}`}
          style={{ color: liked ? 'var(--danger)' : 'var(--text-2)' }}
        >
          <Heart
            size={18}
            fill={liked ? 'var(--danger)' : 'none'}
            className="transition-all"
          />
          <span>{formatCount(likesCount)}</span>
        </button>

        {/* Comment */}
        <button
          onClick={() => setShowComments((v) => !v)}
          className="flex items-center gap-1.5 px-3 py-2 rounded-btn text-sm font-medium text-lo hover:text-hi transition-colors"
        >
          <MessageCircle size={18} />
          <span>{formatCount(post.comments_count ?? 0)}</span>
        </button>

        <div className="flex-1" />

        {/* Save */}
        <button
          onClick={toggleSave}
          className="p-2 rounded-btn transition-colors"
          style={{ color: saved ? 'var(--accent)' : 'var(--text-2)' }}
        >
          <Bookmark size={18} fill={saved ? 'var(--accent)' : 'none'} />
        </button>

        {/* Share */}
        <button
          onClick={handleShare}
          className="p-2 rounded-btn text-lo hover:text-hi transition-colors"
        >
          <Share2 size={18} />
        </button>
      </div>

      {/* ── Comments ─────────────────────────────────────────────────── */}
      {showComments && (
        <CommentsSection
          postId={post.id}
          initialCount={post.comments_count}
        />
      )}

      {/* ── Report modal ─────────────────────────────────────────────── */}
      {showReport && (
        <ReportModal postId={post.id} onClose={() => setShowReport(false)} />
      )}
    </article>
  )
}

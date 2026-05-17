import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { X } from 'lucide-react'
import { formatRelativeTime } from '../../lib/utils'
import Avatar from './Avatar'
import CommentsSection from '../feed/CommentsSection'
import PostCard from '../feed/PostCard'

export default function PostDetailModal({ post, onClose }) {
  /* Lock body scroll */
  useEffect(() => {
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => { document.body.style.overflow = prev }
  }, [])

  /* ESC to close */
  useEffect(() => {
    const handler = (e) => { if (e.key === 'Escape') onClose() }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [onClose])

  const author = post.author ?? { id: post.author_id }
  const hasMedia = post.media_urls?.length > 0

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 modal-backdrop"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        className="card w-full max-w-5xl max-h-[95vh] min-h-[60vh] flex flex-col md:flex-row overflow-hidden animate-fade-in"
        style={{ background: 'var(--surface)' }}
      >
        {/* ── Media side ──────────────────────────────────────────── */}
        {hasMedia && (
          <div
            className="md:w-[55%] flex items-center justify-center shrink-0"
            style={{ background: '#000' }}
          >
            {/\.(mp4|webm|ogg)$/i.test(post.media_urls[0]) ? (
              <video
                src={post.media_urls[0]}
                className="w-full max-h-[92vh] object-contain"
                controls
              />
            ) : (
              <img
                src={post.media_urls[0]}
                alt="post"
                className="w-full max-h-[92vh] object-contain"
              />
            )}
          </div>
        )}

        {/* ── Info side ───────────────────────────────────────────── */}
        <div
          className="flex-1 flex flex-col overflow-hidden min-w-0"
          style={{ minWidth: 300 }}
        >
          {/* Header */}
          <div
            className="flex items-center gap-3 px-4 py-3 border-b shrink-0"
            style={{ borderColor: 'var(--border)' }}
          >
            <Link to={`/profile/${author.id}`} onClick={onClose} className="shrink-0">
              <Avatar
                src={author.avatar_url}
                name={author.full_name ?? author.username}
                size={32}
              />
            </Link>
            <div className="flex-1 min-w-0">
              <Link
                to={`/profile/${author.id}`}
                onClick={onClose}
                className="text-sm font-semibold text-hi hover:underline block truncate"
              >
                {author.full_name ?? author.username}
              </Link>
              <span className="text-[11px] text-lo">
                {formatRelativeTime(post.created_at)}
              </span>
            </div>
            <button
              onClick={onClose}
              className="p-1.5 rounded-btn text-lo hover:text-hi transition-colors shrink-0"
            >
              <X size={18} />
            </button>
          </div>

          {/* Caption */}
          {post.caption && (
            <div
              className="px-4 py-3 border-b shrink-0"
              style={{ borderColor: 'var(--border)' }}
            >
              <p className="text-sm text-hi leading-relaxed">{post.caption}</p>
            </div>
          )}

          {/* Scrollable comments */}
          <div className="flex-1 overflow-y-auto">
            <CommentsSection postId={post.id} initialCount={post.comments_count} />
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { Send, Trash2, Loader2 } from 'lucide-react'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import Avatar from '../shared/Avatar'
import { formatRelativeTime } from '../../lib/utils'

function CommentSkeleton() {
  return (
    <div className="flex gap-2 items-start">
      <div className="skeleton w-7 h-7 rounded-full shrink-0" />
      <div className="flex-1 space-y-1.5 pt-0.5">
        <div className="skeleton w-24 h-2.5 rounded" />
        <div className="skeleton w-48 h-2.5 rounded" />
      </div>
    </div>
  )
}

export default function CommentsSection({ postId, initialCount = 0 }) {
  const { user: me } = useAuthStore()

  const [comments, setComments] = useState([])
  const [cursor,   setCursor]   = useState(null)
  const [hasMore,  setHasMore]  = useState(false)
  const [loading,  setLoading]  = useState(true)
  const [text,     setText]     = useState('')
  const [sending,  setSending]  = useState(false)

  useEffect(() => { load(null) }, [postId]) // eslint-disable-line

  async function load(c) {
    setLoading(true)
    try {
      const params = { limit: 10 }
      if (c) params.cursor = c
      const { data } = await api.get(`/posts/${postId}/comments`, { params })
      const fresh = data.comments ?? []
      setComments((prev) => (c ? [...prev, ...fresh] : fresh))
      setCursor(data.next_cursor ?? null)
      setHasMore(!!data.next_cursor)
    } catch {
      /* swallow */
    } finally {
      setLoading(false)
    }
  }

  async function handleSend() {
    if (!text.trim() || sending) return
    setSending(true)
    const body = text.trim()
    setText('')
    try {
      const { data } = await api.post(`/posts/${postId}/comments`, { body })
      const optimistic = data.comment ?? {
        id:        `tmp-${Date.now()}`,
        body,
        author_id: me?.id,
        author:    me,
        created_at: Math.floor(Date.now() / 1000),
      }
      setComments((prev) => [optimistic, ...prev])
    } catch {
      setText(body) // restore on failure
    } finally {
      setSending(false)
    }
  }

  async function handleDelete(commentId) {
    setComments((prev) => prev.filter((c) => c.id !== commentId))
    try {
      await api.delete(`/posts/${postId}/comments/${commentId}`)
    } catch {
      load(null) // reload on failure
    }
  }

  return (
    <div
      className="border-t px-4 pt-4 pb-3 space-y-4"
      style={{ borderColor: 'var(--border)' }}
    >
      <p className="text-xs font-semibold text-lo">
        Comments ({comments.length + (hasMore ? '+' : '')})
      </p>

      {/* List */}
      <div className="space-y-3">
        {loading && !comments.length
          ? Array.from({ length: 3 }).map((_, i) => <CommentSkeleton key={i} />)
          : comments.map((c) => {
              const isOwn =
                c.author_id === me?.id || c.author?.id === me?.id
              return (
                <div
                  key={c.id}
                  className="flex items-start gap-2 group"
                >
                  <Avatar
                    size={28}
                    src={c.author?.avatar_url}
                    name={c.author?.full_name ?? c.author?.username}
                  />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-1.5 flex-wrap">
                      <span className="text-xs font-semibold text-hi">
                        @{c.author?.username ?? c.author_id?.slice(0, 8)}
                      </span>
                      <span className="text-[10px] text-lo">
                        {formatRelativeTime(c.created_at)}
                      </span>
                    </div>
                    <p className="text-xs text-hi mt-0.5 leading-relaxed">
                      {c.body}
                    </p>
                  </div>
                  {isOwn && (
                    <button
                      onClick={() => handleDelete(c.id)}
                      className="opacity-0 group-hover:opacity-100 p-1 text-lo hover:text-danger transition-all shrink-0"
                      title="Delete"
                    >
                      <Trash2 size={12} />
                    </button>
                  )}
                </div>
              )
            })}
      </div>

      {hasMore && (
        <button
          onClick={() => load(cursor)}
          disabled={loading}
          className="text-xs text-lo hover:text-hi transition-colors disabled:opacity-40"
        >
          {loading ? 'Loading…' : 'Load more comments'}
        </button>
      )}

      {/* Add comment */}
      <div className="flex items-center gap-2 pt-1">
        <Avatar size={28} src={me?.avatar_url} name={me?.full_name ?? me?.username} />
        <div className="flex flex-1 items-center gap-2">
          <input
            type="text"
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSend()}
            placeholder="Add a comment…"
            className="input-base py-1.5 text-xs flex-1"
          />
          <button
            onClick={handleSend}
            disabled={!text.trim() || sending}
            className="shrink-0 transition-opacity disabled:opacity-40"
            style={{ color: 'var(--accent)' }}
          >
            {sending
              ? <Loader2 size={16} className="animate-spin" />
              : <Send size={16} />
            }
          </button>
        </div>
      </div>
    </div>
  )
}

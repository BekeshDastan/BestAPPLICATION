import { useEffect, useState, useRef } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../AuthContext'
import { postApi, userApi } from '../api'
import api from '../api'

function Avatar({ name, size = 32 }) {
  return (
    <div style={{ width: size, height: size, borderRadius: '50%', background: '#333', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: size * 0.4, flexShrink: 0 }}>
      {name?.[0]?.toUpperCase() || '?'}
    </div>
  )
}

function CommentSection({ postId, myId, initialCount }) {
  const [open, setOpen] = useState(false)
  const [comments, setComments] = useState([])
  const [text, setText] = useState('')
  const [count, setCount] = useState(initialCount || 0)
  const [loaded, setLoaded] = useState(false)

  const load = async () => {
    try {
      const { data } = await postApi.listComments(postId)
      setComments(data.comments || [])
      setLoaded(true)
    } catch {}
  }

  const toggle = () => {
    setOpen(o => !o)
    if (!loaded) load()
  }

  const submit = async e => {
    e.preventDefault()
    if (!text.trim()) return
    try {
      const { data } = await postApi.addComment(postId, text.trim())
      setComments(c => [...c, data.comment])
      setCount(n => n + 1)
      setText('')
    } catch {}
  }

  const remove = async (cid) => {
    try {
      await postApi.deleteComment(postId, cid)
      setComments(c => c.filter(x => x.id !== cid))
      setCount(n => Math.max(0, n - 1))
    } catch {}
  }

  return (
    <div>
      <button onClick={toggle} style={s.commentToggle}>
        💬 {count} comment{count !== 1 ? 's' : ''}
      </button>
      {open && (
        <div style={s.commentBox}>
          {comments.map(c => (
            <div key={c.id} style={s.commentRow}>
              <Avatar name={c.author_id || '?'} size={24} />
              <div style={{ flex: 1 }}>
                <span style={s.commentAuthor}>{c.author_id?.slice(0, 8) || 'user'}</span>
                <span style={s.commentText}> {c.body || c.text}</span>
              </div>
              {c.author_id === myId && (
                <button onClick={() => remove(c.id)} style={s.delBtn}>✕</button>
              )}
            </div>
          ))}
          {comments.length === 0 && <p style={s.empty}>No comments yet.</p>}
          <form onSubmit={submit} style={s.commentForm}>
            <input value={text} onChange={e => setText(e.target.value)} placeholder="Add a comment…" style={s.commentInput} />
            <button type="submit" style={s.postCommentBtn} disabled={!text.trim()}>Post</button>
          </form>
        </div>
      )}
    </div>
  )
}

function EditPostModal({ post, onClose, onSaved }) {
  const [caption, setCaption] = useState(post.caption || '')
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  const submit = async e => {
    e.preventDefault()
    setLoading(true); setErr('')
    try {
      const { data } = await postApi.update(post.id, caption)
      onSaved(data.post)
    } catch (e) { setErr(e.response?.data?.error || 'Update failed') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={{ ...s.modal, maxWidth: 480 }} onClick={e => e.stopPropagation()}>
        <div style={s.modalHead}>
          <span style={{ fontWeight: 600 }}>Edit post</span>
          <button onClick={onClose} style={s.closeBtn}>✕</button>
        </div>
        {err && <div style={s.err}>{err}</div>}
        <form onSubmit={submit} style={s.modalForm}>
          <textarea value={caption} onChange={e => setCaption(e.target.value)} rows={4} style={s.textarea} placeholder="Caption…" />
          <button type="submit" disabled={loading} style={s.submitBtn}>{loading ? 'Saving…' : 'Save'}</button>
        </form>
      </div>
    </div>
  )
}

function PostCard({ post: initialPost, myId, onDelete }) {
  const [post, setPost] = useState(initialPost)
  const [liked, setLiked] = useState(false)
  const [likes, setLikes] = useState(initialPost.likes_count || 0)
  const [deleting, setDeleting] = useState(false)
  const [editing, setEditing] = useState(false)
  const [showMenu, setShowMenu] = useState(false)

  const toggleLike = async () => {
    try {
      if (liked) { await postApi.unlike(post.id); setLikes(n => n - 1) }
      else { await postApi.like(post.id); setLikes(n => n + 1) }
      setLiked(l => !l)
    } catch {}
  }

  const del = async () => {
    setShowMenu(false)
    if (!confirm('Delete this post?')) return
    setDeleting(true)
    try { await postApi.delete(post.id); onDelete(post.id) }
    catch { setDeleting(false) }
  }

  const openPost = async () => {
    // GET /posts/:id — fetch full post detail
    try {
      const { data } = await postApi.get(post.id)
      setPost(data.post || post)
    } catch {}
  }

  const ts = post.created_at
    ? new Date(post.created_at > 1e12 ? post.created_at : post.created_at * 1000).toLocaleDateString()
    : ''

  return (
    <div style={s.card}>
      <div style={s.cardHead}>
        <Link to={`/profile/${post.author_id}`} style={{ display: 'flex', alignItems: 'center', gap: 8, textDecoration: 'none', color: 'inherit' }}>
          <Avatar name={post.author_id} size={36} />
          <div>
            <div style={s.author}>{post.username || post.author_id?.slice(0, 8) || 'user'}</div>
            {ts && <div style={s.time}>{ts}</div>}
          </div>
        </Link>
        {post.author_id === myId && (
          <div style={{ position: 'relative' }}>
            <button onClick={() => setShowMenu(m => !m)} style={s.menuBtn}>⋯</button>
            {showMenu && (
              <div style={s.menu}>
                <button onClick={() => { setShowMenu(false); setEditing(true) }} style={s.menuItem}>✏ Edit</button>
                <button onClick={del} disabled={deleting} style={{ ...s.menuItem, color: '#e53935' }}>🗑 Delete</button>
              </div>
            )}
          </div>
        )}
      </div>
      {post.media_urls?.[0] && (
        <img src={post.media_urls[0]} alt="" style={s.img} onError={e => { e.target.style.display = 'none' }} onClick={openPost} />
      )}
      {post.caption && <p style={s.caption}>{post.caption}</p>}
      <div style={s.actions}>
        <button onClick={toggleLike} style={{ ...s.likeBtn, color: liked ? '#e53935' : '#262626' }}>
          {liked ? '❤' : '♡'} {likes}
        </button>
      </div>
      <CommentSection postId={post.id} myId={myId} initialCount={post.comments_count || 0} />
      {editing && <EditPostModal post={post} onClose={() => setEditing(false)} onSaved={p => { setPost(p); setEditing(false) }} />}
    </div>
  )
}

function CreatePostModal({ onClose, onCreated }) {
  const [caption, setCaption] = useState('')
  const [mediaUrl, setMediaUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  const submit = async e => {
    e.preventDefault()
    if (!caption.trim() && !mediaUrl.trim()) return
    setLoading(true); setErr('')
    try {
      if (!mediaUrl.trim()) { setErr('At least one media URL is required'); setLoading(false); return }
      const { data } = await postApi.create(caption, [mediaUrl])
      onCreated(data.post)
    } catch (e) { setErr(e.response?.data?.error || 'Failed to post') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={s.modal} onClick={e => e.stopPropagation()}>
        <div style={s.modalHead}>
          <span style={{ fontWeight: 600 }}>Create new post</span>
          <button onClick={onClose} style={s.closeBtn}>✕</button>
        </div>
        {err && <div style={s.err}>{err}</div>}
        <form onSubmit={submit} style={s.modalForm}>
          <textarea
            placeholder="Write a caption…"
            value={caption}
            onChange={e => setCaption(e.target.value)}
            rows={4}
            style={s.textarea}
          />
          <input
            placeholder="Image URL (required)"
            value={mediaUrl}
            onChange={e => setMediaUrl(e.target.value)}
            style={s.input}
            required
          />
          {mediaUrl && <img src={mediaUrl} alt="preview" style={s.preview} onError={e => e.target.style.display = 'none'} />}
          <button type="submit" disabled={loading || !mediaUrl.trim()} style={s.submitBtn}>
            {loading ? 'Posting…' : 'Share'}
          </button>
        </form>
      </div>
    </div>
  )
}

export default function FeedPage() {
  const { user } = useAuth()
  const [posts, setPosts] = useState([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)

  useEffect(() => {
    const loadFeed = async () => {
      try {
        // Gateway needs following_ids — fetch them first
        const followingRes = await userApi.listFollowing(user.id)
        const ids = (followingRes.data.users || []).map(u => u.id).filter(Boolean)
        const allIds = [...new Set([...ids, user.id])]
        const qs = allIds.join(',')
        const { data } = await api.get(`/posts/feed?following_ids=${qs}`, { params: { limit: 20 } })
        setPosts(data.posts || [])
      } catch {
        setPosts([])
      } finally {
        setLoading(false)
      }
    }
    loadFeed()
  }, [user?.id])

  const handleCreated = post => { setPosts(p => [post, ...p]); setShowCreate(false) }
  const handleDelete = id => setPosts(p => p.filter(x => x.id !== id))

  return (
    <div style={s.page}>
      <div style={s.feed}>
        <button onClick={() => setShowCreate(true)} style={s.newPostBtn}>+ Create Post</button>
        {loading && <div style={s.hint}>Loading feed…</div>}
        {!loading && posts.length === 0 && (
          <div style={s.emptyFeed}>
            <p>No posts yet.</p>
            <p style={{ color: '#8e8e8e', fontSize: 14 }}>Follow users to see their posts, or create your first post!</p>
          </div>
        )}
        {posts.map(p => (
          <PostCard key={p.id} post={p} myId={user?.id} onDelete={handleDelete} />
        ))}
      </div>
      {showCreate && <CreatePostModal onClose={() => setShowCreate(false)} onCreated={handleCreated} />}
    </div>
  )
}

const s = {
  page: { maxWidth: 620, margin: '0 auto', padding: '24px 16px' },
  feed: { display: 'flex', flexDirection: 'column', gap: 16 },
  newPostBtn: { background: '#0095f6', color: '#fff', border: 'none', borderRadius: 8, padding: '10px 20px', fontWeight: 600, fontSize: 14, alignSelf: 'center' },
  card: { background: '#fff', border: '1px solid #dbdbdb', borderRadius: 8 },
  cardHead: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 16px' },
  author: { fontWeight: 600, fontSize: 14 },
  time: { fontSize: 11, color: '#8e8e8e' },
  deletePost: { background: 'none', border: 'none', color: '#999', fontSize: 16, padding: '4px 8px' },
  menuBtn: { background: 'none', border: 'none', fontSize: 20, color: '#8e8e8e', padding: '4px 8px', cursor: 'pointer', lineHeight: 1 },
  menu: { position: 'absolute', right: 0, top: '100%', background: '#fff', border: '1px solid #dbdbdb', borderRadius: 8, boxShadow: '0 4px 16px rgba(0,0,0,.12)', zIndex: 10, minWidth: 120 },
  menuItem: { display: 'block', width: '100%', padding: '10px 16px', background: 'none', border: 'none', textAlign: 'left', fontSize: 13, cursor: 'pointer', color: '#262626' },
  img: { width: '100%', maxHeight: 520, objectFit: 'cover', display: 'block' },
  caption: { padding: '10px 16px', margin: 0, fontSize: 14 },
  actions: { padding: '4px 12px 4px' },
  likeBtn: { background: 'none', border: 'none', fontSize: 18, padding: '6px 4px', fontWeight: 600 },
  commentToggle: { background: 'none', border: 'none', padding: '4px 16px 8px', fontSize: 13, color: '#8e8e8e', cursor: 'pointer' },
  commentBox: { borderTop: '1px solid #efefef', padding: '8px 16px 12px' },
  commentRow: { display: 'flex', alignItems: 'flex-start', gap: 8, marginBottom: 8 },
  commentAuthor: { fontWeight: 600, fontSize: 13 },
  commentText: { fontSize: 13 },
  delBtn: { background: 'none', border: 'none', color: '#bbb', fontSize: 12, padding: '0 4px', cursor: 'pointer' },
  commentForm: { display: 'flex', gap: 8, marginTop: 8 },
  commentInput: { flex: 1, border: 'none', borderBottom: '1px solid #dbdbdb', padding: '6px 4px', fontSize: 13, outline: 'none' },
  postCommentBtn: { background: 'none', border: 'none', color: '#0095f6', fontWeight: 600, fontSize: 13 },
  empty: { color: '#8e8e8e', fontSize: 13, textAlign: 'center', padding: '8px 0' },
  hint: { textAlign: 'center', color: '#8e8e8e', padding: '20px' },
  emptyFeed: { textAlign: 'center', padding: '40px 20px', color: '#262626' },
  overlay: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 200 },
  modal: { background: '#fff', borderRadius: 12, width: 500, maxWidth: '95vw', overflow: 'hidden' },
  modalHead: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '12px 16px', borderBottom: '1px solid #dbdbdb' },
  closeBtn: { background: 'none', border: 'none', fontSize: 18, color: '#8e8e8e', cursor: 'pointer' },
  err: { background: '#fff3f3', color: '#e53935', fontSize: 13, padding: '8px 16px' },
  modalForm: { padding: '16px', display: 'flex', flexDirection: 'column', gap: 10 },
  textarea: { padding: '10px', border: '1px solid #dbdbdb', borderRadius: 6, fontSize: 14, resize: 'vertical', minHeight: 100 },
  input: { padding: '9px 10px', border: '1px solid #dbdbdb', borderRadius: 6, fontSize: 14 },
  preview: { width: '100%', maxHeight: 300, objectFit: 'cover', borderRadius: 6 },
  submitBtn: { padding: '10px', background: '#0095f6', color: '#fff', border: 'none', borderRadius: 6, fontWeight: 700, fontSize: 14 },
}

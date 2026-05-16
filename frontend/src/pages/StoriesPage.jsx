import { useEffect, useState } from 'react'
import { useAuth } from '../AuthContext'
import { storyApi } from '../api'

function StoryViewer({ stories, startIdx, onClose }) {
  const [idx, setIdx] = useState(startIdx)
  const story = stories[idx]

  useEffect(() => {
    if (story) storyApi.markViewed(story.id).catch(() => {})
  }, [story?.id])

  if (!story) return null

  const ts = story.expires_at
    ? new Date(story.expires_at * 1000).toLocaleString()
    : ''

  return (
    <div style={sv.overlay} onClick={onClose}>
      <div style={sv.container} onClick={e => e.stopPropagation()}>
        <div style={sv.bar}>
          {stories.map((_, i) => (
            <div key={i} style={{ ...sv.seg, background: i <= idx ? '#fff' : 'rgba(255,255,255,.4)' }} />
          ))}
        </div>
        <button onClick={onClose} style={sv.close}>✕</button>
        <div style={sv.meta}>
          <div style={sv.dot}>{story.user_id?.[0]?.toUpperCase() || '?'}</div>
          <span style={{ fontSize: 13, marginLeft: 8 }}>{story.user_id?.slice(0, 12)}</span>
          {ts && <span style={{ fontSize: 11, color: 'rgba(255,255,255,.7)', marginLeft: 'auto' }}>exp {ts}</span>}
        </div>
        {story.media_url
          ? <img src={story.media_url} alt="" style={sv.img} onError={e => e.target.style.display = 'none'} />
          : <div style={sv.noMedia} />
        }
        {story.caption && <p style={sv.caption}>{story.caption}</p>}
        <div style={sv.nav}>
          <button style={sv.navBtn} onClick={() => setIdx(i => Math.max(0, i - 1))} disabled={idx === 0}>‹</button>
          <button style={sv.navBtn} onClick={() => setIdx(i => Math.min(stories.length - 1, i + 1))} disabled={idx === stories.length - 1}>›</button>
        </div>
      </div>
    </div>
  )
}

function CreateStoryModal({ onClose, onCreated }) {
  const [mediaUrl, setMediaUrl] = useState('')
  const [caption, setCaption] = useState('')
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')

  const submit = async e => {
    e.preventDefault()
    if (!mediaUrl.trim()) { setErr('Media URL is required'); return }
    setLoading(true); setErr('')
    try {
      const { data } = await storyApi.create(mediaUrl.trim(), 'image', caption.trim())
      onCreated(data.story)
    } catch (e) { setErr(e.response?.data?.error || 'Failed to create story') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={s.modal} onClick={e => e.stopPropagation()}>
        <div style={s.mHead}>
          <span style={{ fontWeight: 600 }}>Create Story</span>
          <button onClick={onClose} style={s.closeBtn}>✕</button>
        </div>
        {err && <div style={s.err}>{err}</div>}
        <form onSubmit={submit} style={s.form}>
          <input
            placeholder="Image URL (required)"
            value={mediaUrl}
            onChange={e => setMediaUrl(e.target.value)}
            style={s.input}
            autoFocus
          />
          {mediaUrl && <img src={mediaUrl} alt="preview" style={s.preview} onError={e => e.target.style.display = 'none'} />}
          <input
            placeholder="Caption (optional)"
            value={caption}
            onChange={e => setCaption(e.target.value)}
            style={s.input}
          />
          <button type="submit" disabled={loading || !mediaUrl.trim()} style={s.btn}>
            {loading ? 'Posting…' : 'Share Story'}
          </button>
        </form>
      </div>
    </div>
  )
}

function StoryBubble({ userId, stories, onOpen }) {
  return (
    <div style={sb.wrap} onClick={() => onOpen(stories, 0)}>
      <div style={sb.ring}>
        <div style={sb.avatar}>{userId?.[0]?.toUpperCase() || '?'}</div>
      </div>
      <div style={sb.label}>{userId?.slice(0, 8) || '?'}</div>
      <div style={sb.count}>{stories.length}</div>
    </div>
  )
}

export default function StoriesPage() {
  const { user } = useAuth()
  const [groups, setGroups] = useState({})
  const [myStories, setMyStories] = useState([])
  const [loading, setLoading] = useState(true)
  const [showCreate, setShowCreate] = useState(false)
  const [viewer, setViewer] = useState(null)

  const load = async () => {
    setLoading(true)
    try {
      const [myR, feedR] = await Promise.all([
        storyApi.listUser(user.id),
        storyApi.listFollowing(),
      ])
      const my = myR.data.stories || []
      setMyStories(my)
      const feed = feedR.data.stories || []
      const g = {}
      feed.forEach(st => {
        if (!g[st.user_id]) g[st.user_id] = []
        g[st.user_id].push(st)
      })
      setGroups(g)
    } catch {}
    finally { setLoading(false) }
  }

  useEffect(() => { load() }, [])

  const handleCreated = story => {
    setMyStories(m => [story, ...m])
    setShowCreate(false)
  }

  const del = async (id) => {
    try {
      await storyApi.delete(id)
      setMyStories(m => m.filter(s => s.id !== id))
    } catch {}
  }

  const openViewer = (stories, idx) => setViewer({ stories, idx })

  return (
    <div style={s.page}>
      <div style={s.headerRow}>
        <h2 style={s.title}>Stories</h2>
        <button onClick={() => setShowCreate(true)} style={s.newBtn}>+ Add Story</button>
      </div>

      {loading && <div style={s.hint}>Loading…</div>}

      {myStories.length > 0 && (
        <section style={s.section}>
          <div style={s.sectionTitle}>Your Stories</div>
          <div style={s.bubbleRow}>
            {myStories.map((st, i) => (
              <div key={st.id} style={sb.wrap}>
                <div style={{ ...sb.ring, boxShadow: '0 0 0 3px #0095f6' }} onClick={() => openViewer(myStories, i)}>
                  <div style={sb.avatar}>{user?.username?.[0]?.toUpperCase() || '?'}</div>
                </div>
                <button onClick={() => del(st.id)} style={s.delStoryBtn}>✕</button>
                <div style={sb.label}>You</div>
              </div>
            ))}
          </div>
        </section>
      )}

      {Object.keys(groups).length > 0 && (
        <section style={s.section}>
          <div style={s.sectionTitle}>Following</div>
          <div style={s.bubbleRow}>
            {Object.entries(groups).map(([uid, strs]) => (
              <StoryBubble key={uid} userId={uid} stories={strs} onOpen={openViewer} />
            ))}
          </div>
        </section>
      )}

      {!loading && Object.keys(groups).length === 0 && myStories.length === 0 && (
        <div style={s.empty}>
          <div style={{ fontSize: 48, marginBottom: 12 }}>◎</div>
          <p style={{ fontWeight: 600 }}>No stories yet.</p>
          <p style={{ color: '#8e8e8e', fontSize: 14 }}>Follow users or share your first story!</p>
        </div>
      )}

      {showCreate && <CreateStoryModal onClose={() => setShowCreate(false)} onCreated={handleCreated} />}
      {viewer && <StoryViewer stories={viewer.stories} startIdx={viewer.idx} onClose={() => setViewer(null)} />}
    </div>
  )
}

const s = {
  page: { maxWidth: 935, margin: '0 auto', padding: '24px 20px' },
  headerRow: { display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24 },
  title: { margin: 0, fontWeight: 300, fontSize: 24 },
  newBtn: { background: '#0095f6', color: '#fff', border: 'none', borderRadius: 8, padding: '8px 18px', fontWeight: 600, fontSize: 14, cursor: 'pointer' },
  section: { marginBottom: 32 },
  sectionTitle: { fontWeight: 600, fontSize: 14, color: '#8e8e8e', marginBottom: 12 },
  bubbleRow: { display: 'flex', gap: 16, flexWrap: 'wrap' },
  hint: { textAlign: 'center', color: '#8e8e8e', padding: '40px' },
  empty: { textAlign: 'center', padding: '60px 20px', color: '#262626' },
  delStoryBtn: { position: 'absolute', top: -4, right: -4, background: '#e53935', color: '#fff', border: 'none', borderRadius: '50%', width: 18, height: 18, fontSize: 10, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 0 },
  overlay: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,.65)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 200 },
  modal: { background: '#fff', borderRadius: 12, width: 420, maxWidth: '95vw' },
  mHead: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 16px', borderBottom: '1px solid #dbdbdb' },
  closeBtn: { background: 'none', border: 'none', fontSize: 18, color: '#8e8e8e', cursor: 'pointer' },
  err: { background: '#fff3f3', color: '#e53935', fontSize: 13, padding: '8px 16px' },
  form: { padding: '16px', display: 'flex', flexDirection: 'column', gap: 10 },
  input: { padding: '9px 12px', border: '1px solid #dbdbdb', borderRadius: 6, fontSize: 14 },
  preview: { width: '100%', maxHeight: 200, objectFit: 'cover', borderRadius: 6 },
  btn: { padding: '10px', background: '#0095f6', color: '#fff', border: 'none', borderRadius: 6, fontWeight: 700, fontSize: 14, cursor: 'pointer' },
}

const sb = {
  wrap: { display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4, cursor: 'pointer', position: 'relative' },
  ring: { width: 64, height: 64, borderRadius: '50%', boxShadow: '0 0 0 3px #e53935, 0 0 0 5px #fff', display: 'flex', alignItems: 'center', justifyContent: 'center' },
  avatar: { width: 56, height: 56, borderRadius: '50%', background: '#333', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 22 },
  label: { fontSize: 11, color: '#262626', maxWidth: 64, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', textAlign: 'center' },
  count: { position: 'absolute', top: -4, left: -4, background: '#0095f6', color: '#fff', borderRadius: '50%', width: 18, height: 18, fontSize: 10, display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700 },
}

const sv = {
  overlay: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,.9)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 300 },
  container: { position: 'relative', width: 360, maxWidth: '95vw', background: '#111', borderRadius: 12, overflow: 'hidden' },
  bar: { display: 'flex', gap: 3, padding: '10px 10px 4px', position: 'absolute', top: 0, left: 0, right: 0, zIndex: 10 },
  seg: { flex: 1, height: 2, borderRadius: 1 },
  close: { position: 'absolute', top: 24, right: 10, background: 'none', border: 'none', color: '#fff', fontSize: 20, cursor: 'pointer', zIndex: 11 },
  meta: { display: 'flex', alignItems: 'center', padding: '30px 12px 8px', position: 'relative', zIndex: 10 },
  dot: { width: 32, height: 32, borderRadius: '50%', background: '#555', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 14 },
  img: { width: '100%', maxHeight: 560, objectFit: 'cover', display: 'block' },
  noMedia: { width: '100%', height: 400, background: '#222' },
  caption: { padding: '10px 14px', margin: 0, color: '#fff', fontSize: 14 },
  nav: { display: 'flex', justifyContent: 'space-between', padding: '8px 12px' },
  navBtn: { background: 'rgba(255,255,255,.15)', border: 'none', color: '#fff', borderRadius: 6, padding: '6px 14px', fontSize: 20, cursor: 'pointer' },
}

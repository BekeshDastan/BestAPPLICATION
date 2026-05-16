import { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useAuth } from '../AuthContext'
import { userApi, postApi, chatApi } from '../api'

function Avatar({ name, size = 80 }) {
  return (
    <div style={{ width: size, height: size, borderRadius: '50%', background: '#333', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: size * 0.35, flexShrink: 0 }}>
      {name?.[0]?.toUpperCase() || '?'}
    </div>
  )
}

function UserListModal({ title, users, onClose }) {
  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={s.modal} onClick={e => e.stopPropagation()}>
        <div style={s.mHead}>
          <span style={{ fontWeight: 600 }}>{title}</span>
          <button onClick={onClose} style={s.closeBtn}>✕</button>
        </div>
        <div style={s.mBody}>
          {users.length === 0 && <p style={s.empty}>No users.</p>}
          {users.map(u => (
            <div key={u.id || u.user_id} style={s.uRow}>
              <Avatar name={u.username} size={36} />
              <div>
                <div style={{ fontWeight: 600, fontSize: 14 }}>{u.username}</div>
                {u.full_name && <div style={{ fontSize: 12, color: '#8e8e8e' }}>{u.full_name}</div>}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default function ProfilePage() {
  const { id } = useParams()
  const { user } = useAuth()
  const nav = useNavigate()
  const [profile, setProfile] = useState(null)
  const [posts, setPosts] = useState([])
  const [following, setFollowing] = useState(false)
  const [loading, setLoading] = useState(true)
  const [modal, setModal] = useState(null) // 'followers'|'following'
  const [modalUsers, setModalUsers] = useState([])
  const [activePost, setActivePost] = useState(null)

  const isMe = user?.id === id

  useEffect(() => {
    setLoading(true)
    const requests = [userApi.getProfile(id), userApi.listUserPosts(id, 30)]
    if (!isMe) requests.push(userApi.isFollowing(id).catch(() => ({ data: { is_following: false } })))
    Promise.all(requests).then(([profR, postsR, followR]) => {
      setProfile(profR.data.user || profR.data)
      setPosts(postsR.data.posts || [])
      if (followR) setFollowing(followR.data.is_following || false)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  const toggleFollow = async () => {
    try {
      if (following) await userApi.unfollow(id)
      else await userApi.follow(id)
      setFollowing(f => !f)
    } catch {}
  }

  const startChat = async () => {
    try {
      const { data } = await chatApi.create([id])
      nav('/chat', { state: { conv: data.conversation } })
    } catch {}
  }

  const openFollowers = async () => {
    const { data } = await userApi.listFollowers(id)
    setModalUsers(data.users || [])
    setModal('followers')
  }

  const openFollowing = async () => {
    const { data } = await userApi.listFollowing(id)
    setModalUsers(data.users || [])
    setModal('following')
  }

  if (loading) return <div style={s.hint}>Loading…</div>
  if (!profile) return <div style={s.hint}>User not found.</div>

  return (
    <div style={s.page}>
      <div style={s.header}>
        <Avatar name={profile.username} size={90} />
        <div style={s.info}>
          <div style={s.topRow}>
            <h2 style={s.username}>{profile.username}</h2>
            {!isMe && (
              <>
                <button onClick={toggleFollow} style={{ ...s.followBtn, background: following ? '#fff' : '#0095f6', color: following ? '#262626' : '#fff', border: following ? '1px solid #dbdbdb' : 'none' }}>
                  {following ? 'Following' : 'Follow'}
                </button>
                <button onClick={startChat} style={s.msgBtn}>Message</button>
              </>
            )}
          </div>
          <div style={s.stats}>
            <span style={s.stat}><b>{posts.length}</b> posts</span>
            <button onClick={openFollowers} style={s.statBtn}><b>{profile.followers_count || 0}</b> followers</button>
            <button onClick={openFollowing} style={s.statBtn}><b>{profile.following_count || 0}</b> following</button>
          </div>
          {profile.full_name && <div style={s.fullName}>{profile.full_name}</div>}
          {profile.bio && <div style={s.bio}>{profile.bio}</div>}
        </div>
      </div>
      <div style={s.divider} />
      <div style={s.grid}>
        {posts.map(p => (
          <div key={p.id} style={s.cell} onClick={() => setActivePost(p)}>
            {p.media_urls?.[0]
              ? <img src={p.media_urls[0]} alt="" style={s.cellImg} onError={e => e.target.style.display = 'none'} />
              : <div style={s.cellText}>{p.caption?.slice(0, 80)}</div>
            }
            <div style={s.cellOverlay}>
              <span>♡ {p.likes_count || 0}</span>
              <span>💬 {p.comments_count || 0}</span>
            </div>
          </div>
        ))}
        {posts.length === 0 && <p style={s.empty}>No posts yet.</p>}
      </div>
      {modal && <UserListModal title={modal} users={modalUsers} onClose={() => setModal(null)} />}
      {activePost && (
        <div style={s.overlay} onClick={() => setActivePost(null)}>
          <div style={s.postModal} onClick={e => e.stopPropagation()}>
            <button style={s.closeBtn} onClick={() => setActivePost(null)}>✕</button>
            {activePost.media_urls?.[0] && <img src={activePost.media_urls[0]} alt="" style={{ width: '100%', maxHeight: 500, objectFit: 'cover' }} />}
            {activePost.caption && <p style={{ padding: '12px 16px', margin: 0 }}>{activePost.caption}</p>}
          </div>
        </div>
      )}
    </div>
  )
}

const s = {
  page: { maxWidth: 935, margin: '0 auto', padding: '30px 20px' },
  header: { display: 'flex', gap: 80, alignItems: 'flex-start', marginBottom: 40 },
  info: { flex: 1 },
  topRow: { display: 'flex', alignItems: 'center', gap: 20, marginBottom: 16 },
  username: { margin: 0, fontSize: 20, fontWeight: 300 },
  followBtn: { padding: '6px 24px', borderRadius: 6, fontWeight: 600, fontSize: 14, cursor: 'pointer' },
  msgBtn: { padding: '6px 20px', borderRadius: 6, fontWeight: 600, fontSize: 14, cursor: 'pointer', background: '#efefef', border: 'none', color: '#262626' },
  stats: { display: 'flex', gap: 32, marginBottom: 12 },
  stat: { fontSize: 16 },
  statBtn: { background: 'none', border: 'none', fontSize: 16, cursor: 'pointer', padding: 0 },
  fullName: { fontWeight: 600, fontSize: 14, marginBottom: 4 },
  bio: { fontSize: 14, color: '#262626', whiteSpace: 'pre-line' },
  divider: { borderTop: '1px solid #dbdbdb', margin: '0 0 8px' },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 4 },
  cell: { position: 'relative', aspectRatio: '1', background: '#efefef', cursor: 'pointer', overflow: 'hidden' },
  cellImg: { width: '100%', height: '100%', objectFit: 'cover', display: 'block' },
  cellText: { width: '100%', height: '100%', padding: '8px', fontSize: 12, color: '#555', overflow: 'hidden' },
  cellOverlay: { position: 'absolute', inset: 0, background: 'rgba(0,0,0,.35)', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 20, opacity: 0, transition: 'opacity .2s', fontSize: 15, fontWeight: 700 },
  empty: { gridColumn: '1/-1', textAlign: 'center', color: '#8e8e8e', padding: '40px', fontSize: 14 },
  hint: { padding: '60px', textAlign: 'center', color: '#8e8e8e' },
  overlay: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,.65)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 200 },
  modal: { background: '#fff', borderRadius: 12, width: 400, maxWidth: '90vw' },
  mHead: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 16px', borderBottom: '1px solid #dbdbdb' },
  mBody: { maxHeight: 400, overflowY: 'auto', padding: '8px 0' },
  uRow: { display: 'flex', alignItems: 'center', gap: 12, padding: '8px 16px' },
  closeBtn: { background: 'none', border: 'none', fontSize: 18, cursor: 'pointer', color: '#8e8e8e' },
  postModal: { background: '#fff', borderRadius: 8, width: 600, maxWidth: '95vw', maxHeight: '90vh', overflow: 'auto', position: 'relative' },
}

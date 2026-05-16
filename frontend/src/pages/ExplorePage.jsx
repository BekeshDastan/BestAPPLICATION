import { useState, useEffect, useRef } from 'react'
import { Link } from 'react-router-dom'
import { userApi, postApi } from '../api'

function Avatar({ name, size = 40 }) {
  return (
    <div style={{ width: size, height: size, borderRadius: '50%', background: '#333', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: size * 0.38, flexShrink: 0 }}>
      {name?.[0]?.toUpperCase() || '?'}
    </div>
  )
}

export default function ExplorePage() {
  const [query, setQuery] = useState('')
  const [tab, setTab] = useState('users') // users | posts
  const [users, setUsers] = useState([])
  const [posts, setPosts] = useState([])
  const [loading, setLoading] = useState(false)
  const debounce = useRef(null)

  useEffect(() => {
    if (!query.trim()) { setUsers([]); setPosts([]); return }
    clearTimeout(debounce.current)
    debounce.current = setTimeout(async () => {
      setLoading(true)
      try {
        const [ur, pr] = await Promise.all([
          userApi.searchUsers(query),
          postApi.search(query),
        ])
        setUsers(ur.data.users || [])
        setPosts(pr.data.posts || [])
      } catch {}
      setLoading(false)
    }, 400)
  }, [query])

  return (
    <div style={s.page}>
      <div style={s.searchWrap}>
        <span style={s.searchIcon}>🔍</span>
        <input
          placeholder="Search users or posts…"
          value={query}
          onChange={e => setQuery(e.target.value)}
          style={s.input}
          autoFocus
        />
        {query && <button onClick={() => setQuery('')} style={s.clear}>✕</button>}
      </div>
      {query.trim() && (
        <div style={s.tabs}>
          <button onClick={() => setTab('users')} style={{ ...s.tab, borderBottom: tab === 'users' ? '2px solid #262626' : '2px solid transparent' }}>
            Users ({users.length})
          </button>
          <button onClick={() => setTab('posts')} style={{ ...s.tab, borderBottom: tab === 'posts' ? '2px solid #262626' : '2px solid transparent' }}>
            Posts ({posts.length})
          </button>
        </div>
      )}
      {loading && <div style={s.hint}>Searching…</div>}
      {!loading && !query.trim() && (
        <div style={s.placeholder}>
          <div style={{ fontSize: 48 }}>🔍</div>
          <p>Search for users or posts</p>
        </div>
      )}
      {!loading && query.trim() && tab === 'users' && (
        <div style={s.list}>
          {users.length === 0 && <div style={s.hint}>No users found.</div>}
          {users.map(u => (
            <Link key={u.id} to={`/profile/${u.id}`} style={s.userCard}>
              <Avatar name={u.username} size={48} />
              <div>
                <div style={s.username}>{u.username}</div>
                {u.full_name && <div style={s.fullName}>{u.full_name}</div>}
                {u.bio && <div style={s.bio}>{u.bio.slice(0, 80)}</div>}
              </div>
            </Link>
          ))}
        </div>
      )}
      {!loading && query.trim() && tab === 'posts' && (
        <div style={s.grid}>
          {posts.length === 0 && <div style={s.hint}>No posts found.</div>}
          {posts.map(p => (
            <div key={p.id} style={s.cell}>
              {p.media_urls?.[0]
                ? <img src={p.media_urls[0]} alt="" style={s.cellImg} onError={e => e.target.style.display = 'none'} />
                : <div style={s.cellText}>{p.caption?.slice(0, 100)}</div>
              }
              <div style={s.cellMeta}>
                <Link to={`/profile/${p.author_id}`} style={{ color: '#0095f6', fontSize: 12 }}>
                  @{p.author_id?.slice(0, 8)}
                </Link>
                <span style={{ fontSize: 12, color: '#8e8e8e' }}>♡ {p.likes_count || 0}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

const s = {
  page: { maxWidth: 700, margin: '0 auto', padding: '24px 16px' },
  searchWrap: { position: 'relative', display: 'flex', alignItems: 'center', marginBottom: 20 },
  searchIcon: { position: 'absolute', left: 12, fontSize: 16, color: '#8e8e8e' },
  input: { width: '100%', padding: '10px 36px 10px 38px', border: '1px solid #dbdbdb', borderRadius: 8, fontSize: 15, background: '#fafafa', outline: 'none' },
  clear: { position: 'absolute', right: 12, background: 'none', border: 'none', fontSize: 14, color: '#8e8e8e', cursor: 'pointer' },
  tabs: { display: 'flex', borderBottom: '1px solid #dbdbdb', marginBottom: 16 },
  tab: { flex: 1, padding: '10px', background: 'none', border: 'none', fontWeight: 600, fontSize: 13, cursor: 'pointer', color: '#262626' },
  list: { display: 'flex', flexDirection: 'column', gap: 1 },
  userCard: { display: 'flex', alignItems: 'center', gap: 14, padding: '12px', borderRadius: 8, textDecoration: 'none', color: 'inherit', background: '#fff', border: '1px solid #efefef', marginBottom: 8 },
  username: { fontWeight: 600, fontSize: 15 },
  fullName: { fontSize: 13, color: '#8e8e8e' },
  bio: { fontSize: 13, color: '#555', marginTop: 2 },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 4 },
  cell: { background: '#efefef', borderRadius: 4, overflow: 'hidden' },
  cellImg: { width: '100%', aspectRatio: '1', objectFit: 'cover', display: 'block' },
  cellText: { aspectRatio: '1', padding: '8px', fontSize: 12, color: '#555', overflow: 'hidden', display: 'flex', alignItems: 'flex-start' },
  cellMeta: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '4px 8px' },
  hint: { textAlign: 'center', color: '#8e8e8e', padding: '40px', fontSize: 14 },
  placeholder: { textAlign: 'center', color: '#8e8e8e', padding: '60px 20px', fontSize: 15 },
}

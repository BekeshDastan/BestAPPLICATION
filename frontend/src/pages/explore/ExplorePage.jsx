import { useEffect, useState, useRef, useCallback } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Search, Users, LayoutGrid, Hash } from 'lucide-react'
import api from '../../lib/api'
import useDebounce from '../../hooks/useDebounce'
import UserCard from '../../components/explore/UserCard'
import PostThumbnail from '../../components/explore/PostThumbnail'
import { ResponsiveMasonry } from '../../components/explore/MasonryGrid'
import PostDetailModal from '../../components/shared/PostDetailModal'

const TABS = [
  { id: 'people', label: 'People',  icon: Users },
  { id: 'posts',  label: 'Posts',   icon: LayoutGrid },
  { id: 'tags',   label: 'Tags',    icon: Hash },
]

/* ── Skeleton helpers ────────────────────────────────────────────────── */
function UserSkeleton() {
  return (
    <div className="card flex flex-col items-center gap-3 p-5">
      <div className="skeleton w-16 h-16 rounded-full" />
      <div className="space-y-1.5 w-full flex flex-col items-center">
        <div className="skeleton w-24 h-3 rounded" />
        <div className="skeleton w-16 h-2.5 rounded" />
      </div>
      <div className="skeleton w-full h-8 rounded-btn" />
    </div>
  )
}

function ThumbSkeleton() {
  return <div className="skeleton aspect-square rounded-card" />
}

/* ── Empty / no-query state ──────────────────────────────────────────── */
function DiscoverState({ onTagClick }) {
  const [users, setUsers]   = useState([])
  const [tags,  setTags]    = useState([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.allSettled([
      api.get('/users/suggestions', { params: { limit: 6 } }),
      api.get('/posts/feed',        { params: { limit: 30 } }),
    ]).then(([sugResult, feedResult]) => {
      if (sugResult.status === 'fulfilled') {
        setUsers(sugResult.value.data.users ?? [])
      }
      if (feedResult.status === 'fulfilled') {
        const posts = feedResult.value.data.posts ?? []
        const tagSet = new Set()
        posts.forEach((p) => (p.tags ?? []).forEach((t) => tagSet.add(t)))
        setTags([...tagSet].slice(0, 12))
      }
    }).finally(() => setLoading(false))
  }, [])

  return (
    <div className="space-y-8 py-4">
      {/* Trending tags */}
      {(loading || tags.length > 0) && (
        <section>
          <p className="text-xs font-semibold text-lo uppercase tracking-wider mb-3">
            Trending
          </p>
          <div className="flex flex-wrap gap-2">
            {loading
              ? Array.from({ length: 8 }).map((_, i) => (
                  <div key={i} className="skeleton w-20 h-7 rounded-full" />
                ))
              : tags.map((t) => (
                  <button
                    key={t}
                    onClick={() => onTagClick(t)}
                    className="px-3 py-1.5 rounded-full text-xs font-medium transition-colors"
                    style={{
                      background:   'var(--accent-glow)',
                      color:        'var(--accent)',
                      border:       '1px solid var(--accent)',
                    }}
                  >
                    #{t}
                  </button>
                ))
            }
          </div>
        </section>
      )}

      {/* Suggested people */}
      <section>
        <p className="text-xs font-semibold text-lo uppercase tracking-wider mb-3">
          Discover people
        </p>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {loading
            ? Array.from({ length: 6 }).map((_, i) => <UserSkeleton key={i} />)
            : users.map((u) => <UserCard key={u.id} user={u} />)
          }
          {!loading && users.length === 0 && (
            <p className="col-span-3 text-sm text-lo">
              Search for people to connect with.
            </p>
          )}
        </div>
      </section>
    </div>
  )
}

/* ── People tab ──────────────────────────────────────────────────────── */
function PeopleTab({ query }) {
  const [users,   setUsers]   = useState([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!query.trim()) { setUsers([]); return }
    setLoading(true)
    api.get('/users/search', { params: { q: query, limit: 30 } })
      .then(({ data }) => setUsers(data.users ?? []))
      .catch(() => setUsers([]))
      .finally(() => setLoading(false))
  }, [query])

  if (!query.trim()) return null // handled by DiscoverState

  if (loading) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => <UserSkeleton key={i} />)}
      </div>
    )
  }

  if (!users.length) {
    return (
      <p className="text-sm text-lo py-12 text-center">
        No people found for "{query}"
      </p>
    )
  }

  return (
    <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
      {users.map((u) => <UserCard key={u.id} user={u} />)}
    </div>
  )
}

/* ── Posts tab ───────────────────────────────────────────────────────── */
function PostsTab({ query }) {
  const [posts,    setPosts]    = useState([])
  const [loading,  setLoading]  = useState(false)
  const [selected, setSelected] = useState(null)

  useEffect(() => {
    setLoading(true)
    const params = query.trim() ? { q: query, limit: 30 } : { limit: 30 }
    const endpoint = query.trim() ? '/posts/search' : '/posts/feed'
    api.get(endpoint, { params })
      .then(({ data }) => setPosts(data.posts ?? []))
      .catch(() => setPosts([]))
      .finally(() => setLoading(false))
  }, [query])

  if (loading) {
    return (
      <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
        {Array.from({ length: 9 }).map((_, i) => <ThumbSkeleton key={i} />)}
      </div>
    )
  }

  if (!posts.length) {
    return (
      <p className="text-sm text-lo py-12 text-center">
        {query ? `No posts found for "${query}"` : 'No posts yet.'}
      </p>
    )
  }

  return (
    <>
      <ResponsiveMasonry
        items={posts}
        gap={6}
        renderItem={(p) => (
          <PostThumbnail post={p} onClick={setSelected} />
        )}
      />
      {selected && (
        <PostDetailModal post={selected} onClose={() => setSelected(null)} />
      )}
    </>
  )
}

/* ── Tags tab ────────────────────────────────────────────────────────── */
function TagsTab({ query }) {
  const tag = query.startsWith('#') ? query.slice(1) : query
  const [posts,    setPosts]    = useState([])
  const [loading,  setLoading]  = useState(false)
  const [selected, setSelected] = useState(null)

  useEffect(() => {
    if (!tag.trim()) { setPosts([]); return }
    setLoading(true)
    api.get('/posts/search', { params: { q: `#${tag}`, limit: 30 } })
      .then(({ data }) => setPosts(data.posts ?? []))
      .catch(() => setPosts([]))
      .finally(() => setLoading(false))
  }, [tag])

  if (!tag.trim()) {
    return (
      <p className="text-sm text-lo py-12 text-center">
        Search a hashtag, e.g. <strong>#photography</strong>
      </p>
    )
  }

  return (
    <>
      {/* Tag header */}
      <div className="flex items-center gap-3 mb-6">
        <div
          className="w-14 h-14 rounded-full flex items-center justify-center text-xl font-bold"
          style={{ background: 'var(--accent-glow)', color: 'var(--accent)' }}
        >
          #
        </div>
        <div>
          <p className="text-lg font-bold text-hi">#{tag}</p>
          {posts.length > 0 && (
            <p className="text-sm text-lo">{posts.length}+ posts</p>
          )}
        </div>
      </div>

      {loading ? (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
          {Array.from({ length: 9 }).map((_, i) => <ThumbSkeleton key={i} />)}
        </div>
      ) : posts.length === 0 ? (
        <p className="text-sm text-lo py-12 text-center">
          No posts found for #{tag}
        </p>
      ) : (
        <>
          <ResponsiveMasonry
            items={posts}
            gap={6}
            renderItem={(p) => (
              <PostThumbnail post={p} onClick={setSelected} />
            )}
          />
          {selected && (
            <PostDetailModal post={selected} onClose={() => setSelected(null)} />
          )}
        </>
      )}
    </>
  )
}

/* ── ExplorePage ─────────────────────────────────────────────────────── */
export default function ExplorePage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const initialTab = searchParams.get('tab') ?? 'people'

  const [query,     setQuery]     = useState(searchParams.get('q') ?? '')
  const [activeTab, setActiveTab] = useState(initialTab)
  const debouncedQ = useDebounce(query, 300)

  /* Sync tab clicks that come from tag buttons */
  function handleTagClick(tag) {
    setQuery(`#${tag}`)
    setActiveTab('tags')
  }

  function switchTab(tab) {
    setActiveTab(tab)
    setSearchParams({ tab, ...(query ? { q: query } : {}) }, { replace: true })
  }

  const showDiscover = !debouncedQ.trim() && activeTab === 'people'

  return (
    <div className="max-w-4xl mx-auto px-4 py-6">
      {/* ── Sticky search bar ──────────────────────────────────────── */}
      <div
        className="sticky top-0 z-10 pb-4 pt-0"
        style={{ background: 'var(--bg)' }}
      >
        <div className="relative mb-4">
          <Search
            size={16}
            className="absolute left-3.5 top-1/2 -translate-y-1/2 pointer-events-none"
            style={{ color: 'var(--text-2)' }}
          />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search people, posts, #tags…"
            className="input-base pl-10"
          />
        </div>

        {/* Tabs */}
        <div
          className="flex border-b"
          style={{ borderColor: 'var(--border)' }}
        >
          {TABS.map(({ id, label, icon: Icon }) => (
            <button
              key={id}
              onClick={() => switchTab(id)}
              className={`flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
                activeTab === id
                  ? 'border-accent text-accent'
                  : 'border-transparent text-lo hover:text-hi'
              }`}
            >
              <Icon size={15} />
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* ── Tab content ────────────────────────────────────────────── */}
      {showDiscover ? (
        <DiscoverState onTagClick={handleTagClick} />
      ) : (
        <>
          {activeTab === 'people' && <PeopleTab query={debouncedQ} />}
          {activeTab === 'posts'  && <PostsTab  query={debouncedQ} />}
          {activeTab === 'tags'   && <TagsTab   query={debouncedQ} />}
        </>
      )}
    </div>
  )
}

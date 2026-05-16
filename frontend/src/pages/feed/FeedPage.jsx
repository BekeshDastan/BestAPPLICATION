import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Users } from 'lucide-react'
import api from '../../lib/api'
import StoriesBar from '../../components/feed/StoriesBar'
import PostCard, { PostSkeleton } from '../../components/feed/PostCard'
import RightPanel from '../../components/feed/RightPanel'

const PAGE_SIZE = 10

/* ── Empty state ─────────────────────────────────────────────────────── */
function EmptyFeed() {
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center px-4">
      <div
        className="w-20 h-20 rounded-full flex items-center justify-center mb-6"
        style={{ background: 'var(--surface-high)' }}
      >
        <Users size={36} style={{ color: 'var(--text-2)' }} />
      </div>
      <h3 className="text-lg font-semibold text-hi mb-2">
        Nothing here yet
      </h3>
      <p className="text-sm text-lo mb-6 max-w-xs">
        Follow people to see their posts in your feed.
      </p>
      <Link to="/explore" className="btn-primary">
        Explore users →
      </Link>
    </div>
  )
}

/* ── FeedPage ────────────────────────────────────────────────────────── */
export default function FeedPage() {
  const [posts,   setPosts]   = useState([])
  const [offset,  setOffset]  = useState(0)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)
  const [initial, setInitial] = useState(true)

  /* Sentinel ref for IntersectionObserver (infinite scroll) */
  const sentinelRef = useRef(null)

  /* Gateway auto-fills following_ids from the caller's profile. */
  const load = useCallback(async (off = 0) => {
    if (loading) return
    setLoading(true)
    try {
      const { data } = await api.get('/posts/feed', {
        params: { limit: PAGE_SIZE, offset: off },
      })
      const fresh = data.posts ?? []
      setPosts((prev) => (off === 0 ? fresh : [...prev, ...fresh]))
      setOffset(off + fresh.length)
      setHasMore(fresh.length === PAGE_SIZE)
    } catch {
      setHasMore(false)
    } finally {
      setLoading(false)
      setInitial(false)
    }
  }, [loading])

  /* Initial load */
  useEffect(() => { load(0) }, []) // eslint-disable-line

  /* Infinite scroll observer */
  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) {
          load(offset)
        }
      },
      { threshold: 0.8 },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [offset, hasMore, loading, load])

  function handleDelete(id) {
    setPosts((prev) => prev.filter((p) => p.id !== id))
  }

  return (
    <div className="flex flex-col min-h-dvh">
      {/* Sticky stories bar */}
      <StoriesBar />

      {/* Content row */}
      <div className="flex flex-1 gap-6 px-4 py-6 max-w-5xl mx-auto w-full">

        {/* ── Feed column ─────────────────────────────────────────── */}
        <div className="flex-1 min-w-0 max-w-[680px] mx-auto space-y-4">

          {/* Initial skeleton */}
          {initial &&
            Array.from({ length: 3 }).map((_, i) => <PostSkeleton key={i} />)}

          {/* Posts */}
          {!initial && posts.length === 0
            ? <EmptyFeed />
            : posts.map((post, idx) => (
                <PostCard
                  key={post.id}
                  post={post}
                  onDelete={handleDelete}
                  lastRef={idx === posts.length - 1 ? undefined : undefined}
                />
              ))
          }

          {/* Sentinel for infinite scroll */}
          <div ref={sentinelRef} className="h-4" />

          {/* Load-more skeleton */}
          {loading && !initial && (
            <PostSkeleton />
          )}

          {/* End of feed */}
          {!hasMore && posts.length > 0 && (
            <p className="text-center text-xs text-lo py-8">
              You've seen everything ✓
            </p>
          )}
        </div>

        {/* ── Right panel (desktop) ────────────────────────────── */}
        <RightPanel />
      </div>
    </div>
  )
}

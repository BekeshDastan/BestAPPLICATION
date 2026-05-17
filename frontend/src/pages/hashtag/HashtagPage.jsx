import { useEffect, useRef, useCallback, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Hash } from 'lucide-react'
import api from '../../lib/api'
import PostThumbnail from '../../components/explore/PostThumbnail'
import { ResponsiveMasonry } from '../../components/explore/MasonryGrid'
import PostDetailModal from '../../components/shared/PostDetailModal'

function ThumbSkeleton() {
  return <div className="skeleton aspect-square rounded-card" />
}

const PAGE_SIZE = 30

export default function HashtagPage() {
  const { tag } = useParams()

  const [posts,    setPosts]    = useState([])
  const [cursor,   setCursor]   = useState(null)
  const [hasMore,  setHasMore]  = useState(true)
  const [loading,  setLoading]  = useState(false)
  const [initial,  setInitial]  = useState(true)
  const [selected, setSelected] = useState(null)

  const sentinelRef = useRef(null)

  const load = useCallback(async (c = null) => {
    if (loading) return
    setLoading(true)
    try {
      const params = { q: `#${tag}`, limit: PAGE_SIZE }
      if (c) params.cursor = c
      const { data } = await api.get('/posts/search', { params })
      const fresh = data.posts ?? []
      setPosts((prev) => (c ? [...prev, ...fresh] : fresh))
      setCursor(data.next_cursor ?? null)
      setHasMore(!!data.next_cursor && fresh.length === PAGE_SIZE)
    } catch {
      setHasMore(false)
    } finally {
      setLoading(false)
      setInitial(false)
    }
  }, [tag, loading]) // eslint-disable-line react-hooks/exhaustive-deps

  /* Initial load when tag changes */
  useEffect(() => {
    setPosts([])
    setCursor(null)
    setHasMore(true)
    setInitial(true)
    load(null)
  }, [tag]) // eslint-disable-line react-hooks/exhaustive-deps

  /* Infinite scroll */
  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loading) load(cursor)
      },
      { threshold: 0.8 },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [cursor, hasMore, loading, load])

  return (
    <div className="max-w-4xl mx-auto px-4 py-6">
      {/* Back */}
      <Link
        to="/explore"
        className="inline-flex items-center gap-1.5 text-sm text-lo hover:text-hi mb-5 transition-colors"
      >
        <ArrowLeft size={16} /> Back to Explore
      </Link>

      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <div
          className="w-16 h-16 rounded-full flex items-center justify-center text-2xl font-bold shrink-0"
          style={{ background: 'var(--accent-glow)', color: 'var(--accent)' }}
        >
          <Hash size={28} />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-hi">#{tag}</h1>
          {!initial && (
            <p className="text-sm text-lo">
              {posts.length}{hasMore ? '+' : ''} posts
            </p>
          )}
        </div>
      </div>

      {/* Grid */}
      {initial ? (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
          {Array.from({ length: 9 }).map((_, i) => (
            <ThumbSkeleton key={i} />
          ))}
        </div>
      ) : posts.length === 0 ? (
        <div className="py-24 text-center">
          <p className="text-sm text-lo mb-4">No posts for #{tag} yet.</p>
          <Link to="/explore" className="btn-primary inline-flex">
            Browse Explore
          </Link>
        </div>
      ) : (
        <ResponsiveMasonry
          items={posts}
          gap={6}
          renderItem={(p) => (
            <PostThumbnail post={p} onClick={setSelected} />
          )}
        />
      )}

      {/* Sentinel */}
      <div ref={sentinelRef} className="h-4 mt-2" />

      {/* Load-more skeleton */}
      {loading && !initial && (
        <div className="grid grid-cols-2 md:grid-cols-3 gap-2 mt-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <ThumbSkeleton key={i} />
          ))}
        </div>
      )}

      {!hasMore && posts.length > 0 && (
        <p className="text-center text-xs text-lo py-8">
          All posts loaded ✓
        </p>
      )}

      {/* Detail modal */}
      {selected && (
        <PostDetailModal post={selected} onClose={() => setSelected(null)} />
      )}
    </div>
  )
}

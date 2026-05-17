import { useState, useEffect, useRef, useCallback } from 'react'
import { Bookmark } from 'lucide-react'
import api from '../../lib/api'
import PostThumbnail from '../../components/explore/PostThumbnail'
import PostDetailModal from '../../components/shared/PostDetailModal'

const PAGE = 30

function ThumbSkeleton() {
  return <div className="skeleton aspect-square rounded-card" />
}

export default function SavedPage() {
  const [posts,    setPosts]    = useState([])
  const [offset,   setOffset]   = useState(0)
  const [hasMore,  setHasMore]  = useState(true)
  const [loading,  setLoading]  = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [selected, setSelected] = useState(null)

  const sentinelRef = useRef(null)

  const load = useCallback(async (off = 0) => {
    if (off > 0) setLoadingMore(true)
    else         setLoading(true)
    try {
      const { data } = await api.get('/posts/saved', {
        params: { limit: PAGE, offset: off },
      })
      const fresh = data.posts ?? []
      setPosts((prev) => (off === 0 ? fresh : [...prev, ...fresh]))
      setOffset(off + fresh.length)
      setHasMore(fresh.length === PAGE)
    } catch {
      setHasMore(false)
    } finally {
      if (off > 0) setLoadingMore(false)
      else         setLoading(false)
    }
  }, [])

  useEffect(() => { load(0) }, [load])

  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const observer = new IntersectionObserver(
      ([entry]) => { if (entry.isIntersecting && hasMore && !loadingMore) load(offset) },
      { threshold: 0.8 },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [offset, hasMore, loadingMore, load])

  function handleUnsave(postId) {
    setPosts((prev) => prev.filter((p) => p.id !== postId))
    setSelected(null)
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-6">
      {/* Header */}
      <div className="flex items-center gap-3 mb-6">
        <Bookmark size={22} style={{ color: 'var(--accent)' }} />
        <div>
          <h1 className="text-2xl font-bold text-hi">Saved Posts</h1>
          {!loading && (
            <p className="text-sm text-lo">
              {posts.length}{hasMore ? '+' : ''} saved
            </p>
          )}
        </div>
      </div>

      {/* Grid */}
      {loading ? (
        <div className="grid grid-cols-3 gap-1.5">
          {Array.from({ length: 9 }).map((_, i) => <ThumbSkeleton key={i} />)}
        </div>
      ) : posts.length === 0 ? (
        <EmptyState />
      ) : (
        <div className="grid grid-cols-3 gap-1.5">
          {posts.map((p) => (
            <PostThumbnail key={p.id} post={p} onClick={setSelected} />
          ))}
        </div>
      )}

      {/* Sentinel */}
      <div ref={sentinelRef} className="h-4 mt-2" />

      {loadingMore && (
        <div className="grid grid-cols-3 gap-1.5 mt-1.5">
          {Array.from({ length: 6 }).map((_, i) => <ThumbSkeleton key={i} />)}
        </div>
      )}

      {!hasMore && posts.length > 0 && (
        <p className="text-center text-xs text-lo py-8">All saved posts loaded ✓</p>
      )}

      {selected && (
        <PostDetailModal
          post={selected}
          onClose={() => setSelected(null)}
          onUnsave={handleUnsave}
        />
      )}
    </div>
  )
}

function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-24 gap-4 text-center">
      <div
        className="w-20 h-20 rounded-full flex items-center justify-center"
        style={{ background: 'var(--surface-high)' }}
      >
        <Bookmark size={36} style={{ color: 'var(--accent)' }} />
      </div>
      <div>
        <p className="font-semibold text-hi text-lg mb-1">No saved posts yet</p>
        <p className="text-sm text-lo max-w-xs">
          Save posts to find them later. Tap the bookmark icon on any post.
        </p>
      </div>
    </div>
  )
}

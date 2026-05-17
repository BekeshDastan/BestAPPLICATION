import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus } from 'lucide-react'
import api from '../../lib/api'
import Avatar from '../shared/Avatar'
import StoryCreationModal from '../stories/StoryCreationModal'

function groupByUser(stories) {
  const map = new Map()
  for (const s of stories) {
    if (!map.has(s.user_id)) {
      map.set(s.user_id, {
        user_id:    s.user_id,
        username:   s.author?.username  ?? s.user_id.slice(0, 8),
        avatar_url: s.author?.avatar_url ?? null,
        full_name:  s.author?.full_name  ?? null,
        seen:       false,
        stories:    [],
      })
    }
    const g = map.get(s.user_id)
    g.stories.push(s)
  }
  return [...map.values()]
}

function StorySkeleton() {
  return (
    <div className="flex flex-col items-center gap-1.5 shrink-0">
      <div className="skeleton w-14 h-14 rounded-full" />
      <div className="skeleton w-10 h-2.5 rounded" />
    </div>
  )
}

export default function StoriesBar() {
  const navigate = useNavigate()
  const [groups,      setGroups]      = useState([])
  const [loading,     setLoading]     = useState(true)
  const [showCreate,  setShowCreate]  = useState(false)

  useEffect(() => {
    api
      .get('/stories/following', { params: { limit: 30 } })
      .then(({ data }) => {
        setGroups(groupByUser(data.stories ?? []))
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  return (
    <>
    <div
      className="sticky top-0 z-10 border-b"
      style={{ background: 'var(--bg)', borderColor: 'var(--border)' }}
    >
      <div
        className="flex items-center gap-4 px-4 py-3 overflow-x-auto"
        style={{ scrollbarWidth: 'none' }}
      >
        {/* Your Story */}
        <button
          onClick={() => setShowCreate(true)}
          className="flex flex-col items-center gap-1.5 shrink-0 group"
        >
          <div
            className="w-14 h-14 rounded-full border-2 border-dashed flex items-center justify-center transition-colors"
            style={{ borderColor: 'var(--border)', background: 'var(--surface-high)' }}
          >
            <Plus
              size={20}
              style={{ color: 'var(--accent)' }}
              className="group-hover:scale-110 transition-transform"
            />
          </div>
          <span className="text-[10px] text-lo font-medium">Your Story</span>
        </button>

        {/* Skeleton */}
        {loading &&
          Array.from({ length: 6 }).map((_, i) => <StorySkeleton key={i} />)}

        {/* Story groups */}
        {!loading &&
          groups.map((g) => (
            <button
              key={g.user_id}
              onClick={() => navigate(`/stories?user=${g.user_id}`)}
              className="flex flex-col items-center gap-1.5 shrink-0"
            >
              {/* Ring: purple = unseen, gray = all seen */}
              <div
                className="p-0.5 rounded-full"
                style={{
                  background: g.seen
                    ? 'var(--border)'
                    : 'linear-gradient(135deg, #7C3AED, #A855F7)',
                }}
              >
                <div
                  className="p-0.5 rounded-full"
                  style={{ background: 'var(--bg)' }}
                >
                  <Avatar
                    src={g.avatar_url}
                    name={g.full_name ?? g.username}
                    size={48}
                  />
                </div>
              </div>
              <span className="text-[10px] text-lo font-medium w-14 truncate text-center">
                {g.username}
              </span>
            </button>
          ))}
      </div>
    </div>

    {showCreate && (
      <StoryCreationModal
        onClose={() => setShowCreate(false)}
        onCreated={() => {
          setShowCreate(false)
          api.get('/stories/following', { params: { limit: 30 } })
            .then(({ data }) => setGroups(groupByUser(data.stories ?? [])))
            .catch(() => {})
        }}
      />
    )}
  </>
  )
}

import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  BadgeCheck, Lock, MoreHorizontal, MessageCircle,
  Bookmark, Grid3x3, Layers, UserMinus,
} from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import { formatCount } from '../../lib/utils'
import Avatar from '../../components/shared/Avatar'
import PostThumbnail from '../../components/explore/PostThumbnail'
import { ResponsiveMasonry } from '../../components/explore/MasonryGrid'
import PostDetailModal from '../../components/shared/PostDetailModal'
import FollowersModal from '../../components/profile/FollowersModal'
import HighlightViewerModal from '../../components/profile/HighlightViewerModal'

const PAGE_SIZE = 18

/* ── Header skeleton ─────────────────────────────────────────────────── */
function HeaderSkeleton() {
  return (
    <div className="card p-6 mb-6">
      <div className="flex gap-6 items-start flex-wrap">
        <div className="skeleton w-24 h-24 rounded-full shrink-0" />
        <div className="flex-1 min-w-[200px] space-y-3">
          <div className="skeleton w-40 h-5 rounded" />
          <div className="skeleton w-24 h-3.5 rounded" />
          <div className="skeleton w-56 h-3 rounded" />
          <div className="flex gap-6">
            {[1, 2, 3].map((i) => (
              <div key={i} className="skeleton w-16 h-4 rounded" />
            ))}
          </div>
          <div className="flex gap-2">
            <div className="skeleton w-28 h-9 rounded-btn" />
            <div className="skeleton w-28 h-9 rounded-btn" />
          </div>
        </div>
      </div>
    </div>
  )
}

/* ── Stat button ─────────────────────────────────────────────────────── */
function Stat({ label, value, onClick }) {
  return (
    <button
      onClick={onClick}
      className="flex flex-col items-center gap-0.5 hover:opacity-70 transition-opacity"
    >
      <span className="text-base font-bold text-hi">{formatCount(value ?? 0)}</span>
      <span className="text-xs text-lo">{label}</span>
    </button>
  )
}

/* ── Profile header ──────────────────────────────────────────────────── */
function ProfileHeader({ profile, isOwn, onOpenFollowers }) {
  const navigate = useNavigate()
  const { user: me } = useAuthStore()

  const [isFollowing, setIsFollowing] = useState(profile.is_following ?? false)
  const [isBlocked,   setIsBlocked]   = useState(profile.is_blocked   ?? false)
  const [menuOpen,    setMenuOpen]    = useState(false)
  const [loadingFollow, setLoadingFollow] = useState(false)

  /* Fetch is-following on mount for other profiles */
  useEffect(() => {
    if (isOwn) return
    api.get(`/users/${profile.id}/is-following`)
      .then(({ data }) => setIsFollowing(data.is_following ?? false))
      .catch(() => {})
  }, [profile.id, isOwn])

  async function toggleFollow() {
    setLoadingFollow(true)
    try {
      if (isFollowing) {
        await api.delete(`/users/${profile.id}/follow`)
        setIsFollowing(false)
      } else {
        await api.post(`/users/${profile.id}/follow`)
        setIsFollowing(true)
      }
    } catch (err) {
      const status = err?.response?.status
      if (status === 409) {
        // backend says already following — sync UI
        setIsFollowing(true)
      } else if (status === 404) {
        toast.error('User not found')
      } else {
        toast.error('Action failed')
      }
    } finally { setLoadingFollow(false) }
  }

  async function handleMessage() {
    try {
      const { data } = await api.post('/chats', {
        member_ids: [profile.id],
      })
      const convId = data.conversation?.id ?? data.id
      navigate(`/chat/${convId}`)
    } catch { toast.error('Could not start conversation') }
  }

  async function toggleBlock() {
    const confirmed = window.confirm(
      isBlocked
        ? `Unblock @${profile.username}?`
        : `Block @${profile.username}? They won't be able to see your posts.`,
    )
    if (!confirmed) return
    try {
      if (isBlocked) await api.delete(`/users/${profile.id}/block`)
      else           await api.post(`/users/${profile.id}/block`)
      setIsBlocked((v) => !v)
      toast.success(isBlocked ? 'Unblocked' : 'Blocked')
    } catch { toast.error('Action failed') }
    setMenuOpen(false)
  }

  return (
    <div className="card p-6 mb-6">
      <div className="flex gap-6 items-start flex-wrap">
        {/* Avatar */}
        <div className="shrink-0">
          {isOwn ? (
            <Link to="/settings?tab=profile">
              <div className="relative group">
                <Avatar
                  src={profile.avatar_url}
                  name={profile.full_name ?? profile.username}
                  size={96}
                />
                <div className="absolute inset-0 rounded-full bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                  <span className="text-white text-xs font-medium">Edit</span>
                </div>
              </div>
            </Link>
          ) : (
            <Avatar
              src={profile.avatar_url}
              name={profile.full_name ?? profile.username}
              size={96}
            />
          )}
        </div>

        {/* Info */}
        <div className="flex-1 min-w-[200px] space-y-2">
          {/* Name row */}
          <div className="flex items-center gap-2 flex-wrap">
            <h1 className="text-xl font-bold text-hi">
              {profile.full_name ?? profile.username}
            </h1>
            {profile.is_verified && (
              <BadgeCheck size={20} style={{ color: 'var(--accent)' }} />
            )}
          </div>

          {/* Username + private */}
          <div className="flex items-center gap-1.5">
            <span className="text-sm text-lo">@{profile.username}</span>
            {profile.is_private && (
              <Lock size={13} style={{ color: 'var(--text-2)' }} />
            )}
          </div>

          {/* Bio */}
          {profile.bio && (
            <p className="text-sm text-hi leading-relaxed max-w-sm">{profile.bio}</p>
          )}

          {/* Stats */}
          <div className="flex items-center gap-6 pt-1">
            <Stat label="Posts"     value={profile.posts_count}     onClick={() => {}} />
            <Stat label="Followers" value={profile.followers_count} onClick={() => onOpenFollowers('followers')} />
            <Stat label="Following" value={profile.following_count} onClick={() => onOpenFollowers('following')} />
          </div>

          {/* Action buttons */}
          <div className="flex items-center gap-2 pt-1 flex-wrap">
            {isOwn ? (
              <Link to="/settings?tab=profile" className="btn-ghost text-sm px-5 py-2">
                Edit Profile
              </Link>
            ) : (
              <>
                <button
                  onClick={toggleFollow}
                  disabled={loadingFollow}
                  className={loadingFollow
                    ? 'btn-ghost text-sm px-5 py-2 opacity-50'
                    : isFollowing
                      ? 'btn-ghost text-sm px-5 py-2'
                      : 'btn-primary text-sm px-5 py-2'
                  }
                >
                  {isFollowing ? 'Following' : 'Follow'}
                </button>

                <button
                  onClick={handleMessage}
                  className="btn-ghost text-sm px-4 py-2 flex items-center gap-1.5"
                >
                  <MessageCircle size={15} /> Message
                </button>

                {/* ⋯ block menu */}
                <div className="relative">
                  <button
                    onClick={() => setMenuOpen((v) => !v)}
                    className="btn-ghost p-2"
                  >
                    <MoreHorizontal size={18} />
                  </button>
                  {menuOpen && (
                    <>
                      <div className="fixed inset-0 z-10" onClick={() => setMenuOpen(false)} />
                      <div
                        className="absolute left-0 top-10 z-20 w-40 rounded-card border py-1 shadow-card"
                        style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}
                      >
                        <button
                          onClick={toggleBlock}
                          className="flex items-center gap-2 w-full px-4 py-2 text-sm hover:bg-elevated transition-colors"
                          style={{ color: isBlocked ? 'var(--text-1)' : 'var(--danger)' }}
                        >
                          <UserMinus size={14} />
                          {isBlocked ? 'Unblock' : 'Block'}
                        </button>
                      </div>
                    </>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

/* ── Posts grid (with infinite scroll) ──────────────────────────────── */
function PostsGrid({ userId }) {
  const [posts,    setPosts]    = useState([])
  const [offset,   setOffset]   = useState(0)
  const [hasMore,  setHasMore]  = useState(true)
  const [loading,  setLoading]  = useState(false)
  const [initial,  setInitial]  = useState(true)
  const [selected, setSelected] = useState(null)
  const sentinelRef = useRef(null)

  const load = useCallback(async (off = 0) => {
    if (loading) return
    setLoading(true)
    try {
      const { data } = await api.get(`/users/${userId}/posts`, {
        params: { limit: PAGE_SIZE, offset: off },
      })
      const fresh = data.posts ?? []
      setPosts((prev) => (off === 0 ? fresh : [...prev, ...fresh]))
      setOffset(off + fresh.length)
      setHasMore(fresh.length === PAGE_SIZE)
    } catch { setHasMore(false) }
    finally { setLoading(false); setInitial(false) }
  }, [userId, loading]) // eslint-disable-line

  useEffect(() => { setPosts([]); setOffset(0); setHasMore(true); setInitial(true); load(0) }, [userId]) // eslint-disable-line

  useEffect(() => {
    const el = sentinelRef.current
    if (!el) return
    const obs = new IntersectionObserver(
      (entries) => { if (entries[0].isIntersecting && hasMore && !loading) load(offset) },
      { threshold: 0.8 },
    )
    obs.observe(el)
    return () => obs.disconnect()
  }, [offset, hasMore, loading, load])

  if (initial) {
    return (
      <div className="grid grid-cols-3 gap-1">
        {Array.from({ length: 9 }).map((_, i) => (
          <div key={i} className="skeleton aspect-square rounded-card" />
        ))}
      </div>
    )
  }

  if (!posts.length) {
    return (
      <div className="py-20 text-center">
        <Grid3x3 size={32} className="mx-auto mb-3 text-lo" style={{ color: 'var(--text-2)' }} />
        <p className="text-sm text-lo">No posts yet.</p>
      </div>
    )
  }

  return (
    <>
      <ResponsiveMasonry
        items={posts}
        gap={4}
        renderItem={(p) => <PostThumbnail post={p} onClick={setSelected} />}
      />
      <div ref={sentinelRef} className="h-4" />
      {loading && !initial && (
        <div className="grid grid-cols-3 gap-1 mt-1">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="skeleton aspect-square rounded-card" />
          ))}
        </div>
      )}
      {!hasMore && posts.length > 0 && (
        <p className="text-center text-xs text-lo py-6">All posts loaded ✓</p>
      )}
      {selected && (
        <PostDetailModal post={selected} onClose={() => setSelected(null)} />
      )}
    </>
  )
}

/* ── Saved grid ──────────────────────────────────────────────────────── */
function SavedGrid() {
  const [posts,    setPosts]    = useState([])
  const [loading,  setLoading]  = useState(true)
  const [selected, setSelected] = useState(null)

  useEffect(() => {
    api.get('/posts/saved', { params: { limit: 30 } })
      .then(({ data }) => setPosts(data.posts ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className="grid grid-cols-3 gap-1">
        {Array.from({ length: 9 }).map((_, i) => (
          <div key={i} className="skeleton aspect-square rounded-card" />
        ))}
      </div>
    )
  }

  if (!posts.length) {
    return (
      <div className="py-20 text-center">
        <Bookmark size={32} className="mx-auto mb-3" style={{ color: 'var(--text-2)' }} />
        <p className="text-sm text-lo">Nothing saved yet.</p>
      </div>
    )
  }

  return (
    <>
      <ResponsiveMasonry
        items={posts}
        gap={4}
        renderItem={(p) => <PostThumbnail post={p} onClick={setSelected} />}
      />
      {selected && (
        <PostDetailModal post={selected} onClose={() => setSelected(null)} />
      )}
    </>
  )
}

/* ── Highlights row ──────────────────────────────────────────────────── */
function HighlightsRow({ userId }) {
  const [highlights, setHighlights] = useState([])
  const [loading,    setLoading]    = useState(true)
  const [viewing,    setViewing]    = useState(null)

  useEffect(() => {
    api.get(`/highlights/user/${userId}`)
      .then(({ data }) => setHighlights(data.highlights ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [userId])

  if (loading) {
    return (
      <div className="flex gap-4 overflow-x-auto pb-2" style={{ scrollbarWidth: 'none' }}>
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="flex flex-col items-center gap-2 shrink-0">
            <div className="skeleton w-20 h-20 rounded-full" />
            <div className="skeleton w-14 h-2.5 rounded" />
          </div>
        ))}
      </div>
    )
  }

  if (!highlights.length) {
    return <p className="text-sm text-lo py-8 text-center">No highlights yet.</p>
  }

  return (
    <>
      <div className="flex gap-5 overflow-x-auto pb-2" style={{ scrollbarWidth: 'none' }}>
        {highlights.map((h) => (
          <button
            key={h.id}
            onClick={() => setViewing(h)}
            className="flex flex-col items-center gap-2 shrink-0 group"
          >
            <div
              className="w-20 h-20 rounded-full overflow-hidden border-2 transition-all group-hover:scale-105"
              style={{ borderColor: 'var(--accent)' }}
            >
              {h.cover_url ? (
                <img
                  src={h.cover_url}
                  alt={h.title}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div
                  className="w-full h-full flex items-center justify-center text-2xl"
                  style={{ background: 'var(--accent-glow)' }}
                >
                  <Layers size={24} style={{ color: 'var(--accent)' }} />
                </div>
              )}
            </div>
            <span className="text-xs text-lo w-20 truncate text-center">{h.title}</span>
          </button>
        ))}
      </div>
      {viewing && (
        <HighlightViewerModal
          highlight={viewing}
          onClose={() => setViewing(null)}
        />
      )}
    </>
  )
}

/* ── ProfilePage ─────────────────────────────────────────────────────── */
const TABS = [
  { id: 'posts',      label: 'Posts',      icon: Grid3x3 },
  { id: 'saved',      label: 'Saved',      icon: Bookmark,  ownOnly: true },
  { id: 'highlights', label: 'Highlights', icon: Layers },
]

export default function ProfilePage() {
  const { id }     = useParams()
  const { user: me } = useAuthStore()

  const profileId = id ?? me?.id
  const isOwn     = !id || id === me?.id

  const [profile,    setProfile]    = useState(null)
  const [loading,    setLoading]    = useState(true)
  const [activeTab,  setActiveTab]  = useState('posts')
  const [followersModal, setFollowersModal] = useState(null) // 'followers' | 'following' | null

  useEffect(() => {
    if (!profileId) return
    setLoading(true)
    const endpoint = isOwn ? '/users/me' : `/users/${profileId}`
    api.get(endpoint)
      .then(({ data }) => setProfile(data))
      .catch(() => toast.error('Profile not found'))
      .finally(() => setLoading(false))
  }, [profileId, isOwn])

  const visibleTabs = TABS.filter((t) => !t.ownOnly || isOwn)

  /* Private account guard */
  const isPrivateOther = profile?.is_private && !isOwn && !profile?.is_following

  if (loading) return (
    <div className="max-w-3xl mx-auto px-4 py-6">
      <HeaderSkeleton />
    </div>
  )

  if (!profile) return (
    <div className="max-w-3xl mx-auto px-4 py-6">
      <div className="card p-12 text-center">
        <p className="text-hi font-semibold mb-2">User not found</p>
        <Link to="/" className="btn-primary inline-flex mt-4">Go home</Link>
      </div>
    </div>
  )

  return (
    <div className="max-w-3xl mx-auto px-4 py-6">
      {/* Header */}
      <ProfileHeader
        profile={profile}
        isOwn={isOwn}
        onOpenFollowers={(type) => setFollowersModal(type)}
      />

      {/* Tabs */}
      <div
        className="flex border-b mb-6"
        style={{ borderColor: 'var(--border)' }}
      >
        {visibleTabs.map(({ id: tid, label, icon: Icon }) => (
          <button
            key={tid}
            onClick={() => setActiveTab(tid)}
            className={`flex items-center gap-1.5 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors ${
              activeTab === tid
                ? 'border-accent text-accent'
                : 'border-transparent text-lo hover:text-hi'
            }`}
          >
            <Icon size={15} />
            {label}
          </button>
        ))}
      </div>

      {/* Private lock screen */}
      {isPrivateOther && activeTab === 'posts' ? (
        <div className="card py-20 text-center">
          <Lock size={36} className="mx-auto mb-4" style={{ color: 'var(--text-2)' }} />
          <p className="font-semibold text-hi mb-1">This account is private.</p>
          <p className="text-sm text-lo">Follow to see their posts.</p>
        </div>
      ) : (
        <>
          {activeTab === 'posts'      && <PostsGrid userId={profileId} />}
          {activeTab === 'saved'      && isOwn && <SavedGrid />}
          {activeTab === 'highlights' && <HighlightsRow userId={profileId} />}
        </>
      )}

      {/* Followers / Following modal */}
      {followersModal && (
        <FollowersModal
          userId={profileId}
          type={followersModal}
          onClose={() => setFollowersModal(null)}
        />
      )}
    </div>
  )
}

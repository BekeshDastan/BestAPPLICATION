import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../shared/Avatar'
import { formatCount } from '../../lib/utils'

export default function UserCard({ user }) {
  const [following, setFollowing] = useState(user.is_following ?? false)
  const [loading,   setLoading]   = useState(false)

  async function toggle() {
    setLoading(true)
    try {
      if (following) await api.delete(`/users/${user.id}/follow`)
      else           await api.post(`/users/${user.id}/follow`)
      setFollowing((v) => !v)
    } catch {
      toast.error('Action failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card flex flex-col items-center gap-3 p-5 text-center transition-shadow hover:shadow-accent">
      <Link to={`/profile/${user.id}`}>
        <Avatar
          src={user.avatar_url}
          name={user.full_name ?? user.username}
          size={64}
        />
      </Link>

      <div className="space-y-0.5 w-full">
        <Link
          to={`/profile/${user.id}`}
          className="block text-sm font-semibold text-hi hover:underline truncate"
        >
          {user.full_name ?? user.username}
        </Link>
        <p className="text-xs text-lo truncate">@{user.username}</p>
        {user.followers_count != null && (
          <p className="text-xs text-lo">
            {formatCount(user.followers_count)} followers
          </p>
        )}
      </div>

      <button
        onClick={toggle}
        disabled={loading}
        className={`w-full text-sm py-1.5 rounded-btn font-semibold transition-all disabled:opacity-40 ${
          following ? 'btn-ghost' : 'btn-primary'
        }`}
      >
        {loading
          ? <Loader2 size={14} className="animate-spin mx-auto" />
          : following ? 'Following' : 'Follow'
        }
      </button>
    </div>
  )
}

import { useState, useEffect } from 'react'
import { Search, Plus } from 'lucide-react'
import api from '../../lib/api'
import Avatar from '../shared/Avatar'
import { formatRelativeTime } from '../../lib/utils'

export default function ConversationList({ selectedId, onSelect, onNewChat }) {
  const [convs,   setConvs]   = useState([])
  const [loading, setLoading] = useState(true)
  const [query,   setQuery]   = useState('')

  useEffect(() => {
    api.get('/chats')
      .then(({ data }) => setConvs(data.conversations ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const filtered = query.trim()
    ? convs.filter((c) =>
        c.name?.toLowerCase().includes(query.toLowerCase()) ||
        c.other_user?.username?.toLowerCase().includes(query.toLowerCase()) ||
        c.other_user?.full_name?.toLowerCase().includes(query.toLowerCase())
      )
    : convs

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="px-4 pt-4 pb-3 shrink-0">
        <div className="flex items-center justify-between mb-3">
          <h2 className="font-bold text-hi text-lg">Messages</h2>
          <button
            onClick={onNewChat}
            className="w-8 h-8 rounded-full flex items-center justify-center hover:bg-elevated transition-colors"
            title="New chat"
          >
            <Plus size={18} style={{ color: 'var(--accent)' }} />
          </button>
        </div>
        <div className="relative">
          <Search
            size={14}
            className="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none"
            style={{ color: 'var(--text-2)' }}
          />
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search conversations..."
            className="input-base pl-9 py-2 text-sm"
          />
        </div>
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto">
        {loading
          ? Array.from({ length: 6 }).map((_, i) => <ConvSkeleton key={i} />)
          : filtered.length === 0
            ? (
              <p className="text-center text-lo text-sm py-10">
                {query ? 'No results.' : 'No conversations yet.'}
              </p>
            )
            : filtered.map((c) => (
              <ConvRow
                key={c.id}
                conv={c}
                active={c.id === selectedId}
                onClick={() => onSelect(c)}
              />
            ))
        }
      </div>
    </div>
  )
}

function ConvRow({ conv, active, onClick }) {
  const displayName = conv.is_group
    ? conv.name
    : conv.other_user?.full_name ?? conv.other_user?.username ?? 'Unknown'
  const avatarSrc  = conv.is_group ? null : conv.other_user?.avatar_url
  const avatarName = conv.is_group ? conv.name : displayName
  const unread     = conv.unread_count ?? 0
  const isOnline   = !conv.is_group && conv.other_user?.is_online

  return (
    <button
      onClick={onClick}
      className="w-full flex items-center gap-3 px-4 py-3 text-left hover:bg-elevated transition-colors"
      style={{ background: active ? 'var(--surface-high)' : 'transparent' }}
    >
      <div className="relative shrink-0">
        <Avatar src={avatarSrc} name={avatarName} size={44} />
        {isOnline && (
          <span
            className="absolute bottom-0 right-0 w-3 h-3 rounded-full border-2"
            style={{ background: 'var(--online)', borderColor: 'var(--surface)' }}
          />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between">
          <span className="text-sm font-semibold text-hi truncate">{displayName}</span>
          {conv.last_message && (
            <span className="text-[11px] text-lo shrink-0 ml-1">
              {formatRelativeTime(conv.last_message.created_at)}
            </span>
          )}
        </div>
        <div className="flex items-center justify-between">
          <p className="text-xs text-lo truncate">
            {conv.last_message
              ? conv.is_group
                ? `${conv.last_message.sender?.username ?? '?'}: ${conv.last_message.text ?? 'Media'}`
                : (conv.last_message.text ?? 'Media')
              : 'No messages yet'}
          </p>
          {unread > 0 && (
            <span
              className="shrink-0 ml-1.5 min-w-[18px] h-[18px] rounded-full text-white text-[10px] font-bold flex items-center justify-center px-1"
              style={{ background: 'var(--accent)' }}
            >
              {unread > 99 ? '99+' : unread}
            </span>
          )}
        </div>
      </div>
    </button>
  )
}

function ConvSkeleton() {
  return (
    <div className="flex items-center gap-3 px-4 py-3">
      <div className="skeleton w-11 h-11 rounded-full shrink-0" />
      <div className="flex-1 space-y-1.5">
        <div className="skeleton w-28 h-3 rounded" />
        <div className="skeleton w-40 h-2.5 rounded" />
      </div>
    </div>
  )
}

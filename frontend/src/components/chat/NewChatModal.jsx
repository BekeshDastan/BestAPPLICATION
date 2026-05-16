import { useState, useEffect } from 'react'
import { X, Search, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../shared/Avatar'
import useDebounce from '../../hooks/useDebounce'

export default function NewChatModal({ onClose, onConversationCreated }) {
  const [query,    setQuery]    = useState('')
  const [users,    setUsers]    = useState([])
  const [loading,  setLoading]  = useState(false)
  const [creating, setCreating] = useState(null)

  const dq = useDebounce(query, 300)

  useEffect(() => {
    if (!dq.trim()) { setUsers([]); return }
    setLoading(true)
    api.get('/users/search', { params: { q: dq, limit: 20 } })
      .then(({ data }) => setUsers(data.users ?? []))
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [dq])

  async function startChat(user) {
    setCreating(user.id)
    try {
      const { data } = await api.post('/chats', { member_ids: [user.id] })
      onConversationCreated(data.conversation ?? data)
      onClose()
    } catch { toast.error('Failed to start conversation') }
    finally { setCreating(null) }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 modal-backdrop"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-sm overflow-hidden animate-fade-in">
        <div
          className="flex items-center justify-between px-4 py-3 border-b"
          style={{ borderColor: 'var(--border)' }}
        >
          <h3 className="font-semibold text-hi">New Chat</h3>
          <button onClick={onClose} className="text-lo hover:text-hi p-1"><X size={18} /></button>
        </div>

        <div className="px-3 py-2 border-b" style={{ borderColor: 'var(--border)' }}>
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
              placeholder="Search users..."
              className="input-base pl-9 py-1.5 text-sm"
              autoFocus
            />
          </div>
        </div>

        <div className="overflow-y-auto" style={{ maxHeight: '340px' }}>
          {loading ? (
            <div className="py-8 flex justify-center">
              <Loader2 size={20} className="animate-spin" style={{ color: 'var(--accent)' }} />
            </div>
          ) : users.length === 0 && query.trim() ? (
            <p className="text-center text-sm text-lo py-8">No users found.</p>
          ) : users.length === 0 ? (
            <p className="text-center text-sm text-lo py-8">Search for someone to chat with.</p>
          ) : users.map((u) => (
            <button
              key={u.id}
              onClick={() => startChat(u)}
              disabled={!!creating}
              className="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-elevated transition-colors disabled:opacity-60"
            >
              <Avatar src={u.avatar_url} name={u.full_name ?? u.username} size={40} />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold text-hi truncate">{u.full_name ?? u.username}</p>
                <p className="text-xs text-lo">@{u.username}</p>
              </div>
              {creating === u.id && (
                <Loader2 size={16} className="animate-spin shrink-0" style={{ color: 'var(--accent)' }} />
              )}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

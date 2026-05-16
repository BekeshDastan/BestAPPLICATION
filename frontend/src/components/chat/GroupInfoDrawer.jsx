import { useState } from 'react'
import { X, UserMinus, LogOut, Check } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../shared/Avatar'
import useAuthStore from '../../store/authStore'

export default function GroupInfoDrawer({ conv, onClose, onUpdate }) {
  const { user: me } = useAuthStore()
  const [name,    setName]    = useState(conv.name ?? '')
  const [editing, setEditing] = useState(false)
  const [saving,  setSaving]  = useState(false)

  const adminId = conv.admin_id ?? conv.creator_id
  const isAdmin = adminId === me?.id

  async function updateName() {
    if (!name.trim()) return
    setSaving(true)
    try {
      await api.put(`/chats/${conv.id}`, { name: name.trim() })
      toast.success('Group name updated')
      onUpdate({ ...conv, name: name.trim() })
      setEditing(false)
    } catch { toast.error('Failed to update') }
    finally { setSaving(false) }
  }

  async function removeParticipant(userId) {
    try {
      await api.delete(`/chats/${conv.id}/participants/${userId}`)
      toast.success('Participant removed')
      onUpdate({
        ...conv,
        participants: conv.participants?.filter((p) => p.id !== userId),
      })
    } catch { toast.error('Failed to remove participant') }
  }

  async function leaveGroup() {
    if (!window.confirm('Leave this group?')) return
    try {
      await api.delete(`/chats/${conv.id}/participants/me`)
      toast.success('You left the group')
      onClose('left')
    } catch { toast.error('Failed to leave group') }
  }

  return (
    <div
      className="fixed inset-y-0 right-0 z-40 flex flex-col animate-slide-in-right shadow-xl"
      style={{
        width: 300,
        background: 'var(--surface)',
        borderLeft: '1px solid var(--border)',
      }}
    >
      {/* Header */}
      <div
        className="flex items-center justify-between px-4 py-4 border-b shrink-0"
        style={{ borderColor: 'var(--border)' }}
      >
        <h3 className="font-semibold text-hi">Group Info</h3>
        <button onClick={() => onClose()} className="text-lo hover:text-hi p-1 transition-colors">
          <X size={18} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {/* Group name */}
        <div className="px-4 py-4 border-b" style={{ borderColor: 'var(--border)' }}>
          {editing ? (
            <div className="flex items-center gap-2">
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="input-base flex-1 text-sm"
                onKeyDown={(e) => e.key === 'Enter' && updateName()}
                autoFocus
              />
              <button
                onClick={updateName}
                disabled={saving}
                className="btn-primary px-3 py-1.5 text-xs"
              >
                <Check size={14} />
              </button>
              <button
                onClick={() => { setEditing(false); setName(conv.name ?? '') }}
                className="text-lo text-xs px-2 py-1.5"
              >
                ✕
              </button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <p className="font-semibold text-hi flex-1">{name || conv.name}</p>
              {isAdmin && (
                <button
                  onClick={() => setEditing(true)}
                  className="text-xs text-lo hover:text-hi transition-colors"
                >
                  Edit
                </button>
              )}
            </div>
          )}
          <p className="text-xs text-lo mt-1">
            {conv.participants?.length ?? 0} participants
          </p>
        </div>

        {/* Participants */}
        <div className="py-1">
          <p className="px-4 py-2 text-[11px] text-lo uppercase font-semibold tracking-wide">
            Members
          </p>
          {conv.participants?.map((p) => (
            <div
              key={p.id}
              className="flex items-center gap-3 px-4 py-2.5 hover:bg-elevated transition-colors"
            >
              <Avatar src={p.avatar_url} name={p.username} size={36} />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-semibold text-hi truncate">@{p.username}</p>
                {p.id === adminId && (
                  <p className="text-[11px]" style={{ color: 'var(--accent)' }}>Admin</p>
                )}
              </div>
              {isAdmin && p.id !== me?.id && (
                <button
                  onClick={() => removeParticipant(p.id)}
                  className="text-lo hover:text-danger p-1 transition-colors"
                  title="Remove"
                >
                  <UserMinus size={15} />
                </button>
              )}
            </div>
          ))}
        </div>
      </div>

      {/* Leave button */}
      <div className="p-4 border-t shrink-0" style={{ borderColor: 'var(--border)' }}>
        <button
          onClick={leaveGroup}
          className="w-full flex items-center justify-center gap-2 py-2 rounded-btn text-sm transition-colors"
          style={{ color: 'var(--danger)' }}
          onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.1)')}
          onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
        >
          <LogOut size={15} /> Leave Group
        </button>
      </div>
    </div>
  )
}

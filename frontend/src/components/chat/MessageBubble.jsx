import { useState } from 'react'
import { Reply, Edit2, Trash2, Pin } from 'lucide-react'
import Avatar from '../shared/Avatar'
import { formatRelativeTime } from '../../lib/utils'

export default function MessageBubble({ message, isOwn, onReply, onEdit, onDelete, onPin }) {
  const [hovered, setHovered] = useState(false)

  return (
    <div
      className={`flex gap-2 ${isOwn ? 'flex-row-reverse' : 'flex-row'} items-end mb-1.5`}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {!isOwn && (
        <Avatar src={message.sender?.avatar_url} name={message.sender?.username} size={28} />
      )}

      <div className={`max-w-[72%] flex flex-col ${isOwn ? 'items-end' : 'items-start'}`}>
        {/* Reply preview */}
        {message.reply_to && (
          <div
            className="mb-1 px-3 py-1.5 rounded-lg text-xs max-w-full border-l-2"
            style={{
              background:   'var(--surface-high)',
              borderColor:  'var(--accent)',
              color:        'var(--text-2)',
            }}
          >
            <span className="block truncate">↩ {message.reply_to.text ?? 'Media'}</span>
          </div>
        )}

        {/* Pinned label */}
        {message.is_pinned && (
          <div className="flex items-center gap-1 text-[10px] text-lo mb-0.5">
            <span>📌</span><span>Pinned</span>
          </div>
        )}

        {/* Bubble */}
        <div
          className="px-3 py-2 text-sm leading-relaxed"
          style={{
            background:          isOwn ? 'var(--accent)' : 'var(--surface-high)',
            color:               isOwn ? '#fff' : 'var(--text-1)',
            borderRadius:        isOwn ? '16px 16px 4px 16px' : '16px 16px 16px 4px',
            boxShadow:           message.is_pinned ? '0 0 0 1px var(--accent)' : 'none',
          }}
        >
          {message.media_url && (
            <img
              src={message.media_url}
              alt=""
              className="rounded-lg max-w-[220px] max-h-[220px] object-cover mb-1 block"
            />
          )}
          {message.text && <p className="whitespace-pre-wrap break-words">{message.text}</p>}
        </div>

        {/* Timestamp + read */}
        <div className="flex items-center gap-1 mt-0.5 px-0.5">
          <span className="text-[10px] text-lo">{formatRelativeTime(message.created_at)}</span>
          {isOwn && (
            <span
              className="text-[11px]"
              style={{ color: message.is_read ? 'var(--accent)' : 'var(--text-2)' }}
            >
              ✓✓
            </span>
          )}
        </div>
      </div>

      {/* Action bar on hover */}
      {hovered && (
        <div className={`flex items-center gap-0.5 shrink-0 self-center ${isOwn ? 'mr-1' : 'ml-1'}`}>
          <ActionBtn icon={<Reply size={12} />}  onClick={() => onReply(message)} title="Reply" />
          {isOwn && <ActionBtn icon={<Edit2 size={12} />}  onClick={() => onEdit(message)}  title="Edit" />}
          <ActionBtn icon={<Pin size={12} />}    onClick={() => onPin(message)}   title="Pin" />
          {isOwn && <ActionBtn icon={<Trash2 size={12} />} onClick={() => onDelete(message)} title="Delete" danger />}
        </div>
      )}
    </div>
  )
}

function ActionBtn({ icon, onClick, title, danger }) {
  return (
    <button
      onClick={onClick}
      title={title}
      className="w-6 h-6 rounded-full flex items-center justify-center transition-colors"
      style={{
        background: 'var(--surface-high)',
        color: danger ? 'var(--danger)' : 'var(--text-2)',
      }}
      onMouseEnter={(e) => (e.currentTarget.style.color = danger ? 'var(--danger)' : 'var(--text-1)')}
      onMouseLeave={(e) => (e.currentTarget.style.color = danger ? 'var(--danger)' : 'var(--text-2)')}
    >
      {icon}
    </button>
  )
}

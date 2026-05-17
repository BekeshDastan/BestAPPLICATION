import { useState, useEffect, useRef, useCallback } from 'react'
import { Send, Paperclip, Info, Search, X } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import Avatar from '../shared/Avatar'
import MessageBubble from './MessageBubble'
import GroupInfoDrawer from './GroupInfoDrawer'

const PAGE = 30

export default function ConversationView({ conv, ws, lastWsMsg, onConvUpdate, onLeave }) {
  const { user: me } = useAuthStore()

  const [messages,    setMessages]    = useState([])
  const [loading,     setLoading]     = useState(true)
  const [offset,      setOffset]      = useState(0)
  const [hasMore,     setHasMore]     = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)

  const [text,       setText]       = useState('')
  const [replyTo,    setReplyTo]    = useState(null)
  const [editingMsg, setEditingMsg] = useState(null)
  const [isTyping,   setIsTyping]   = useState(false)

  const [showGroupInfo, setShowGroupInfo] = useState(false)
  const [showSearch,    setShowSearch]    = useState(false)
  const [searchQuery,   setSearchQuery]   = useState('')

  const bottomRef      = useRef(null)
  const topRef         = useRef(null)
  const typingTimer    = useRef(null)
  const typingEmit     = useRef(null)
  const fileInputRef   = useRef(null)
  const lastWsMsgRef   = useRef(null)
  const firstLoad      = useRef(true)

  const otherUser   = conv.is_group ? null : conv.other_user
  const displayName = conv.is_group
    ? conv.name
    : otherUser?.full_name ?? otherUser?.username ?? 'Unknown'

  /* ── Load messages ───────────────────────────────────────────── */
  useEffect(() => {
    setLoading(true)
    setMessages([])
    setOffset(0)
    setHasMore(true)
    firstLoad.current = true

    api.get(`/chats/${conv.id}/messages`, { params: { limit: PAGE, offset: 0 } })
      .then(({ data }) => {
        const msgs = (data.messages ?? []).reverse()
        setMessages(msgs)
        setOffset(msgs.length)
        setHasMore(msgs.length === PAGE)
      })
      .catch(() => {})
      .finally(() => {
        setLoading(false)
        api.post(`/chats/${conv.id}/read`).catch(() => {})
      })
  }, [conv.id])

  /* ── Scroll to bottom on initial load ───────────────────────── */
  useEffect(() => {
    if (!loading && firstLoad.current) {
      firstLoad.current = false
      setTimeout(() => bottomRef.current?.scrollIntoView(), 50)
    }
  }, [loading])

  /* ── Incoming WS messages ────────────────────────────────────── */
  useEffect(() => {
    if (!lastWsMsg || lastWsMsg === lastWsMsgRef.current) return
    lastWsMsgRef.current = lastWsMsg
    const msg = lastWsMsg

    if (msg.type === 'message' && msg.conversation_id === conv.id) {
      const payload = msg.payload ?? msg
      setMessages((prev) => {
        if (prev.some((m) => m.id === payload.id)) return prev
        return [...prev, payload]
      })
      setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50)
      setIsTyping(false)
    }
    if (msg.type === 'typing' && msg.conversation_id === conv.id && msg.user_id !== me?.id) {
      setIsTyping(true)
      clearTimeout(typingTimer.current)
      typingTimer.current = setTimeout(() => setIsTyping(false), 2000)
    }
    if (msg.type === 'read' && msg.conversation_id === conv.id) {
      setMessages((prev) => prev.map((m) => m.sender_id === me?.id ? { ...m, is_read: true } : m))
    }
    if (msg.type === 'message_deleted' && msg.conversation_id === conv.id) {
      setMessages((prev) => prev.filter((m) => m.id !== msg.message_id))
    }
    if (msg.type === 'message_edited' && msg.conversation_id === conv.id) {
      setMessages((prev) => prev.map((m) => m.id === msg.message_id ? { ...m, text: msg.text } : m))
    }
  }, [lastWsMsg, conv.id, me?.id])

  /* ── Load older on scroll-to-top ────────────────────────────── */
  useEffect(() => {
    const el = topRef.current
    if (!el) return
    const observer = new IntersectionObserver(async ([entry]) => {
      if (entry.isIntersecting && hasMore && !loadingMore && !loading) {
        setLoadingMore(true)
        try {
          const { data } = await api.get(`/chats/${conv.id}/messages`, {
            params: { limit: PAGE, offset },
          })
          const older = (data.messages ?? []).reverse()
          setMessages((prev) => [...older, ...prev])
          setOffset((o) => o + older.length)
          setHasMore(older.length === PAGE)
        } catch {}
        finally { setLoadingMore(false) }
      }
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [conv.id, offset, hasMore, loadingMore, loading])

  /* ── Typing emit ─────────────────────────────────────────────── */
  function handleTextChange(val) {
    setText(val)
    if (!ws?.send) return
    clearTimeout(typingEmit.current)
    typingEmit.current = setTimeout(() => {
      ws.send({ type: 'typing', conversation_id: conv.id })
    }, 600)
  }

  /* ── Send / edit message ─────────────────────────────────────── */
  async function sendMessage() {
    const trimmed = text.trim()
    if (!trimmed) return

    if (editingMsg) {
      try {
        await api.put(`/messages/${editingMsg.id}`, { text: trimmed })
        setMessages((prev) => prev.map((m) => m.id === editingMsg.id ? { ...m, text: trimmed } : m))
        setText('')
        setEditingMsg(null)
      } catch { toast.error('Failed to edit message') }
      return
    }

    const tempId = `temp-${Date.now()}`
    const optimistic = {
      id: tempId, conversation_id: conv.id,
      sender_id: me?.id, sender: me,
      text: trimmed, reply_to: replyTo,
      created_at: new Date().toISOString(), is_read: false,
    }
    setMessages((prev) => [...prev, optimistic])
    setText('')
    setReplyTo(null)
    setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50)

    try {
      const { data } = await api.post('/messages', {
        conversation_id: conv.id,
        text: trimmed,
        reply_to_id: replyTo?.id ?? undefined,
      })
      setMessages((prev) => prev.map((m) => m.id === tempId ? data : m))
    } catch {
      setMessages((prev) => prev.filter((m) => m.id !== tempId))
      toast.error('Failed to send message')
    }
  }

  async function deleteMessage(msg) {
    if (!window.confirm('Delete this message?')) return
    try {
      await api.delete(`/messages/${msg.id}`)
      setMessages((prev) => prev.filter((m) => m.id !== msg.id))
    } catch { toast.error('Failed to delete') }
  }

  async function pinMessage(msg) {
    try {
      await api.post(`/messages/${msg.id}/pin`)
      setMessages((prev) => prev.map((m) => m.id === msg.id ? { ...m, is_pinned: !m.is_pinned } : m))
    } catch { toast.error('Failed to pin') }
  }

  async function uploadMedia(file) {
    try {
      const { data: urlData } = await api.get('/media/upload-url', { params: { type: 'chat' } })
      await fetch(urlData.upload_url, {
        method: 'PUT', body: file, headers: { 'Content-Type': file.type },
      })
      const { data } = await api.post('/messages', {
        conversation_id: conv.id, media_url: urlData.media_url,
      })
      setMessages((prev) => [...prev, data])
      setTimeout(() => bottomRef.current?.scrollIntoView({ behavior: 'smooth' }), 50)
    } catch { toast.error('Failed to send file') }
  }

  const displayed = searchQuery.trim()
    ? messages.filter((m) => m.text?.toLowerCase().includes(searchQuery.toLowerCase()))
    : messages

  return (
    <div className="flex flex-col h-full relative overflow-hidden">
      {/* Header */}
      <div
        className="flex items-center gap-3 px-4 py-3 shrink-0 border-b"
        style={{ borderColor: 'var(--border)', background: 'var(--surface)' }}
      >
        <Avatar
          src={conv.is_group ? null : otherUser?.avatar_url}
          name={displayName}
          size={40}
        />
        <div className="flex-1 min-w-0">
          <p className="font-semibold text-hi truncate">{displayName}</p>
          <p className="text-xs text-lo flex items-center gap-1">
            {conv.is_group
              ? `${conv.participants?.length ?? 0} members`
              : otherUser?.is_online
                ? <><span className="w-1.5 h-1.5 rounded-full bg-green-500 inline-block" />online</>
                : `@${otherUser?.username}`}
          </p>
        </div>
        <button
          onClick={() => setShowSearch((v) => !v)}
          className="p-2 rounded-btn text-lo hover:text-hi transition-colors"
          title="Search"
        >
          <Search size={17} />
        </button>
        <button
          onClick={() => setShowGroupInfo((v) => !v)}
          className="p-2 rounded-btn text-lo hover:text-hi transition-colors"
          title="Info"
        >
          <Info size={17} />
        </button>
      </div>

      {/* Search bar */}
      {showSearch && (
        <div
          className="flex items-center gap-2 px-4 py-2 border-b"
          style={{ borderColor: 'var(--border)', background: 'var(--surface-high)' }}
        >
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search messages..."
            className="input-base flex-1 text-sm py-1.5"
            autoFocus
          />
          <button
            onClick={() => { setShowSearch(false); setSearchQuery('') }}
            className="text-lo hover:text-hi transition-colors"
          >
            <X size={16} />
          </button>
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-4">
        <div ref={topRef} className="h-1" />

        {loadingMore && (
          <p className="text-center text-xs text-lo py-2">Loading older messages...</p>
        )}

        {loading ? (
          <div className="space-y-3">
            {Array.from({ length: 8 }).map((_, i) => (
              <div
                key={i}
                className={`flex gap-2 ${i % 3 === 0 ? 'flex-row-reverse' : ''}`}
              >
                <div className="skeleton w-7 h-7 rounded-full shrink-0" />
                <div className={`skeleton h-9 rounded-2xl ${i % 3 === 0 ? 'w-44' : 'w-52'}`} />
              </div>
            ))}
          </div>
        ) : displayed.length === 0 ? (
          <p className="text-center text-lo text-sm py-8">
            {searchQuery ? 'No messages match your search.' : 'No messages yet. Say hi!'}
          </p>
        ) : displayed.map((m) => (
          <MessageBubble
            key={m.id}
            message={m}
            isOwn={m.sender_id === me?.id}
            onReply={setReplyTo}
            onEdit={(msg) => { setEditingMsg(msg); setText(msg.text ?? '') }}
            onDelete={deleteMessage}
            onPin={pinMessage}
          />
        ))}

        {/* Typing indicator */}
        {isTyping && (
          <div className="flex items-end gap-2 mb-2">
            <Avatar src={otherUser?.avatar_url} name={otherUser?.username} size={24} />
            <div
              className="flex items-center gap-1 px-3 py-2 rounded-2xl"
              style={{ background: 'var(--surface-high)', borderRadius: '16px 16px 16px 4px' }}
            >
              {[0, 150, 300].map((delay) => (
                <span
                  key={delay}
                  className="w-1.5 h-1.5 rounded-full animate-bounce"
                  style={{ background: 'var(--text-2)', animationDelay: `${delay}ms` }}
                />
              ))}
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      {/* Reply bar */}
      {replyTo && (
        <div
          className="flex items-center gap-2 px-4 py-2 border-t shrink-0"
          style={{ borderColor: 'var(--border)', background: 'var(--surface-high)' }}
        >
          <div className="flex-1 text-xs truncate">
            <span style={{ color: 'var(--accent)' }}>↩ Replying to: </span>
            <span className="text-lo">{replyTo.text ?? 'Media'}</span>
          </div>
          <button onClick={() => setReplyTo(null)} className="text-lo hover:text-hi transition-colors p-1">
            <X size={14} />
          </button>
        </div>
      )}

      {/* Edit bar */}
      {editingMsg && (
        <div
          className="flex items-center gap-2 px-4 py-2 border-t shrink-0"
          style={{ borderColor: 'var(--border)', background: 'var(--surface-high)' }}
        >
          <p className="flex-1 text-xs" style={{ color: 'var(--accent)' }}>✏ Editing message</p>
          <button
            onClick={() => { setEditingMsg(null); setText('') }}
            className="text-lo hover:text-hi transition-colors p-1"
          >
            <X size={14} />
          </button>
        </div>
      )}

      {/* Input bar */}
      <div
        className="flex items-end gap-2 px-4 py-3 shrink-0 border-t"
        style={{ borderColor: 'var(--border)', background: 'var(--surface)' }}
      >
        <button
          onClick={() => fileInputRef.current?.click()}
          className="p-2 rounded-btn text-lo hover:text-hi transition-colors shrink-0 mb-0.5"
          title="Attach file"
        >
          <Paperclip size={18} />
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*,video/*"
          className="hidden"
          onChange={(e) => { const f = e.target.files[0]; if (f) uploadMedia(f) }}
        />
        <textarea
          value={text}
          onChange={(e) => handleTextChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault()
              sendMessage()
            }
          }}
          placeholder="Type a message..."
          rows={1}
          className="input-base flex-1 text-sm resize-none py-2 min-h-[38px] max-h-32 overflow-y-auto"
          style={{ lineHeight: 1.5 }}
        />
        <button
          onClick={sendMessage}
          disabled={!text.trim()}
          className="p-2 rounded-btn transition-colors shrink-0 disabled:opacity-40 mb-0.5"
          style={{ background: 'var(--accent)', color: '#fff' }}
        >
          <Send size={18} />
        </button>
      </div>

      {/* Group info drawer */}
      {showGroupInfo && (
        <GroupInfoDrawer
          conv={conv}
          onClose={(reason) => {
            setShowGroupInfo(false)
            if (reason === 'left') onLeave?.()
          }}
          onUpdate={onConvUpdate}
        />
      )}
    </div>
  )
}

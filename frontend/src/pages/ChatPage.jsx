import { useEffect, useState, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import { useAuth } from '../AuthContext'
import { chatApi } from '../api'

function Avatar({ name, size = 40 }) {
  return (
    <div style={{ width: size, height: size, borderRadius: '50%', background: '#333', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: size * 0.38, flexShrink: 0 }}>
      {name?.[0]?.toUpperCase() || '?'}
    </div>
  )
}

function NewChatModal({ onClose, onCreated }) {
  const [uid, setUid] = useState('')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async e => {
    e.preventDefault()
    if (!uid.trim()) return
    setLoading(true); setErr('')
    try {
      const { data } = await chatApi.create([uid.trim()])
      onCreated(data.conversation)
    } catch (e) { setErr(e.response?.data?.error || 'Failed to create chat') }
    finally { setLoading(false) }
  }

  return (
    <div style={s.overlay} onClick={onClose}>
      <div style={s.modal} onClick={e => e.stopPropagation()}>
        <div style={s.mHead}>
          <span style={{ fontWeight: 600 }}>New Message</span>
          <button onClick={onClose} style={s.closeBtn}>✕</button>
        </div>
        {err && <div style={s.err}>{err}</div>}
        <form onSubmit={submit} style={s.mForm}>
          <input placeholder="Recipient User ID" value={uid} onChange={e => setUid(e.target.value)} style={s.input} autoFocus />
          <button type="submit" disabled={loading || !uid.trim()} style={s.sendBtn}>{loading ? '…' : 'Start Chat'}</button>
        </form>
      </div>
    </div>
  )
}

function ConvList({ convs, active, onSelect, onNew }) {
  return (
    <div style={s.sidebar}>
      <div style={s.sHead}>
        <span style={{ fontWeight: 600, fontSize: 16 }}>Messages</span>
        <button onClick={onNew} style={s.newBtn} title="New message">✎</button>
      </div>
      <div style={{ overflowY: 'auto', flex: 1 }}>
        {convs.length === 0 && <div style={s.empty}>No conversations yet.</div>}
        {convs.map(c => (
          <div key={c.id} onClick={() => onSelect(c)} style={{ ...s.convItem, background: active?.id === c.id ? '#f5f5f5' : 'transparent' }}>
            <Avatar name={c.name || c.id} size={44} />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={s.convName}>{c.name || c.id?.slice(0, 16)}</div>
              <div style={s.convSub}>{c.type === 'group' ? 'Group chat' : 'Direct message'}</div>
            </div>
            {c.unread_count > 0 && <span style={s.badge}>{c.unread_count}</span>}
          </div>
        ))}
      </div>
    </div>
  )
}

function ConvInfoBtn({ conv, onDeleted }) {
  const [open, setOpen] = useState(false)
  const [detail, setDetail] = useState(null)

  const loadDetail = async () => {
    try {
      // GET /chats/:id — fetch full conversation info
      const { data } = await chatApi.get(conv.id)
      setDetail(data.conversation)
    } catch {}
    setOpen(true)
  }

  const deleteConv = async () => {
    if (!confirm('Delete this conversation?')) return
    try {
      // DELETE /chats/:id
      await chatApi.delete(conv.id)
      setOpen(false)
      onDeleted(conv.id)
    } catch {}
  }

  return (
    <>
      <button onClick={loadDetail} style={s.infoBtn} title="Conversation info">ⓘ</button>
      {open && (
        <div style={s.overlay} onClick={() => setOpen(false)}>
          <div style={s.modal} onClick={e => e.stopPropagation()}>
            <div style={s.mHead}>
              <span style={{ fontWeight: 600 }}>Conversation Info</span>
              <button onClick={() => setOpen(false)} style={s.closeBtn}>✕</button>
            </div>
            <div style={{ padding: '16px' }}>
              {detail ? (
                <>
                  <div style={s.infoRow}><span style={s.infoLabel}>ID</span><span style={{ fontSize: 12, fontFamily: 'monospace' }}>{detail.id}</span></div>
                  <div style={s.infoRow}><span style={s.infoLabel}>Type</span><span>{detail.type}</span></div>
                  <div style={s.infoRow}><span style={s.infoLabel}>Created by</span><span style={{ fontSize: 12 }}>{detail.created_by?.slice(0, 16)}…</span></div>
                  {detail.last_message_at && <div style={s.infoRow}><span style={s.infoLabel}>Last message</span><span>{new Date(detail.last_message_at * 1000).toLocaleString()}</span></div>}
                </>
              ) : <div style={{ color: '#8e8e8e', fontSize: 13 }}>Loading…</div>}
              <button onClick={deleteConv} style={s.deleteConvBtn}>🗑 Delete conversation</button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

function ChatWindow({ conv, myId, onDeleted }) {
  const [msgs, setMsgs] = useState([])
  const [text, setText] = useState('')
  const [loading, setLoading] = useState(false)
  const bottom = useRef(null)
  const pollRef = useRef(null)

  const loadMsgs = async () => {
    if (!conv) return
    try {
      const { data } = await chatApi.listMessages(conv.id)
      setMsgs(data.messages || [])
    } catch {}
  }

  useEffect(() => {
    if (!conv) return
    setLoading(true)
    loadMsgs().finally(() => setLoading(false))
    pollRef.current = setInterval(loadMsgs, 5000)
    return () => clearInterval(pollRef.current)
  }, [conv?.id])

  useEffect(() => { bottom.current?.scrollIntoView({ behavior: 'smooth' }) }, [msgs])

  const send = async e => {
    e.preventDefault()
    if (!text.trim()) return
    try {
      const { data } = await chatApi.sendMessage(conv.id, text.trim())
      setMsgs(m => [...m, data.message])
      setText('')
    } catch {}
  }

  const del = async (msgId) => {
    try {
      await chatApi.deleteMessage(conv.id, msgId)
      setMsgs(m => m.filter(x => x.id !== msgId))
    } catch {}
  }

  if (!conv) {
    return (
      <div style={s.empty2}>
        <div style={{ fontSize: 48, marginBottom: 12 }}>✉</div>
        <p style={{ fontWeight: 600 }}>Your Messages</p>
        <p style={{ color: '#8e8e8e', fontSize: 14 }}>Send private messages to a friend.</p>
      </div>
    )
  }

  return (
    <div style={s.chatWin}>
      <div style={s.chatHead}>
        <Avatar name={conv.name || conv.id} size={32} />
        <span style={{ marginLeft: 10, fontWeight: 600, flex: 1 }}>{conv.name || conv.id?.slice(0, 16)}</span>
        <ConvInfoBtn conv={conv} onDeleted={onDeleted} />
      </div>
      <div style={s.msgList}>
        {loading && <div style={s.hint}>Loading…</div>}
        {msgs.map(m => (
          <div key={m.id} style={{ display: 'flex', flexDirection: 'column', alignItems: m.sender_id === myId ? 'flex-end' : 'flex-start', gap: 2 }}>
            <div style={{ ...s.bubble, background: m.sender_id === myId ? '#0095f6' : '#efefef', color: m.sender_id === myId ? '#fff' : '#262626' }}>
              {m.text}
              {m.media_url && <img src={m.media_url} alt="" style={{ maxWidth: 200, borderRadius: 4, display: 'block', marginTop: 4 }} />}
            </div>
            {m.sender_id === myId && (
              <button onClick={() => del(m.id)} style={s.delMsg}>Delete</button>
            )}
          </div>
        ))}
        <div ref={bottom} />
      </div>
      <form onSubmit={send} style={s.inputRow}>
        <input value={text} onChange={e => setText(e.target.value)} placeholder="Message…" style={s.msgInput} />
        <button type="submit" disabled={!text.trim()} style={s.sendMsgBtn}>Send</button>
      </form>
    </div>
  )
}

export default function ChatPage() {
  const { user } = useAuth()
  const location = useLocation()
  const [convs, setConvs] = useState([])
  const [active, setActive] = useState(null)
  const [showNew, setShowNew] = useState(false)

  useEffect(() => {
    chatApi.list().then(r => {
      const list = r.data.conversations || []
      setConvs(list)
      // If navigated from profile "Message" button, auto-select or add conversation
      const nav = location.state?.conv
      if (nav) {
        const exists = list.find(c => c.id === nav.id)
        if (!exists) setConvs(c => [nav, ...c])
        setActive(nav)
      }
    }).catch(() => {})
  }, [])

  const handleNew = conv => { setConvs(c => [conv, ...c]); setActive(conv); setShowNew(false) }
  const handleDelete = id => { setConvs(c => c.filter(x => x.id !== id)); setActive(a => a?.id === id ? null : a) }

  return (
    <div style={s.page}>
      <ConvList convs={convs} active={active} onSelect={setActive} onNew={() => setShowNew(true)} />
      <ChatWindow conv={active} myId={user?.id} onDeleted={handleDelete} />
      {showNew && <NewChatModal onClose={() => setShowNew(false)} onCreated={handleNew} />}
    </div>
  )
}

const s = {
  page: { display: 'flex', height: '100vh', background: '#fafafa' },
  sidebar: { width: 350, borderRight: '1px solid #dbdbdb', background: '#fff', display: 'flex', flexDirection: 'column' },
  sHead: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '16px 20px', borderBottom: '1px solid #dbdbdb' },
  newBtn: { background: 'none', border: 'none', fontSize: 20, cursor: 'pointer', color: '#262626' },
  convItem: { display: 'flex', alignItems: 'center', gap: 12, padding: '12px 20px', cursor: 'pointer', borderBottom: '1px solid #fafafa' },
  convName: { fontWeight: 600, fontSize: 14, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' },
  convSub: { fontSize: 12, color: '#8e8e8e', marginTop: 2 },
  badge: { background: '#e53935', color: '#fff', borderRadius: 10, padding: '2px 7px', fontSize: 11, fontWeight: 700 },
  chatWin: { flex: 1, display: 'flex', flexDirection: 'column' },
  chatHead: { display: 'flex', alignItems: 'center', padding: '12px 20px', borderBottom: '1px solid #dbdbdb', background: '#fff' },
  msgList: { flex: 1, padding: '16px 20px', overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 },
  bubble: { maxWidth: '70%', padding: '10px 14px', borderRadius: 18, fontSize: 14, wordBreak: 'break-word' },
  delMsg: { background: 'none', border: 'none', color: '#bbb', fontSize: 11, cursor: 'pointer', padding: '2px 4px' },
  inputRow: { display: 'flex', gap: 8, padding: '12px 20px', background: '#fff', borderTop: '1px solid #dbdbdb' },
  msgInput: { flex: 1, padding: '10px 14px', border: '1px solid #dbdbdb', borderRadius: 22, fontSize: 14, outline: 'none' },
  sendMsgBtn: { padding: '10px 18px', background: '#0095f6', color: '#fff', border: 'none', borderRadius: 22, fontWeight: 700, fontSize: 14 },
  empty: { padding: '20px', textAlign: 'center', color: '#8e8e8e', fontSize: 13 },
  empty2: { flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: '#262626' },
  hint: { textAlign: 'center', color: '#8e8e8e', fontSize: 13 },
  overlay: { position: 'fixed', inset: 0, background: 'rgba(0,0,0,.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 200 },
  modal: { background: '#fff', borderRadius: 12, width: 400, maxWidth: '90vw' },
  mHead: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '12px 16px', borderBottom: '1px solid #dbdbdb' },
  mForm: { display: 'flex', gap: 8, padding: '16px' },
  input: { flex: 1, padding: '9px 12px', border: '1px solid #dbdbdb', borderRadius: 6, fontSize: 14 },
  sendBtn: { padding: '9px 16px', background: '#0095f6', color: '#fff', border: 'none', borderRadius: 6, fontWeight: 700 },
  closeBtn: { background: 'none', border: 'none', fontSize: 18, color: '#8e8e8e', cursor: 'pointer' },
  err: { color: '#e53935', fontSize: 13, padding: '8px 16px', background: '#fff3f3' },
  infoBtn: { background: 'none', border: 'none', fontSize: 20, color: '#8e8e8e', cursor: 'pointer', padding: '4px 8px' },
  infoRow: { display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #fafafa', fontSize: 13 },
  infoLabel: { color: '#8e8e8e', fontWeight: 600, minWidth: 90 },
  deleteConvBtn: { marginTop: 16, width: '100%', padding: '10px', background: '#fff', border: '1px solid #e53935', color: '#e53935', borderRadius: 6, fontWeight: 600, fontSize: 13, cursor: 'pointer' },
}

import { useState, useEffect, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { MessageSquare, Plus } from 'lucide-react'
import api from '../../lib/api'
import useWsStore from '../../store/wsStore'
import ConversationList from '../../components/chat/ConversationList'
import ConversationView from '../../components/chat/ConversationView'
import NewChatModal from '../../components/chat/NewChatModal'

export default function ChatPage() {
  const { convId }     = useParams()
  const [selectedConv, setSelectedConv] = useState(null)
  const [showNewChat,  setShowNewChat]  = useState(false)
  const [listKey,      setListKey]      = useState(0)
  const [lastWsMsg,    setLastWsMsg]    = useState(null)

  const send       = useWsStore((s) => s.send)
  const connected  = useWsStore((s) => s.connected)
  const addHandler = useWsStore((s) => s.addHandler)

  // Subscribe to WS messages via global store
  useEffect(() => {
    const unsub = addHandler((msg) => setLastWsMsg(msg))
    return unsub
  }, [addHandler])

  // If navigated to /chat/:convId, auto-select that conversation
  useEffect(() => {
    if (!convId) return
    api.get(`/chats/${convId}`)
      .then(({ data }) => setSelectedConv(data))
      .catch(() => {})
  }, [convId])

  const ws = { send, connected }

  function handleConvCreated(newConv) {
    setSelectedConv(newConv)
    setListKey((k) => k + 1)
  }

  return (
    <div className="flex overflow-hidden" style={{ height: '100dvh' }}>
      {/* Left pane */}
      <div
        className="w-80 shrink-0 flex flex-col border-r"
        style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}
      >
        <ConversationList
          key={listKey}
          selectedId={selectedConv?.id}
          onSelect={setSelectedConv}
          onNewChat={() => setShowNewChat(true)}
        />
      </div>

      {/* Right pane */}
      <div className="flex-1 flex flex-col min-w-0" style={{ background: 'var(--bg)' }}>
        {selectedConv ? (
          <ConversationView
            key={selectedConv.id}
            conv={selectedConv}
            ws={ws}
            lastWsMsg={lastWsMsg}
            onConvUpdate={(updated) => setSelectedConv(updated)}
            onLeave={() => { setSelectedConv(null); setListKey((k) => k + 1) }}
          />
        ) : (
          <EmptyState onNewChat={() => setShowNewChat(true)} />
        )}
      </div>

      {showNewChat && (
        <NewChatModal
          onClose={() => setShowNewChat(false)}
          onConversationCreated={handleConvCreated}
        />
      )}
    </div>
  )
}

function EmptyState({ onNewChat }) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center gap-5 text-center px-8">
      <div
        className="w-20 h-20 rounded-full flex items-center justify-center"
        style={{ background: 'var(--surface-high)' }}
      >
        <MessageSquare size={36} style={{ color: 'var(--accent)' }} />
      </div>
      <div>
        <h2 className="text-xl font-bold text-hi mb-2">Your Messages</h2>
        <p className="text-lo text-sm max-w-xs">
          Select a conversation from the list or start a new one.
        </p>
      </div>
      <button
        onClick={onNewChat}
        className="btn-primary px-6 py-2.5 flex items-center gap-2"
      >
        <Plus size={16} /> New Chat
      </button>
    </div>
  )
}

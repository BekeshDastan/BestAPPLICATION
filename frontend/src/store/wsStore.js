import { create } from 'zustand'
import { getAccessToken } from '../lib/api'

const BACKOFF_BASE = 1000
const BACKOFF_MAX  = 30000

let _socket    = null
let _attempts  = 0
let _reconnectTimer = null

const useWsStore = create((set, get) => ({
  connected: false,
  _handlers: new Set(),

  addHandler(fn) {
    get()._handlers.add(fn)
    return () => get()._handlers.delete(fn)
  },

  send(data) {
    if (_socket?.readyState === WebSocket.OPEN) {
      _socket.send(typeof data === 'string' ? data : JSON.stringify(data))
    }
  },

  connect() {
    if (_socket && (
      _socket.readyState === WebSocket.OPEN ||
      _socket.readyState === WebSocket.CONNECTING
    )) return

    const token = getAccessToken()
    if (!token) return

    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const url   = `${proto}://${window.location.host}/api/v1/ws?token=${token}`
    _socket     = new WebSocket(url)

    _socket.onopen = () => {
      _attempts = 0
      set({ connected: true })
    }

    _socket.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        get()._handlers.forEach((fn) => fn(msg))
      } catch {}
    }

    _socket.onclose = () => {
      set({ connected: false })
      _socket = null
      // Only reconnect if we still have a token (user still logged in)
      if (!getAccessToken()) return
      const delay = Math.min(BACKOFF_BASE * 2 ** _attempts, BACKOFF_MAX)
      _attempts++
      _reconnectTimer = setTimeout(() => {
        if (getAccessToken()) get().connect()
      }, delay)
    }

    _socket.onerror = () => {
      _socket?.close()
    }
  },

  disconnect() {
    clearTimeout(_reconnectTimer)
    _reconnectTimer = null
    _attempts = 0
    _socket?.close()
    _socket = null
    set({ connected: false })
  },
}))

export default useWsStore

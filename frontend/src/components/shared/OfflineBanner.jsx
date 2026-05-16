import { useState, useEffect } from 'react'
import useWsStore from '../../store/wsStore'
import useAuthStore from '../../store/authStore'

export default function OfflineBanner() {
  const [online, setOnline] = useState(navigator.onLine)
  const connected      = useWsStore((s) => s.connected)
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)

  useEffect(() => {
    const markOnline  = () => setOnline(true)
    const markOffline = () => setOnline(false)
    window.addEventListener('online',  markOnline)
    window.addEventListener('offline', markOffline)
    return () => {
      window.removeEventListener('online',  markOnline)
      window.removeEventListener('offline', markOffline)
    }
  }, [])

  // Only show to authenticated users; hide when everything is fine
  if (!isAuthenticated || (online && connected)) return null

  const offline = !online

  return (
    <div
      className="fixed bottom-5 left-1/2 -translate-x-1/2 z-[200] flex items-center gap-2.5 px-5 py-2.5 rounded-full text-sm font-medium shadow-xl animate-fade-in pointer-events-none"
      style={{ background: offline ? 'rgba(239,68,68,0.95)' : 'rgba(245,158,11,0.95)', color: '#fff' }}
    >
      <span className="w-2 h-2 rounded-full bg-white opacity-80 shrink-0" />
      {offline ? "You're offline. Reconnect to continue." : 'Reconnecting…'}
    </div>
  )
}

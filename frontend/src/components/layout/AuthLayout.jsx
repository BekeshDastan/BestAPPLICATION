import { Outlet } from 'react-router-dom'
import { Aperture } from 'lucide-react'

export default function AuthLayout() {
  return (
    <div
      className="min-h-dvh flex flex-col items-center justify-center px-4 py-12"
      style={{ background: 'var(--bg)' }}
    >
      {/* Logo */}
      <div className="flex items-center gap-2 mb-8 select-none">
        <Aperture size={28} style={{ color: 'var(--accent)' }} />
        <span
          className="text-xl font-bold tracking-tight"
          style={{ color: 'var(--text-1)' }}
        >
          Social
        </span>
      </div>

      {/* Card */}
      <div className="w-full max-w-[400px]">
        <Outlet />
      </div>

      {/* Footer */}
      <p className="mt-8 text-xs" style={{ color: 'var(--text-2)' }}>
        © {new Date().getFullYear()} Social. All rights reserved.
      </p>
    </div>
  )
}

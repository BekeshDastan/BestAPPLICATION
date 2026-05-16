import { useEffect, useState } from 'react'
import { NavLink, Outlet, Link, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard, Users, Image, BookOpen,
  Flag, Server, ArrowLeft, Shield,
} from 'lucide-react'
import api from '../../lib/api'
import useAuthStore from '../../store/authStore'
import Avatar from '../../components/shared/Avatar'

export const ADMIN_ACCENT = '#10B981'

const NAV = [
  { to: '/admin',          icon: LayoutDashboard, label: 'Dashboard', end: true },
  { to: '/admin/users',    icon: Users,            label: 'Users' },
  { to: '/admin/posts',    icon: Image,            label: 'Posts' },
  { to: '/admin/stories',  icon: BookOpen,         label: 'Stories' },
  { to: '/admin/reports',  icon: Flag,             label: 'Reports' },
  { to: '/admin/system',   icon: Server,           label: 'System' },
]

export default function AdminLayout() {
  const [checking, setChecking] = useState(true)
  const [allowed,  setAllowed]  = useState(false)
  const { user } = useAuthStore()

  useEffect(() => {
    api.get('/admin/me')
      .then(({ data }) => setAllowed(data.is_admin === true))
      .catch(() => setAllowed(false))
      .finally(() => setChecking(false))
  }, [])

  if (checking) {
    return (
      <div className="fixed inset-0 flex items-center justify-center" style={{ background: 'var(--bg)' }}>
        <div className="flex items-center gap-3">
          <div className="skeleton w-5 h-5 rounded-full" />
          <div className="skeleton w-32 h-4 rounded" />
        </div>
      </div>
    )
  }

  if (!allowed) return <AccessDenied />

  return (
    <div className="flex min-h-dvh" style={{ background: '#0A0A0A' }}>
      {/* Sidebar */}
      <aside
        className="w-60 shrink-0 flex flex-col sticky top-0 h-dvh"
        style={{ background: '#111111', borderRight: '1px solid #1F1F1F' }}
      >
        {/* Logo */}
        <div className="flex items-center gap-2.5 px-5 h-16 shrink-0 select-none">
          <Shield size={20} style={{ color: ADMIN_ACCENT }} />
          <span className="text-sm font-bold text-white tracking-tight">Social Admin</span>
        </div>

        <div className="mx-4 mb-3 h-px" style={{ background: '#1F1F1F' }} />

        {/* Nav */}
        <nav className="flex-1 flex flex-col gap-0.5 px-2 overflow-y-auto">
          {NAV.map(({ to, icon: Icon, label, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className="relative flex items-center gap-3 px-4 py-2.5 rounded-btn text-sm font-medium transition-all duration-150"
              style={({ isActive }) => ({
                color:      isActive ? ADMIN_ACCENT : '#71717A',
                background: isActive ? `${ADMIN_ACCENT}18` : 'transparent',
              })}
              onMouseEnter={(e) => { if (!e.currentTarget.dataset.active) e.currentTarget.style.background = 'rgba(255,255,255,0.04)' }}
              onMouseLeave={(e) => { if (!e.currentTarget.dataset.active) e.currentTarget.style.background = 'transparent' }}
            >
              {({ isActive }) => (
                <>
                  {isActive && (
                    <span
                      className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 rounded-r-full"
                      style={{ background: ADMIN_ACCENT }}
                    />
                  )}
                  <Icon size={17} />
                  {label}
                </>
              )}
            </NavLink>
          ))}
        </nav>

        <div className="mx-4 mt-3 h-px" style={{ background: '#1F1F1F' }} />

        {/* Bottom */}
        <div className="p-3 shrink-0 space-y-1">
          <div
            className="flex items-center gap-2.5 px-3 py-2 rounded-btn"
            style={{ background: '#1A1A1A' }}
          >
            <Avatar src={user?.avatar_url} name={user?.full_name ?? user?.username} size={28} />
            <div className="flex-1 min-w-0">
              <p className="text-xs font-semibold text-white truncate">{user?.username}</p>
              <p className="text-[10px]" style={{ color: ADMIN_ACCENT }}>Administrator</p>
            </div>
          </div>
          <Link
            to="/"
            className="flex items-center gap-2 px-3 py-2 rounded-btn text-xs transition-colors"
            style={{ color: '#52525B' }}
            onMouseEnter={(e) => (e.currentTarget.style.color = '#A1A1AA')}
            onMouseLeave={(e) => (e.currentTarget.style.color = '#52525B')}
          >
            <ArrowLeft size={13} /> Back to App
          </Link>
        </div>
      </aside>

      {/* Main */}
      <main className="flex-1 min-w-0 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  )
}

function AccessDenied() {
  const navigate = useNavigate()
  return (
    <div
      className="fixed inset-0 flex flex-col items-center justify-center gap-5 text-center px-4"
      style={{ background: 'var(--bg)' }}
    >
      <div
        className="w-24 h-24 rounded-full flex items-center justify-center"
        style={{ background: 'rgba(239,68,68,0.1)' }}
      >
        <Shield size={44} style={{ color: 'var(--danger)' }} />
      </div>
      <div>
        <h1 className="text-2xl font-bold text-hi mb-2">Access Denied</h1>
        <p className="text-lo text-sm">You don't have permission to access the admin panel.</p>
      </div>
      <button onClick={() => navigate('/')} className="btn-primary px-6 py-2.5 text-sm">
        Go Home
      </button>
    </div>
  )
}

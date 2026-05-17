import { useEffect, useState } from 'react'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  Home, Compass, BookOpen, MessageCircle,
  Bell, User, Settings, LogOut, Aperture, Sun, Moon, PlusSquare,
} from 'lucide-react'
import { toast } from 'sonner'
import api, { getRefreshToken, clearRefreshToken } from '../../lib/api'
import useAuthStore from '../../store/authStore'
import useThemeStore from '../../store/themeStore'
import PostCreationModal from '../feed/PostCreationModal'

const NAV = [
  { to: '/',            icon: Home,          label: 'Home' },
  { to: '/explore',     icon: Compass,       label: 'Explore' },
  { to: '/stories',     icon: BookOpen,      label: 'Stories' },
  { to: '/chat',        icon: MessageCircle, label: 'Messages',      badge: 'messages' },
  { to: '/notifications', icon: Bell,        label: 'Notifications', badge: 'notifs' },
  { to: '/profile',     icon: User,          label: 'Profile' },
  { to: '/settings',    icon: Settings,      label: 'Settings' },
]

function SidebarLink({ to, icon: Icon, label, count }) {
  return (
    <NavLink
      to={to}
      end={to === '/'}
      className={({ isActive }) =>
        `relative flex items-center gap-3 px-4 py-2.5 rounded-btn text-sm font-medium transition-all duration-150
         ${isActive
           ? 'text-accent bg-[var(--accent-glow)]'
           : 'text-lo hover:text-hi hover:bg-elevated'
         }`
      }
    >
      {({ isActive }) => (
        <>
          {isActive && (
            <span
              className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 rounded-r-full"
              style={{ background: 'var(--accent)' }}
            />
          )}
          <Icon size={18} />
          <span className="flex-1">{label}</span>
          {count > 0 && (
            <span
              className="min-w-[18px] h-[18px] rounded-full text-white text-[10px] font-bold flex items-center justify-center px-1"
              style={{ background: 'var(--accent)' }}
            >
              {count > 99 ? '99+' : count}
            </span>
          )}
        </>
      )}
    </NavLink>
  )
}

export default function AppLayout() {
  const { user, logout } = useAuthStore()
  const { theme, toggleTheme } = useThemeStore()
  const navigate = useNavigate()
  const [unreadNotifs, setUnreadNotifs] = useState(0)
  const [showCreatePost, setShowCreatePost] = useState(false)

  useEffect(() => {
    function fetchCount() {
      api.get('/notifications/count')
        .then(({ data }) => setUnreadNotifs(data.count ?? data.unread_count ?? 0))
        .catch(() => {})
    }
    fetchCount()
    const id = setInterval(fetchCount, 30000)
    return () => clearInterval(id)
  }, [])

  async function handleLogout() {
    const rt = getRefreshToken()
    try {
      await api.post('/auth/logout', { refresh_token: rt })
    } catch {
      /* swallow */
    }
    clearRefreshToken()
    logout()
    toast.success('Logged out')
    navigate('/login')
  }

  return (
    <div className="flex min-h-dvh" style={{ background: 'var(--bg)' }}>
      {/* ── Sidebar ─────────────────────────────────────────── */}
      <aside
        className="hidden md:flex flex-col w-60 shrink-0 sticky top-0 h-dvh border-r"
        style={{
          background: 'var(--surface)',
          borderColor: 'var(--border)',
        }}
      >
        {/* Logo */}
        <div className="flex items-center gap-2 px-5 h-16 select-none shrink-0">
          <Aperture size={22} style={{ color: 'var(--accent)' }} />
          <span className="text-base font-bold tracking-tight text-hi">Social</span>
        </div>

        <div
          className="mx-4 mb-4 h-px"
          style={{ background: 'var(--border)' }}
        />

        {/* Nav */}
        <nav className="flex-1 flex flex-col gap-0.5 px-2 overflow-y-auto">
          {NAV.map((item) => (
            <SidebarLink
              key={item.to}
              {...item}
              count={item.badge === 'notifs' ? unreadNotifs : 0}
            />
          ))}

          {/* Create post button — styled like a NavLink */}
          <button
            onClick={() => setShowCreatePost(true)}
            className="relative flex items-center gap-3 px-4 py-2.5 rounded-btn text-sm font-medium text-lo hover:text-hi hover:bg-elevated transition-all duration-150"
          >
            <PlusSquare size={18} />
            <span className="flex-1 text-left">Create</span>
          </button>
        </nav>

        <div
          className="mx-4 mt-4 h-px"
          style={{ background: 'var(--border)' }}
        />

        {/* Bottom: theme toggle + user */}
        <div className="p-3 shrink-0 space-y-1">
          {/* Theme toggle */}
          <button
            onClick={toggleTheme}
            className="btn-ghost w-full justify-start gap-3 text-sm"
          >
            {theme === 'dark'
              ? <Sun size={16} />
              : <Moon size={16} />
            }
            {theme === 'dark' ? 'Light mode' : 'Dark mode'}
          </button>

          {/* User row */}
          <div
            className="flex items-center gap-3 px-3 py-2 rounded-btn"
            style={{ background: 'var(--surface-high)' }}
          >
            <div
              className="w-8 h-8 rounded-full shrink-0 flex items-center justify-center text-xs font-bold text-white"
              style={{ background: 'var(--accent)' }}
            >
              {user?.username?.[0]?.toUpperCase() ?? 'U'}
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-xs font-semibold text-hi truncate">
                {user?.full_name || user?.username}
              </p>
              <p className="text-[11px] text-lo truncate">@{user?.username}</p>
            </div>
            <button
              onClick={handleLogout}
              className="text-lo hover:text-danger transition-colors shrink-0"
              title="Logout"
            >
              <LogOut size={14} />
            </button>
          </div>
        </div>
      </aside>

      {/* ── Main ──────────────────────────────────────────── */}
      <main className="flex-1 min-w-0">
        <Outlet />
      </main>

      {/* ── Mobile bottom tab bar ─────────────────────────── */}
      <nav
        className="md:hidden fixed bottom-0 inset-x-0 flex justify-around items-center h-16 border-t z-50"
        style={{
          background: 'var(--surface)',
          borderColor: 'var(--border)',
        }}
      >
        {[NAV[0], NAV[1]].map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              `flex flex-col items-center gap-1 px-3 py-2 transition-colors
               ${isActive ? 'text-accent' : 'text-lo'}`
            }
          >
            <Icon size={20} />
            <span className="text-[10px] font-medium">{label}</span>
          </NavLink>
        ))}
        <button
          onClick={() => setShowCreatePost(true)}
          className="flex flex-col items-center gap-1 px-3 py-2 transition-colors text-lo"
        >
          <PlusSquare size={20} />
          <span className="text-[10px] font-medium">Create</span>
        </button>
        {[NAV[3], NAV[5]].map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            end={to === '/'}
            className={({ isActive }) =>
              `flex flex-col items-center gap-1 px-3 py-2 transition-colors
               ${isActive ? 'text-accent' : 'text-lo'}`
            }
          >
            <Icon size={20} />
            <span className="text-[10px] font-medium">{label}</span>
          </NavLink>
        ))}
      </nav>

      {/* Post creation modal */}
      {showCreatePost && (
        <PostCreationModal
          onClose={() => setShowCreatePost(false)}
          onCreated={() => {
            setShowCreatePost(false)
            // Reload feed if currently on it
            if (window.location.pathname === '/') {
              window.location.reload()
            } else {
              navigate('/')
            }
          }}
        />
      )}
    </div>
  )
}

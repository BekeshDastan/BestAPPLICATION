import { Link, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../AuthContext'

const ICON = {
  home: '⊞',
  explore: '◎',
  stories: '◌',
  chat: '✉',
  profile: '◉',
  settings: '⚙',
  logout: '⏻',
}

function NavItem({ to, icon, label, active }) {
  return (
    <Link to={to} style={{
      display: 'flex', alignItems: 'center', gap: 12,
      padding: '12px 16px', borderRadius: 8,
      background: active ? '#f0f0f0' : 'transparent',
      color: active ? '#262626' : '#555',
      fontWeight: active ? 600 : 400,
      fontSize: 15, transition: 'background .15s',
    }}>
      <span style={{ fontSize: 20, lineHeight: 1 }}>{icon}</span>
      {label}
    </Link>
  )
}

export default function Sidebar() {
  const { user, logout } = useAuth()
  const { pathname } = useLocation()
  const nav = useNavigate()

  const doLogout = async () => { await logout(); nav('/login') }

  return (
    <div style={{
      position: 'fixed', top: 0, left: 0, height: '100vh', width: 240,
      background: '#fff', borderRight: '1px solid #dbdbdb',
      display: 'flex', flexDirection: 'column', padding: '24px 16px', zIndex: 100,
    }}>
      <div style={{ fontWeight: 700, fontSize: 22, color: '#262626', marginBottom: 32, paddingLeft: 16 }}>
        Social
      </div>
      <nav style={{ display: 'flex', flexDirection: 'column', gap: 4, flex: 1 }}>
        <NavItem to="/" icon={ICON.home} label="Home" active={pathname === '/'} />
        <NavItem to="/explore" icon={ICON.explore} label="Explore" active={pathname === '/explore'} />
        <NavItem to="/stories" icon={ICON.stories} label="Stories" active={pathname === '/stories'} />
        <NavItem to="/chat" icon={ICON.chat} label="Messages" active={pathname === '/chat'} />
        <NavItem to={`/profile/${user?.id}`} icon={ICON.profile} label="Profile" active={pathname.startsWith('/profile')} />
        <NavItem to="/settings" icon={ICON.settings} label="Settings" active={pathname === '/settings'} />
      </nav>
      <div style={{ borderTop: '1px solid #eee', paddingTop: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '0 8px 12px' }}>
          <div style={styles.avatar}>{user?.username?.[0]?.toUpperCase() || '?'}</div>
          <div>
            <div style={{ fontWeight: 600, fontSize: 14 }}>{user?.username}</div>
            <div style={{ fontSize: 12, color: '#999' }}>{user?.full_name}</div>
          </div>
        </div>
        <button onClick={doLogout} style={styles.logoutBtn}>
          <span style={{ fontSize: 18 }}>{ICON.logout}</span> Logout
        </button>
      </div>
    </div>
  )
}

const styles = {
  avatar: { width: 36, height: 36, borderRadius: '50%', background: '#333', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 700, fontSize: 14, flexShrink: 0 },
  logoutBtn: { display: 'flex', alignItems: 'center', gap: 10, width: '100%', padding: '10px 16px', background: 'none', border: 'none', borderRadius: 8, color: '#e53935', fontSize: 14, fontWeight: 500 },
}

import { lazy, Suspense, useEffect } from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import {
  setLogoutCallback,
  getRefreshToken,
  setRefreshToken,
  clearRefreshToken,
} from './lib/api'
import api from './lib/api'
import useAuthStore from './store/authStore'
import useWsStore from './store/wsStore'
import OfflineBanner from './components/shared/OfflineBanner'

/* ── Layouts (not lazy — always needed) ─────────────────────────── */
import AuthLayout  from './components/layout/AuthLayout'
import AppLayout   from './components/layout/AppLayout'
import AdminLayout from './pages/admin/AdminLayout'

/* ── Auth pages ─────────────────────────────────────────────────── */
const LoginPage         = lazy(() => import('./pages/auth/LoginPage'))
const RegisterPage      = lazy(() => import('./pages/auth/RegisterPage'))
const VerifyEmailPage   = lazy(() => import('./pages/auth/VerifyEmailPage'))
const ForgotPasswordPage = lazy(() => import('./pages/auth/ForgotPasswordPage'))
const ResetPasswordPage = lazy(() => import('./pages/auth/ResetPasswordPage'))

/* ── App pages ──────────────────────────────────────────────────── */
const FeedPage          = lazy(() => import('./pages/feed/FeedPage'))
const ExplorePage       = lazy(() => import('./pages/explore/ExplorePage'))
const HashtagPage       = lazy(() => import('./pages/hashtag/HashtagPage'))
const ProfilePage       = lazy(() => import('./pages/profile/ProfilePage'))
const StoriesPage       = lazy(() => import('./pages/stories/StoriesPage'))
const ChatPage          = lazy(() => import('./pages/chat/ChatPage'))
const NotificationsPage = lazy(() => import('./pages/notifications/NotificationsPage'))
const SettingsPage      = lazy(() => import('./pages/settings/SettingsPage'))
const SavedPage         = lazy(() => import('./pages/saved/SavedPage'))
const PostPage          = lazy(() => import('./pages/post/PostPage'))

/* ── Admin pages ────────────────────────────────────────────────── */
const AdminDashboard = lazy(() => import('./pages/admin/AdminDashboard'))
const AdminUsers     = lazy(() => import('./pages/admin/AdminUsers'))
const AdminPosts     = lazy(() => import('./pages/admin/AdminPosts'))
const AdminStories   = lazy(() => import('./pages/admin/AdminStories'))
const AdminReports   = lazy(() => import('./pages/admin/AdminReports'))
const AdminSystem    = lazy(() => import('./pages/admin/AdminSystem'))

/* ── Fallback while lazy chunks load ───────────────────────────── */
function PageSpinner() {
  return (
    <div className="flex-1 flex items-center justify-center min-h-screen">
      <div className="w-8 h-8 rounded-full border-2 border-t-transparent animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
    </div>
  )
}

function ProtectedRoute({ children }) {
  const { isAuthenticated, isLoading } = useAuthStore()
  if (isLoading) return null
  if (!isAuthenticated) return <Navigate to="/login" replace />
  return children
}

function AuthRoute({ children }) {
  const { isAuthenticated, isLoading } = useAuthStore()
  if (isLoading) return null
  if (isAuthenticated) return <Navigate to="/" replace />
  return children
}

export default function App() {
  const { login, logout, setLoading, isAuthenticated } = useAuthStore()
  const connect    = useWsStore((s) => s.connect)
  const disconnect = useWsStore((s) => s.disconnect)

  /* Register the logout callback so api.js can call it on refresh failure */
  useEffect(() => {
    setLogoutCallback(logout)
  }, [logout])

  /* Connect / disconnect WebSocket when auth state changes */
  useEffect(() => {
    if (isAuthenticated) {
      connect()
    } else {
      disconnect()
    }
  }, [isAuthenticated, connect, disconnect])

  /* Try to restore session using refresh token from localStorage */
  useEffect(() => {
    const rt = getRefreshToken()
    if (!rt) {
      setLoading(false)
      return
    }
    api
      .post('/auth/refresh', { refresh_token: rt })
      .then(async ({ data }) => {
        if (data.refresh_token) setRefreshToken(data.refresh_token)
        const me = await api.get('/users/me', {
          headers: { Authorization: `Bearer ${data.access_token}` },
        })
        login(data.access_token, me.data)
      })
      .catch(() => {
        clearRefreshToken()
        setLoading(false)
      })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <>
      <Suspense fallback={<PageSpinner />}>
        <Routes>
          {/* ── Auth routes ─────────────────────────────────── */}
          <Route element={<AuthLayout />}>
            <Route path="/login"           element={<AuthRoute><LoginPage /></AuthRoute>} />
            <Route path="/register"        element={<AuthRoute><RegisterPage /></AuthRoute>} />
            <Route path="/verify-email"    element={<VerifyEmailPage />} />
            <Route path="/forgot-password" element={<AuthRoute><ForgotPasswordPage /></AuthRoute>} />
            <Route path="/reset-password"  element={<ResetPasswordPage />} />
          </Route>

          {/* ── App routes ──────────────────────────────────── */}
          <Route element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
            <Route path="/"              element={<FeedPage />} />
            <Route path="/explore"       element={<ExplorePage />} />
            <Route path="/hashtag/:tag"  element={<HashtagPage />} />
            <Route path="/profile"       element={<ProfilePage />} />
            <Route path="/profile/:id"   element={<ProfilePage />} />
            <Route path="/stories"       element={<StoriesPage />} />
            <Route path="/chat"          element={<ChatPage />} />
            <Route path="/chat/:convId"  element={<ChatPage />} />
            <Route path="/notifications" element={<NotificationsPage />} />
            <Route path="/settings"      element={<SettingsPage />} />
            <Route path="/saved"         element={<SavedPage />} />
            <Route path="/posts/:id"     element={<PostPage />} />
          </Route>

          {/* ── Admin routes ────────────────────────────────── */}
          <Route
            path="/admin"
            element={<ProtectedRoute><AdminLayout /></ProtectedRoute>}
          >
            <Route index          element={<AdminDashboard />} />
            <Route path="users"   element={<AdminUsers />} />
            <Route path="posts"   element={<AdminPosts />} />
            <Route path="stories" element={<AdminStories />} />
            <Route path="reports" element={<AdminReports />} />
            <Route path="system"  element={<AdminSystem />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>

      <OfflineBanner />
    </>
  )
}

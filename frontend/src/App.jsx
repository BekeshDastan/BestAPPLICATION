import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './AuthContext'
import Sidebar from './components/Sidebar'
import LoginPage from './pages/LoginPage'
import RegisterPage from './pages/RegisterPage'
import ForgotPasswordPage from './pages/ForgotPasswordPage'
import FeedPage from './pages/FeedPage'
import ExplorePage from './pages/ExplorePage'
import ProfilePage from './pages/ProfilePage'
import ChatPage from './pages/ChatPage'
import StoriesPage from './pages/StoriesPage'
import SettingsPage from './pages/SettingsPage'

function Private({ children }) {
  const { user } = useAuth()
  return user ? children : <Navigate to="/login" replace />
}

export default function App() {
  const { user } = useAuth()
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      {user && <Sidebar />}
      <div style={{ flex: 1, marginLeft: user ? 240 : 0 }}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<RegisterPage />} />
          <Route path="/forgot-password" element={<ForgotPasswordPage />} />
          <Route path="/" element={<Private><FeedPage /></Private>} />
          <Route path="/explore" element={<Private><ExplorePage /></Private>} />
          <Route path="/profile/:id" element={<Private><ProfilePage /></Private>} />
          <Route path="/chat" element={<Private><ChatPage /></Private>} />
          <Route path="/stories" element={<Private><StoriesPage /></Private>} />
          <Route path="/settings" element={<Private><SettingsPage /></Private>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </div>
    </div>
  )
}

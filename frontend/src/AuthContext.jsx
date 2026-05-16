import { createContext, useContext, useState, useCallback } from 'react'
import { authApi, userApi } from './api'

const Ctx = createContext(null)

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => {
    try { return JSON.parse(localStorage.getItem('user')) } catch { return null }
  })

  const saveAuth = (tokens, u) => {
    localStorage.setItem('access_token', tokens.access_token)
    localStorage.setItem('refresh_token', tokens.refresh_token)
    localStorage.setItem('user', JSON.stringify(u))
    setUser(u)
  }

  const login = useCallback(async (email, password) => {
    const { data } = await authApi.login(email, password)
    saveAuth(data.tokens, data.user)
    return data.user
  }, [])

  const register = useCallback(async (email, username, password, full_name) => {
    const { data } = await authApi.register(email, username, password, full_name)
    saveAuth(data.tokens, data.user)
    return data.user
  }, [])

  const logout = useCallback(async () => {
    const rf = localStorage.getItem('refresh_token')
    try { await authApi.logout(rf) } catch {}
    localStorage.clear()
    setUser(null)
  }, [])

  const refreshUser = useCallback(async () => {
    try {
      const { data } = await userApi.getMe()
      const u = data.user || data
      localStorage.setItem('user', JSON.stringify(u))
      setUser(u)
    } catch {}
  }, [])

  return <Ctx.Provider value={{ user, login, register, logout, refreshUser }}>{children}</Ctx.Provider>
}

export const useAuth = () => useContext(Ctx)

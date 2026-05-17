import { create } from 'zustand'
import { setAccessToken } from '../lib/api'

const useAuthStore = create((set, get) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true, // true until initial auth check resolves

  login(accessToken, user) {
    setAccessToken(accessToken)
    set({ user, isAuthenticated: true, isLoading: false })
  },

  logout() {
    setAccessToken(null)
    set({ user: null, isAuthenticated: false, isLoading: false })
  },

  setUser(user) {
    set({ user })
  },

  setLoading(isLoading) {
    set({ isLoading })
  },
}))

export default useAuthStore

import { create } from 'zustand'

const STORAGE_KEY = 'social-theme'

const saved = localStorage.getItem(STORAGE_KEY) ?? 'dark'
document.documentElement.setAttribute('data-theme', saved)

const useThemeStore = create((set) => ({
  theme: saved,

  toggleTheme() {
    set((s) => {
      const next = s.theme === 'dark' ? 'light' : 'dark'
      localStorage.setItem(STORAGE_KEY, next)
      document.documentElement.setAttribute('data-theme', next)
      return { theme: next }
    })
  },

  setTheme(theme) {
    localStorage.setItem(STORAGE_KEY, theme)
    document.documentElement.setAttribute('data-theme', theme)
    set({ theme })
  },
}))

export default useThemeStore

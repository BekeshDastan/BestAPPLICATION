import axios from 'axios'
import { toast } from 'sonner'

/* ─── Token store ────────────────────────────────────────────────────── */
const REFRESH_KEY = 'social.refresh_token'

let _accessToken = null
let _onLogout = null

export const setAccessToken = (token) => { _accessToken = token }
export const getAccessToken = () => _accessToken
export const setLogoutCallback = (cb) => { _onLogout = cb }

export const setRefreshToken = (token) => {
  if (token) localStorage.setItem(REFRESH_KEY, token)
  else       localStorage.removeItem(REFRESH_KEY)
}
export const getRefreshToken = () => localStorage.getItem(REFRESH_KEY)
export const clearRefreshToken = () => localStorage.removeItem(REFRESH_KEY)

/* ─── Axios instance ─────────────────────────────────────────────────── */
const api = axios.create({
  baseURL: '/api/v1',
})

/* ─── Request interceptor: attach Bearer token ───────────────────────── */
api.interceptors.request.use((config) => {
  if (_accessToken) {
    config.headers.Authorization = `Bearer ${_accessToken}`
  }
  return config
})

/* ─── Response interceptor: silent token refresh on 401 ─────────────── */
let _refreshing = false
let _queue = []

const processQueue = (error, token = null) => {
  _queue.forEach(({ resolve, reject }) => {
    if (error) reject(error)
    else resolve(token)
  })
  _queue = []
}

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config

    const status = error.response?.status

    if (status === 403) {
      toast.error("You don't have permission to do that.")
    } else if (status === 429) {
      toast.error('Slow down! Try again in a moment.')
    }

    if (status !== 401 || original._retry) {
      return Promise.reject(error)
    }

    original._retry = true

    if (_refreshing) {
      return new Promise((resolve, reject) => {
        _queue.push({ resolve, reject })
      }).then((token) => {
        original.headers.Authorization = `Bearer ${token}`
        return api(original)
      })
    }

    _refreshing = true

    try {
      const refreshToken = getRefreshToken()
      if (!refreshToken) throw new Error('no refresh token')

      const { data } = await axios.post('/api/v1/auth/refresh', {
        refresh_token: refreshToken,
      })
      setAccessToken(data.access_token)
      if (data.refresh_token) setRefreshToken(data.refresh_token)
      processQueue(null, data.access_token)
      original.headers.Authorization = `Bearer ${data.access_token}`
      return api(original)
    } catch (err) {
      processQueue(err)
      setAccessToken(null)
      clearRefreshToken()
      _onLogout?.()
      return Promise.reject(err)
    } finally {
      _refreshing = false
    }
  },
)

export default api

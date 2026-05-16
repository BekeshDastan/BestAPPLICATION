import axios from 'axios'

const api = axios.create({ baseURL: '/api/v1' })

let isRefreshing = false
let queue = []

api.interceptors.request.use(cfg => {
  const t = localStorage.getItem('access_token')
  if (t) cfg.headers.Authorization = `Bearer ${t}`
  return cfg
})

api.interceptors.response.use(
  r => r,
  async err => {
    const orig = err.config
    if (err.response?.status === 401 && !orig._retry) {
      orig._retry = true
      const rf = localStorage.getItem('refresh_token')
      if (!rf) { localStorage.clear(); window.location.href = '/login'; return Promise.reject(err) }
      if (isRefreshing) {
        return new Promise((res, rej) => queue.push({ res, rej }))
          .then(tok => { orig.headers.Authorization = `Bearer ${tok}`; return api.request(orig) })
      }
      isRefreshing = true
      try {
        const { data } = await axios.post('/api/v1/auth/refresh', { refresh_token: rf })
        const at = data.tokens?.access_token || data.access_token
        const rt = data.tokens?.refresh_token || data.refresh_token
        localStorage.setItem('access_token', at)
        if (rt) localStorage.setItem('refresh_token', rt)
        queue.forEach(p => p.res(at)); queue = []
        orig.headers.Authorization = `Bearer ${at}`
        return api.request(orig)
      } catch (e) {
        queue.forEach(p => p.rej(e)); queue = []
        localStorage.clear(); window.location.href = '/login'
        return Promise.reject(e)
      } finally { isRefreshing = false }
    }
    return Promise.reject(err)
  }
)

export const authApi = {
  register: (email, username, password, full_name) => api.post('/auth/register', { email, username, password, full_name }),
  login: (email, password) => api.post('/auth/login', { email, password }),
  logout: (rf) => api.post('/auth/logout', { refresh_token: rf }),
  forgotPassword: (email) => api.post('/auth/forgot-password', { email }),
  resetPassword: (token, new_password) => api.post('/auth/reset-password', { token, new_password }),
}

export const userApi = {
  getMe: () => api.get('/users/me'),
  updateMe: (data) => api.put('/users/me', data),
  getProfile: (id) => api.get(`/users/${id}`),
  searchUsers: (q, limit = 20) => api.get('/users/search', { params: { q, limit } }),
  isFollowing: (id) => api.get(`/users/${id}/is-following`),
  follow: (id) => api.post(`/users/${id}/follow`),
  unfollow: (id) => api.delete(`/users/${id}/follow`),
  listFollowers: (id) => api.get(`/users/${id}/followers`, { params: { limit: 50 } }),
  listFollowing: (id) => api.get(`/users/${id}/following`, { params: { limit: 50 } }),
  listUserPosts: (id, limit = 20, offset = 0) => api.get(`/users/${id}/posts`, { params: { limit, offset } }),
}

export const storyApi = {
  create: (media_url, media_type = 'image', caption = '') => api.post('/stories', { media_url, media_type, caption }),
  get: (id) => api.get(`/stories/${id}`),
  delete: (id) => api.delete(`/stories/${id}`),
  listUser: (user_id, limit = 20) => api.get(`/stories/user/${user_id}`, { params: { limit } }),
  listFollowing: () => api.get('/stories/following'),
  markViewed: (id) => api.post(`/stories/${id}/view`),
  listViewers: (id) => api.get(`/stories/${id}/viewers`),
  reply: (id, text) => api.post(`/stories/${id}/reply`, { text }),
  addReaction: (id, emoji) => api.post(`/stories/${id}/reaction`, { emoji }),
  removeReaction: (id) => api.delete(`/stories/${id}/reaction`),
  analytics: (id) => api.get(`/stories/${id}/analytics`),
  createHighlight: (title, cover_url = '') => api.post('/highlights', { title, cover_url }),
  listHighlights: (user_id) => api.get(`/highlights/user/${user_id}`),
  deleteHighlight: (id) => api.delete(`/highlights/${id}`),
  addToHighlight: (highlight_id, story_id) => api.post(`/highlights/${highlight_id}/stories`, { story_id }),
  removeFromHighlight: (highlight_id, story_id) => api.delete(`/highlights/${highlight_id}/stories/${story_id}`),
}

export const postApi = {
  create: (caption, media_urls = [], tags = []) => api.post('/posts', { caption, media_urls, tags }),
  get: (id) => api.get(`/posts/${id}`),
  update: (id, caption, tags = []) => api.put(`/posts/${id}`, { caption, tags }),
  delete: (id) => api.delete(`/posts/${id}`),
  feed: (limit = 20, offset = 0) => api.get('/posts/feed', { params: { limit, offset } }),
  search: (q, limit = 20, offset = 0) => api.get('/posts/search', { params: { q, limit, offset } }),
  like: (id) => api.post(`/posts/${id}/like`),
  unlike: (id) => api.delete(`/posts/${id}/like`),
  listComments: (id, limit = 30) => api.get(`/posts/${id}/comments`, { params: { limit } }),
  addComment: (id, body) => api.post(`/posts/${id}/comments`, { body }),
  deleteComment: (pid, cid) => api.delete(`/posts/${pid}/comments/${cid}`),
}

export const chatApi = {
  list: () => api.get('/chats', { params: { limit: 30 } }),
  create: (member_ids) => api.post('/chats', { member_ids }),
  get: (id) => api.get(`/chats/${id}`),
  delete: (id) => api.delete(`/chats/${id}`),
  listMessages: (id, limit = 50) => api.get(`/chats/${id}/messages`, { params: { limit } }),
  sendMessage: (id, text, media_url = '') => api.post(`/chats/${id}/messages`, { text, media_url }),
  deleteMessage: (cid, mid) => api.delete(`/chats/${cid}/messages/${mid}`),
}

export default api

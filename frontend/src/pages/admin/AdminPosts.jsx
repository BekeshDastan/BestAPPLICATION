import { useState, useEffect, useCallback } from 'react'
import { Search, Trash2, Eye, ChevronLeft, ChevronRight } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../../components/shared/Avatar'
import PostDetailModal from '../../components/shared/PostDetailModal'
import { formatRelativeTime, formatCount } from '../../lib/utils'
import { ADMIN_ACCENT } from './AdminLayout'

const PAGE = 20

export default function AdminPosts() {
  const [posts,   setPosts]   = useState([])
  const [total,   setTotal]   = useState(0)
  const [page,    setPage]    = useState(1)
  const [loading, setLoading] = useState(true)
  const [search,  setSearch]  = useState('')
  const [dateFrom, setDateFrom] = useState('')
  const [dateTo,   setDateTo]   = useState('')
  const [selected, setSelected] = useState(null)
  const [confirm,  setConfirm]  = useState(null)

  const load = useCallback(async (p = page) => {
    setLoading(true)
    try {
      const params = { page: p, limit: PAGE }
      if (search.trim()) params.q    = search.trim()
      if (dateFrom)      params.from = dateFrom
      if (dateTo)        params.to   = dateTo
      const { data } = await api.get('/admin/posts', { params })
      setPosts(data.posts ?? [])
      setTotal(data.total ?? 0)
    } catch { toast.error('Failed to load posts') }
    finally { setLoading(false) }
  }, [page, search, dateFrom, dateTo]) // eslint-disable-line

  useEffect(() => { load(page) }, [page]) // eslint-disable-line
  function applyFilters() { setPage(1); load(1) }

  async function deletePost(post) {
    try {
      await api.delete(`/admin/posts/${post.id}`)
      toast.success('Post deleted')
      setPosts((prev) => prev.filter((p) => p.id !== post.id))
      setTotal((t) => t - 1)
    } catch { toast.error('Failed to delete post') }
    setConfirm(null)
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE))

  return (
    <div className="p-6">
      <h1 className="text-xl font-bold text-white mb-5">Posts</h1>

      {/* Filters */}
      <div
        className="rounded-card p-4 mb-4 flex flex-wrap items-end gap-3"
        style={{ background: '#161616', border: '1px solid #1F1F1F' }}
      >
        <div className="flex-1 min-w-48 relative">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" style={{ color: '#52525B' }} />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && applyFilters()}
            placeholder="Search by caption or author…"
            className="w-full pl-9 pr-3 py-2 text-sm rounded-btn outline-none"
            style={{ background: '#1A1A1A', border: '1px solid #2E2E2E', color: '#fff' }}
            onFocus={(e) => (e.target.style.borderColor = ADMIN_ACCENT)}
            onBlur={(e)  => (e.target.style.borderColor = '#2E2E2E')}
          />
        </div>
        <div>
          <p className="text-[10px] mb-1" style={{ color: '#52525B' }}>From</p>
          <input type="date" value={dateFrom} onChange={(e) => setDateFrom(e.target.value)} className="px-3 py-2 text-sm rounded-btn outline-none" style={{ background: '#1A1A1A', border: '1px solid #2E2E2E', color: '#A1A1AA' }} />
        </div>
        <div>
          <p className="text-[10px] mb-1" style={{ color: '#52525B' }}>To</p>
          <input type="date" value={dateTo} onChange={(e) => setDateTo(e.target.value)} className="px-3 py-2 text-sm rounded-btn outline-none" style={{ background: '#1A1A1A', border: '1px solid #2E2E2E', color: '#A1A1AA' }} />
        </div>
        <button onClick={applyFilters} className="px-4 py-2 text-sm rounded-btn" style={{ background: ADMIN_ACCENT, color: '#fff' }}>
          Apply
        </button>
        <button onClick={() => { setSearch(''); setDateFrom(''); setDateTo(''); setPage(1); }} className="px-3 py-2 text-xs rounded-btn" style={{ color: '#71717A', background: '#1A1A1A' }}>
          Reset
        </button>
      </div>

      {/* Table */}
      <div className="rounded-card overflow-hidden" style={{ background: '#161616', border: '1px solid #1F1F1F' }}>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: '1px solid #1F1F1F' }}>
                {['', 'Author', 'Caption', '❤️', '💬', 'Created', 'Actions'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-semibold" style={{ color: '#52525B' }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading
                ? Array.from({ length: 8 }).map((_, i) => <RowSkeleton key={i} cols={7} />)
                : posts.map((p, idx) => (
                  <tr
                    key={p.id}
                    style={{ borderBottom: idx < posts.length - 1 ? '1px solid #1F1F1F' : 'none' }}
                    onMouseEnter={(e) => (e.currentTarget.style.background = '#1A1A1A')}
                    onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                  >
                    {/* Thumbnail */}
                    <td className="px-4 py-3">
                      <div
                        className="w-12 h-12 rounded-btn overflow-hidden shrink-0 flex items-center justify-center"
                        style={{ background: '#1A1A1A' }}
                      >
                        {p.media_url ? (
                          <img src={p.media_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          <span className="text-lg">📝</span>
                        )}
                      </div>
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-2">
                        <Avatar src={p.author?.avatar_url} name={p.author?.username} size={26} />
                        <span className="text-white text-xs font-medium">@{p.author?.username ?? '—'}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 max-w-xs">
                      <p className="text-xs truncate" style={{ color: '#A1A1AA', maxWidth: 240 }}>
                        {p.caption ? `"${p.caption.slice(0, 60)}${p.caption.length > 60 ? '…' : ''}"` : '—'}
                      </p>
                    </td>
                    <td className="px-4 py-3 text-xs" style={{ color: '#A1A1AA' }}>{formatCount(p.likes_count ?? 0)}</td>
                    <td className="px-4 py-3 text-xs" style={{ color: '#A1A1AA' }}>{formatCount(p.comments_count ?? 0)}</td>
                    <td className="px-4 py-3 text-xs" style={{ color: '#52525B' }}>{formatRelativeTime(p.created_at)}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1.5">
                        <ActionBtn icon={<Eye size={13} />} onClick={() => setSelected(p)} title="View" />
                        <ActionBtn icon={<Trash2 size={13} />} onClick={() => setConfirm(p)} title="Delete" color="#EF4444" />
                      </div>
                    </td>
                  </tr>
                ))
              }
            </tbody>
          </table>
        </div>

        {!loading && (
          <div className="flex items-center justify-between px-4 py-3 border-t" style={{ borderColor: '#1F1F1F' }}>
            <p className="text-xs" style={{ color: '#52525B' }}>
              {total} posts · Page {page} of {totalPages}
            </p>
            <Pagination current={page} total={totalPages} onChange={setPage} />
          </div>
        )}
      </div>

      {selected && <PostDetailModal post={selected} onClose={() => setSelected(null)} />}

      {confirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.8)' }}>
          <div className="rounded-card p-6 w-72 text-center animate-fade-in" style={{ background: '#1A1A1A', border: '1px solid #2E2E2E' }}>
            <p className="font-semibold text-white mb-2">Delete this post?</p>
            <p className="text-sm mb-5" style={{ color: '#71717A' }}>This action cannot be undone.</p>
            <div className="flex gap-3">
              <button onClick={() => setConfirm(null)} className="flex-1 py-2 rounded-btn text-sm" style={{ border: '1px solid #2E2E2E', color: '#A1A1AA' }}>Cancel</button>
              <button onClick={() => deletePost(confirm)} className="flex-1 py-2 rounded-btn text-sm text-white" style={{ background: '#EF4444' }}>Delete</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function ActionBtn({ icon, onClick, title, color }) {
  return (
    <button onClick={onClick} title={title} className="w-6 h-6 rounded flex items-center justify-center transition-colors" style={{ color: color ?? '#71717A', background: '#1F1F1F' }}
      onMouseEnter={(e) => (e.currentTarget.style.color = color ?? '#fff')}
      onMouseLeave={(e) => (e.currentTarget.style.color = color ?? '#71717A')}
    >{icon}</button>
  )
}

function Pagination({ current, total, onChange }) {
  const pages = []
  for (let i = Math.max(1, current - 2); i <= Math.min(total, current + 2); i++) pages.push(i)
  return (
    <div className="flex items-center gap-1">
      <PageBtn disabled={current === 1} onClick={() => onChange(current - 1)} icon={<ChevronLeft size={14} />} />
      {pages.map((p) => <PageBtn key={p} label={p} active={p === current} onClick={() => onChange(p)} />)}
      <PageBtn disabled={current === total} onClick={() => onChange(current + 1)} icon={<ChevronRight size={14} />} />
    </div>
  )
}

function PageBtn({ label, icon, active, disabled, onClick }) {
  return (
    <button onClick={onClick} disabled={disabled} className="min-w-[28px] h-7 px-1.5 rounded text-xs font-medium disabled:opacity-30" style={{ background: active ? ADMIN_ACCENT : '#1A1A1A', color: active ? '#fff' : '#71717A' }}>
      {icon ?? label}
    </button>
  )
}

function RowSkeleton({ cols }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} className="px-4 py-3.5">
          <div className="skeleton h-3.5 rounded" style={{ width: i === 0 ? 48 : i === 2 ? 160 : 80 }} />
        </td>
      ))}
    </tr>
  )
}

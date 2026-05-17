import { useState, useEffect, useCallback } from 'react'
import { Trash2, ChevronLeft, ChevronRight } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../../components/shared/Avatar'
import { formatRelativeTime, formatCount } from '../../lib/utils'
import { ADMIN_ACCENT } from './AdminLayout'

const PAGE = 20

export default function AdminStories() {
  const [stories, setStories] = useState([])
  const [total,   setTotal]   = useState(0)
  const [page,    setPage]    = useState(1)
  const [loading, setLoading] = useState(true)
  const [confirm, setConfirm] = useState(null)

  const load = useCallback(async (p = page) => {
    setLoading(true)
    try {
      const { data } = await api.get('/admin/stories', { params: { page: p, limit: PAGE } })
      setStories(data.stories ?? [])
      setTotal(data.total ?? 0)
    } catch { toast.error('Failed to load stories') }
    finally { setLoading(false) }
  }, [page]) // eslint-disable-line

  useEffect(() => { load(page) }, [page]) // eslint-disable-line

  async function deleteStory(story) {
    try {
      await api.delete(`/admin/stories/${story.id}`)
      toast.success('Story deleted')
      setStories((prev) => prev.filter((s) => s.id !== story.id))
      setTotal((t) => t - 1)
    } catch { toast.error('Failed to delete story') }
    setConfirm(null)
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE))

  return (
    <div className="p-6">
      <h1 className="text-xl font-bold text-white mb-5">Stories</h1>

      <div className="rounded-card overflow-hidden" style={{ background: '#161616', border: '1px solid #1F1F1F' }}>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: '1px solid #1F1F1F' }}>
                {['', 'Author', 'Caption', '👁 Views', 'Reactions', 'Expires', 'Actions'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-semibold" style={{ color: '#52525B' }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading
                ? Array.from({ length: 8 }).map((_, i) => <RowSkeleton key={i} />)
                : stories.map((s, idx) => (
                  <tr
                    key={s.id}
                    style={{ borderBottom: idx < stories.length - 1 ? '1px solid #1F1F1F' : 'none' }}
                    onMouseEnter={(e) => (e.currentTarget.style.background = '#1A1A1A')}
                    onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td className="px-4 py-3">
                      <div className="w-12 h-16 rounded-btn overflow-hidden" style={{ background: '#1A1A1A' }}>
                        {s.media_url ? (
                          s.media_type === 'video'
                            ? <video src={s.media_url} className="w-full h-full object-cover" muted />
                            : <img src={s.media_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          <div className="w-full h-full flex items-center justify-center text-lg">📖</div>
                        )}
                      </div>
                    </td>
                    <td className="px-3 py-3">
                      <div className="flex items-center gap-2">
                        <Avatar src={s.author?.avatar_url} name={s.author?.username} size={26} />
                        <span className="text-white text-xs font-medium">@{s.author?.username ?? '—'}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3 max-w-xs">
                      <p className="text-xs truncate" style={{ color: '#A1A1AA', maxWidth: 160 }}>
                        {s.caption ? `"${s.caption.slice(0, 40)}${s.caption.length > 40 ? '…' : ''}"` : '—'}
                      </p>
                    </td>
                    <td className="px-4 py-3 text-xs" style={{ color: '#A1A1AA' }}>{formatCount(s.view_count ?? 0)}</td>
                    <td className="px-4 py-3 text-xs" style={{ color: '#A1A1AA' }}>{formatCount(s.reactions_count ?? 0)}</td>
                    <td className="px-4 py-3 text-xs" style={{ color: '#52525B' }}>
                      {s.expires_at ? formatRelativeTime(s.expires_at) : '—'}
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => setConfirm(s)}
                        title="Delete"
                        className="w-6 h-6 rounded flex items-center justify-center transition-colors"
                        style={{ color: '#EF4444', background: '#1F1F1F' }}
                        onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.15)')}
                        onMouseLeave={(e) => (e.currentTarget.style.background = '#1F1F1F')}
                      >
                        <Trash2 size={13} />
                      </button>
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
              {total} stories · Page {page} of {totalPages}
            </p>
            <Pagination current={page} total={totalPages} onChange={setPage} />
          </div>
        )}
      </div>

      {confirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.8)' }}>
          <div className="rounded-card p-6 w-72 text-center animate-fade-in" style={{ background: '#1A1A1A', border: '1px solid #2E2E2E' }}>
            <p className="font-semibold text-white mb-2">Delete this story?</p>
            <p className="text-sm mb-5" style={{ color: '#71717A' }}>This cannot be undone.</p>
            <div className="flex gap-3">
              <button onClick={() => setConfirm(null)} className="flex-1 py-2 rounded-btn text-sm" style={{ border: '1px solid #2E2E2E', color: '#A1A1AA' }}>Cancel</button>
              <button onClick={() => deleteStory(confirm)} className="flex-1 py-2 rounded-btn text-sm text-white" style={{ background: '#EF4444' }}>Delete</button>
            </div>
          </div>
        </div>
      )}
    </div>
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

function RowSkeleton() {
  return (
    <tr>
      {[48, 100, 140, 60, 60, 80, 40].map((w, i) => (
        <td key={i} className="px-4 py-3.5">
          <div className="skeleton h-3.5 rounded" style={{ width: w }} />
        </td>
      ))}
    </tr>
  )
}

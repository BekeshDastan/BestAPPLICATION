import { useState, useEffect, useCallback } from 'react'
import { ChevronDown } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'
import Avatar from '../../components/shared/Avatar'
import PostDetailModal from '../../components/shared/PostDetailModal'
import { formatRelativeTime } from '../../lib/utils'
import { ADMIN_ACCENT } from './AdminLayout'

const REASONS  = ['All', 'Spam', 'Harassment', 'Hate Speech', 'Violence', 'Nudity', 'Other']
const STATUSES = ['All', 'Pending', 'Resolved']
const PAGE     = 20

export default function AdminReports() {
  const [reports, setReports] = useState([])
  const [total,   setTotal]   = useState(0)
  const [page,    setPage]    = useState(1)
  const [loading, setLoading] = useState(true)
  const [reason,  setReason]  = useState('All')
  const [status,  setStatus]  = useState('Pending')
  const [viewPost, setViewPost] = useState(null)
  const [confirm,  setConfirm]  = useState(null) // { type: 'dismiss'|'delete', report }

  const load = useCallback(async (p = page) => {
    setLoading(true)
    try {
      const params = { page: p, limit: PAGE }
      if (reason !== 'All') params.reason = reason.toLowerCase().replace(' ', '_')
      if (status !== 'All') params.status = status.toLowerCase()
      const { data } = await api.get('/admin/reports', { params })
      setReports(data.reports ?? [])
      setTotal(data.total ?? 0)
    } catch { toast.error('Failed to load reports') }
    finally { setLoading(false) }
  }, [page, reason, status]) // eslint-disable-line

  useEffect(() => { setPage(1); load(1) }, [reason, status]) // eslint-disable-line
  useEffect(() => { load(page) }, [page]) // eslint-disable-line

  async function dismissReport(report) {
    try {
      await api.put(`/admin/reports/${report.id}/resolve`)
      toast.success('Report dismissed')
      setReports((prev) => prev.filter((r) => r.id !== report.id))
    } catch { toast.error('Failed to dismiss report') }
    setConfirm(null)
  }

  async function deletePost(report) {
    try {
      await api.delete(`/admin/posts/${report.post_id}`)
      await api.put(`/admin/reports/${report.id}/resolve`).catch(() => {})
      toast.success('Post deleted and report resolved')
      setReports((prev) => prev.filter((r) => r.id !== report.id))
    } catch { toast.error('Failed to delete post') }
    setConfirm(null)
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE))

  return (
    <div className="p-6">
      <h1 className="text-xl font-bold text-white mb-5">Reports</h1>

      {/* Filters */}
      <div
        className="rounded-card p-4 mb-4 flex items-end gap-3"
        style={{ background: '#161616', border: '1px solid #1F1F1F' }}
      >
        <AdminSelect label="Reason" value={reason} onChange={setReason} options={REASONS} />
        <AdminSelect label="Status" value={status} onChange={setStatus} options={STATUSES} />
        <button
          onClick={() => { setReason('All'); setStatus('Pending') }}
          className="px-3 py-2 text-xs rounded-btn"
          style={{ color: '#71717A', background: '#1A1A1A' }}
        >
          Reset
        </button>
      </div>

      <div className="rounded-card overflow-hidden" style={{ background: '#161616', border: '1px solid #1F1F1F' }}>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: '1px solid #1F1F1F' }}>
                {['Post', 'Reporter', 'Reason', 'Date', 'Status', 'Actions'].map((h) => (
                  <th key={h} className="px-4 py-3 text-left text-xs font-semibold" style={{ color: '#52525B' }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading
                ? Array.from({ length: 6 }).map((_, i) => <RowSkeleton key={i} />)
                : reports.length === 0
                  ? (
                    <tr>
                      <td colSpan={6} className="text-center py-12 text-xs" style={{ color: '#52525B' }}>
                        No reports found.
                      </td>
                    </tr>
                  )
                  : reports.map((r, idx) => (
                    <tr
                      key={r.id}
                      style={{ borderBottom: idx < reports.length - 1 ? '1px solid #1F1F1F' : 'none' }}
                      onMouseEnter={(e) => (e.currentTarget.style.background = '#1A1A1A')}
                      onMouseLeave={(e) => (e.currentTarget.style.background = 'transparent')}
                    >
                      {/* Post thumbnail */}
                      <td className="px-4 py-3">
                        <button
                          onClick={() => r.post && setViewPost(r.post)}
                          className="flex items-center gap-2 group"
                        >
                          <div className="w-10 h-10 rounded overflow-hidden shrink-0" style={{ background: '#1A1A1A' }}>
                            {r.post?.media_url
                              ? <img src={r.post.media_url} alt="" className="w-full h-full object-cover" />
                              : <div className="w-full h-full flex items-center justify-center text-sm">📝</div>
                            }
                          </div>
                          <span
                            className="text-xs group-hover:underline transition-colors"
                            style={{ color: ADMIN_ACCENT }}
                          >
                            View Post
                          </span>
                        </button>
                      </td>
                      {/* Reporter */}
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <Avatar src={r.reporter?.avatar_url} name={r.reporter?.username} size={24} />
                          <span className="text-xs text-white">@{r.reporter?.username ?? '—'}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className="text-[11px] px-2 py-0.5 rounded-full font-medium"
                          style={{ background: 'rgba(239,68,68,0.15)', color: '#EF4444' }}
                        >
                          {r.reason ?? 'Unknown'}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs" style={{ color: '#52525B' }}>
                        {formatRelativeTime(r.created_at)}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className="text-[11px] px-2 py-0.5 rounded-full font-medium"
                          style={{
                            background: r.status === 'resolved' ? `${ADMIN_ACCENT}20` : 'rgba(245,158,11,0.15)',
                            color:      r.status === 'resolved' ? ADMIN_ACCENT : '#F59E0B',
                          }}
                        >
                          {r.status === 'resolved' ? 'Resolved' : 'Pending'}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {r.status !== 'resolved' && (
                          <div className="flex items-center gap-2">
                            <button
                              onClick={() => setConfirm({ type: 'dismiss', report: r })}
                              className="text-xs px-3 py-1 rounded-btn transition-colors"
                              style={{ color: '#71717A', background: '#1F1F1F' }}
                              onMouseEnter={(e) => (e.currentTarget.style.color = '#fff')}
                              onMouseLeave={(e) => (e.currentTarget.style.color = '#71717A')}
                            >
                              Dismiss
                            </button>
                            <button
                              onClick={() => setConfirm({ type: 'delete', report: r })}
                              className="text-xs px-3 py-1 rounded-btn transition-colors"
                              style={{ color: '#EF4444', background: 'rgba(239,68,68,0.1)' }}
                              onMouseEnter={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.2)')}
                              onMouseLeave={(e) => (e.currentTarget.style.background = 'rgba(239,68,68,0.1)')}
                            >
                              Delete Post
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))
              }
            </tbody>
          </table>
        </div>

        {!loading && total > PAGE && (
          <div className="flex items-center justify-between px-4 py-3 border-t" style={{ borderColor: '#1F1F1F' }}>
            <p className="text-xs" style={{ color: '#52525B' }}>
              {total} reports · Page {page} of {totalPages}
            </p>
            <div className="flex items-center gap-1">
              <PageBtn disabled={page === 1}          onClick={() => setPage(page - 1)} label="←" />
              <PageBtn disabled={page === totalPages}  onClick={() => setPage(page + 1)} label="→" />
            </div>
          </div>
        )}
      </div>

      {viewPost && <PostDetailModal post={viewPost} onClose={() => setViewPost(null)} />}

      {confirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.8)' }}>
          <div className="rounded-card p-6 w-80 animate-fade-in" style={{ background: '#1A1A1A', border: '1px solid #2E2E2E' }}>
            <p className="font-semibold text-white mb-2">
              {confirm.type === 'dismiss' ? 'Dismiss this report?' : 'Delete the reported post?'}
            </p>
            <p className="text-sm mb-5" style={{ color: '#71717A' }}>
              {confirm.type === 'dismiss'
                ? 'The post will remain. The report will be marked as resolved.'
                : 'The post will be permanently deleted and the report resolved.'}
            </p>
            <div className="flex gap-3">
              <button onClick={() => setConfirm(null)} className="flex-1 py-2 rounded-btn text-sm" style={{ border: '1px solid #2E2E2E', color: '#A1A1AA' }}>
                Cancel
              </button>
              <button
                onClick={() => confirm.type === 'dismiss' ? dismissReport(confirm.report) : deletePost(confirm.report)}
                className="flex-1 py-2 rounded-btn text-sm text-white"
                style={{ background: confirm.type === 'delete' ? '#EF4444' : ADMIN_ACCENT }}
              >
                {confirm.type === 'dismiss' ? 'Dismiss' : 'Delete Post'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function AdminSelect({ label, value, onChange, options }) {
  return (
    <div>
      <p className="text-[10px] mb-1" style={{ color: '#52525B' }}>{label}</p>
      <select value={value} onChange={(e) => onChange(e.target.value)} className="px-3 py-2 text-sm rounded-btn outline-none cursor-pointer" style={{ background: '#1A1A1A', border: '1px solid #2E2E2E', color: '#A1A1AA' }}>
        {options.map((o) => <option key={o} value={o}>{o}</option>)}
      </select>
    </div>
  )
}

function PageBtn({ label, disabled, onClick }) {
  return (
    <button onClick={onClick} disabled={disabled} className="min-w-[28px] h-7 px-2 rounded text-xs font-medium disabled:opacity-30" style={{ background: '#1A1A1A', color: '#71717A' }}>
      {label}
    </button>
  )
}

function RowSkeleton() {
  return (
    <tr>
      {[48, 100, 80, 80, 80, 120].map((w, i) => (
        <td key={i} className="px-4 py-3.5">
          <div className="skeleton h-3.5 rounded" style={{ width: w }} />
        </td>
      ))}
    </tr>
  )
}


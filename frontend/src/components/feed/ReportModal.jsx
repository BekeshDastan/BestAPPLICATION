import { useState } from 'react'
import { X, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'

const REASONS = ['Spam', 'Nudity', 'Harassment', 'Misinformation', 'Other']

export default function ReportModal({ postId, onClose }) {
  const [reason, setReason]   = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit() {
    if (!reason) return
    setLoading(true)
    try {
      await api.post(`/posts/${postId}/report`, { reason })
      toast.success('Report submitted. Thank you.')
      onClose()
    } catch {
      toast.error('Failed to submit report')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 modal-backdrop"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-sm p-6 animate-fade-in">
        {/* Header */}
        <div className="flex items-center justify-between mb-1">
          <h3 className="font-semibold text-hi">Report post</h3>
          <button
            onClick={onClose}
            className="p-1 rounded-btn text-lo hover:text-hi transition-colors"
          >
            <X size={18} />
          </button>
        </div>
        <p className="text-sm text-lo mb-5">
          Why are you reporting this post?
        </p>

        {/* Radio options */}
        <div className="space-y-2 mb-6">
          {REASONS.map((r) => (
            <label
              key={r}
              className="flex items-center gap-3 cursor-pointer py-1 group"
              onClick={() => setReason(r)}
            >
              {/* Custom radio */}
              <div
                className="w-4 h-4 rounded-full border-2 flex items-center justify-center shrink-0 transition-colors"
                style={{
                  borderColor: reason === r ? 'var(--accent)' : 'var(--border)',
                  background:  reason === r ? 'var(--accent)' : 'transparent',
                }}
              >
                {reason === r && (
                  <div className="w-1.5 h-1.5 rounded-full bg-white" />
                )}
              </div>
              <span
                className="text-sm transition-colors"
                style={{ color: reason === r ? 'var(--text-1)' : 'var(--text-2)' }}
              >
                {r}
              </span>
            </label>
          ))}
        </div>

        <button
          onClick={handleSubmit}
          disabled={!reason || loading}
          className="btn-primary w-full"
        >
          {loading && <Loader2 size={16} className="animate-spin" />}
          {loading ? 'Submitting…' : 'Submit report'}
        </button>
      </div>
    </div>
  )
}

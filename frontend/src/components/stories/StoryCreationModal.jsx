import { useRef, useState } from 'react'
import { X, Upload } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'

export default function StoryCreationModal({ onClose, onCreated }) {
  const [file,      setFile]      = useState(null)
  const [preview,   setPreview]   = useState(null)
  const [caption,   setCaption]   = useState('')
  const [uploading, setUploading] = useState(false)
  const [dragging,  setDragging]  = useState(false)
  const inputRef = useRef(null)

  function handleFile(f) {
    if (!f) return
    setFile(f)
    setPreview(URL.createObjectURL(f))
  }

  function onDrop(e) {
    e.preventDefault()
    setDragging(false)
    const f = e.dataTransfer.files[0]
    if (f) handleFile(f)
  }

  async function submit() {
    if (!file) return
    setUploading(true)
    try {
      const { data: urlData } = await api.get('/media/upload-url', { params: { type: 'story' } })
      await fetch(urlData.upload_url, {
        method: 'PUT',
        body: file,
        headers: { 'Content-Type': file.type },
      })
      await api.post('/stories', {
        media_url:  urlData.media_url,
        media_type: file.type.startsWith('video/') ? 'video' : 'image',
        caption:    caption.trim() || undefined,
      })
      toast.success('Story posted!')
      onCreated?.()
      onClose()
    } catch {
      toast.error('Failed to post story')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 modal-backdrop"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-md overflow-hidden animate-fade-in">
        {/* Header */}
        <div
          className="flex items-center justify-between px-5 py-4 border-b"
          style={{ borderColor: 'var(--border)' }}
        >
          <h2 className="font-semibold text-hi">Create Story</h2>
          <button onClick={onClose} className="text-lo hover:text-hi p-1">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4">
          {/* Drop zone / preview */}
          {!preview ? (
            <div
              onDrop={onDrop}
              onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
              onDragLeave={() => setDragging(false)}
              onClick={() => inputRef.current?.click()}
              className="border-2 border-dashed rounded-card flex flex-col items-center justify-center py-12 cursor-pointer transition-colors"
              style={{
                borderColor: dragging ? 'var(--accent)' : 'var(--border)',
                background:  dragging ? 'var(--accent-glow)' : 'var(--surface-high)',
              }}
            >
              <Upload size={32} className="mb-3" style={{ color: 'var(--text-2)' }} />
              <p className="text-sm text-hi font-medium mb-1">Drag & drop or choose file</p>
              <p className="text-xs text-lo">Image or video</p>
              <input
                ref={inputRef}
                type="file"
                accept="image/*,video/*"
                className="hidden"
                onChange={(e) => handleFile(e.target.files[0])}
              />
            </div>
          ) : (
            <div
              className="relative rounded-card overflow-hidden"
              style={{ aspectRatio: '9/16', maxHeight: '320px' }}
            >
              {file?.type?.startsWith('video/') ? (
                <video src={preview} className="w-full h-full object-cover" controls />
              ) : (
                <img src={preview} alt="" className="w-full h-full object-cover" />
              )}
              <button
                onClick={() => { setFile(null); setPreview(null) }}
                className="absolute top-2 right-2 w-7 h-7 rounded-full bg-black/60 flex items-center justify-center text-white hover:bg-black/80"
              >
                <X size={14} />
              </button>
            </div>
          )}

          {/* Caption */}
          <input
            type="text"
            value={caption}
            onChange={(e) => setCaption(e.target.value)}
            placeholder="Add a caption..."
            className="input-base text-sm"
            maxLength={300}
          />
        </div>

        <div className="flex gap-3 px-5 pb-5">
          <button
            onClick={onClose}
            className="flex-1 py-2 rounded-btn border text-sm text-lo"
            style={{ borderColor: 'var(--border)' }}
          >
            Cancel
          </button>
          <button
            onClick={submit}
            disabled={!file || uploading}
            className="flex-1 btn-primary py-2 text-sm disabled:opacity-40"
          >
            {uploading ? 'Posting...' : 'Post →'}
          </button>
        </div>
      </div>
    </div>
  )
}

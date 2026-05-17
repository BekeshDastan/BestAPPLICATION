import { useRef, useState } from 'react'
import { X, Upload, ChevronLeft, ChevronRight, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import api from '../../lib/api'

export default function PostCreationModal({ onClose, onCreated }) {
  const [files,     setFiles]     = useState([])
  const [previews,  setPreviews]  = useState([])
  const [current,   setCurrent]   = useState(0)
  const [caption,   setCaption]   = useState('')
  const [tags,      setTags]      = useState('')
  const [uploading, setUploading] = useState(false)
  const [dragging,  setDragging]  = useState(false)
  const inputRef = useRef(null)

  function addFiles(incoming) {
    const arr = Array.from(incoming)
    const merged = [...files, ...arr].slice(0, 10)
    setFiles(merged)
    setPreviews(merged.map((f) => URL.createObjectURL(f)))
    setCurrent(0)
  }

  function onDrop(e) {
    e.preventDefault()
    setDragging(false)
    addFiles(e.dataTransfer.files)
  }

  function removeFile(idx) {
    const next = files.filter((_, i) => i !== idx)
    setFiles(next)
    setPreviews(next.map((f) => URL.createObjectURL(f)))
    setCurrent(Math.min(current, Math.max(0, next.length - 1)))
  }

  async function submit() {
    if (!files.length) return
    setUploading(true)
    try {
      const mediaUrls = await Promise.all(
        files.map(async (file) => {
          const { data: u } = await api.get('/media/upload-url', { params: { type: 'post' } })
          await fetch(u.upload_url, {
            method: 'PUT', body: file, headers: { 'Content-Type': file.type },
          })
          return u.media_url
        }),
      )
      const tagList = tags.split(/[\s,]+/).map((t) => t.replace(/^#/, '').trim()).filter(Boolean)
      await api.post('/posts', {
        caption:    caption.trim() || undefined,
        media_urls: mediaUrls,
        tags:       tagList.length ? tagList : undefined,
      })
      toast.success('Post created!')
      onCreated?.()
      onClose()
    } catch {
      toast.error('Failed to create post')
    } finally {
      setUploading(false)
    }
  }

  const canSubmit = files.length > 0 && !uploading

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 modal-backdrop"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-lg flex flex-col animate-fade-in" style={{ maxHeight: '90vh' }}>
        {/* Header */}
        <div
          className="flex items-center justify-between px-5 py-4 border-b"
          style={{ borderColor: 'var(--border)' }}
        >
          <h2 className="font-semibold text-hi">New Post</h2>
          <button onClick={onClose} className="text-lo hover:text-hi p-1 transition-colors">
            <X size={18} />
          </button>
        </div>

        <div className="p-5 space-y-4 overflow-y-auto flex-1">
          {/* Media area */}
          {previews.length === 0 ? (
            <div
              onDrop={onDrop}
              onDragOver={(e) => { e.preventDefault(); setDragging(true) }}
              onDragLeave={() => setDragging(false)}
              onClick={() => inputRef.current?.click()}
              className="border-2 border-dashed rounded-card flex flex-col items-center justify-center py-14 cursor-pointer transition-colors"
              style={{
                borderColor: dragging ? 'var(--accent)' : 'var(--border)',
                background:  dragging ? 'var(--accent-glow)' : 'var(--surface-high)',
              }}
            >
              <Upload size={32} className="mb-3" style={{ color: 'var(--text-2)' }} />
              <p className="text-sm font-medium text-hi mb-1">Drag & drop or click to upload</p>
              <p className="text-xs text-lo">Up to 10 images or videos</p>
              <input
                ref={inputRef}
                type="file"
                accept="image/*,video/*"
                multiple
                className="hidden"
                onChange={(e) => addFiles(e.target.files)}
              />
            </div>
          ) : (
            <div className="space-y-2">
              {/* Carousel */}
              <div className="relative rounded-card overflow-hidden bg-black" style={{ aspectRatio: '1 / 1' }}>
                {files[current]?.type?.startsWith('video/') ? (
                  <video
                    src={previews[current]}
                    className="w-full h-full object-contain"
                    controls
                  />
                ) : (
                  <img
                    src={previews[current]}
                    alt=""
                    className="w-full h-full object-contain"
                  />
                )}
                {/* Prev / Next */}
                {previews.length > 1 && (
                  <>
                    <button
                      onClick={() => setCurrent((v) => Math.max(0, v - 1))}
                      disabled={current === 0}
                      className="absolute left-2 top-1/2 -translate-y-1/2 w-8 h-8 rounded-full bg-black/60 flex items-center justify-center text-white disabled:opacity-30"
                    >
                      <ChevronLeft size={18} />
                    </button>
                    <button
                      onClick={() => setCurrent((v) => Math.min(previews.length - 1, v + 1))}
                      disabled={current === previews.length - 1}
                      className="absolute right-2 top-1/2 -translate-y-1/2 w-8 h-8 rounded-full bg-black/60 flex items-center justify-center text-white disabled:opacity-30"
                    >
                      <ChevronRight size={18} />
                    </button>
                    {/* Dots */}
                    <div className="absolute bottom-2 left-1/2 -translate-x-1/2 flex gap-1">
                      {previews.map((_, i) => (
                        <button
                          key={i}
                          onClick={() => setCurrent(i)}
                          className="w-1.5 h-1.5 rounded-full transition-colors"
                          style={{ background: i === current ? 'white' : 'rgba(255,255,255,0.4)' }}
                        />
                      ))}
                    </div>
                  </>
                )}
                {/* Remove current */}
                <button
                  onClick={() => removeFile(current)}
                  className="absolute top-2 right-2 w-7 h-7 rounded-full bg-black/60 flex items-center justify-center text-white hover:bg-black/80"
                >
                  <X size={14} />
                </button>
              </div>

              {/* Thumbnails + add more */}
              <div className="flex gap-2 overflow-x-auto pb-1">
                {previews.map((src, i) => (
                  <button
                    key={i}
                    onClick={() => setCurrent(i)}
                    className="relative shrink-0 w-14 h-14 rounded-btn overflow-hidden border-2 transition-all"
                    style={{ borderColor: i === current ? 'var(--accent)' : 'transparent' }}
                  >
                    <img src={src} alt="" className="w-full h-full object-cover" />
                  </button>
                ))}
                {files.length < 10 && (
                  <button
                    onClick={() => inputRef.current?.click()}
                    className="shrink-0 w-14 h-14 rounded-btn border-2 border-dashed flex items-center justify-center transition-colors hover:border-accent"
                    style={{ borderColor: 'var(--border)' }}
                  >
                    <Upload size={16} style={{ color: 'var(--text-2)' }} />
                  </button>
                )}
                <input
                  ref={inputRef}
                  type="file"
                  accept="image/*,video/*"
                  multiple
                  className="hidden"
                  onChange={(e) => addFiles(e.target.files)}
                />
              </div>
            </div>
          )}

          {/* Caption */}
          <div>
            <textarea
              value={caption}
              onChange={(e) => setCaption(e.target.value)}
              maxLength={2200}
              rows={3}
              placeholder="Write a caption..."
              className="input-base text-sm resize-none"
            />
            <p className="text-right text-[11px] text-lo mt-1">{caption.length}/2200</p>
          </div>

          {/* Tags */}
          <input
            type="text"
            value={tags}
            onChange={(e) => setTags(e.target.value)}
            placeholder="Tags: nature, travel, food  (space or comma separated)"
            className="input-base text-sm"
          />
        </div>

        {/* Footer */}
        <div className="flex gap-3 px-5 pb-5">
          <button
            onClick={onClose}
            className="flex-1 py-2 rounded-btn border text-sm text-lo transition-colors hover:text-hi"
            style={{ borderColor: 'var(--border)' }}
          >
            Cancel
          </button>
          <button
            onClick={submit}
            disabled={!canSubmit}
            className="flex-1 btn-primary py-2 text-sm disabled:opacity-40 flex items-center justify-center gap-2"
          >
            {uploading && <Loader2 size={14} className="animate-spin" />}
            {uploading ? 'Posting...' : 'Share Post'}
          </button>
        </div>
      </div>
    </div>
  )
}

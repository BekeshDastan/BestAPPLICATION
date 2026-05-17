import { useState } from 'react'
import { toast } from 'sonner'
import api from '../lib/api'

const MAX_BYTES = 50 * 1024 * 1024 // 50 MB

export default function useMediaUpload() {
  const [progress,  setProgress]  = useState(0)
  const [uploading, setUploading] = useState(false)

  async function uploadMedia(file, type) {
    if (file.size > MAX_BYTES) {
      toast.error('File too large (max 50 MB)')
      return null
    }

    setUploading(true)
    setProgress(0)

    try {
      // 1. Get presigned upload URL from the gateway
      const { data } = await api.get('/media/upload-url', {
        params: { type, filename: file.name, content_type: file.type },
      })
      const { upload_url, media_url } = data

      // 2. PUT directly to MinIO — use XHR so we get upload progress events
      await new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest()
        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) setProgress(Math.round((e.loaded / e.total) * 100))
        }
        xhr.onload  = () => xhr.status >= 200 && xhr.status < 300 ? resolve() : reject()
        xhr.onerror = () => reject()
        xhr.open('PUT', upload_url)
        xhr.setRequestHeader('Content-Type', file.type)
        xhr.send(file)
      })

      setProgress(100)
      return media_url
    } catch {
      toast.error('Upload failed')
      return null
    } finally {
      setUploading(false)
    }
  }

  return { uploadMedia, progress, uploading }
}

import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import api from '../../lib/api'
import PostCard, { PostSkeleton } from '../../components/feed/PostCard'

export default function PostPage() {
  const { id } = useParams()
  const navigate = useNavigate()

  const [post,    setPost]    = useState(null)
  const [loading, setLoading] = useState(true)
  const [error,   setError]   = useState(false)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError(false)
    api.get(`/posts/${id}`)
      .then(({ data }) => {
        // gateway returns { post: {...} } from grpc CreatePost/GetPost response
        setPost(data.post ?? data)
      })
      .catch(() => setError(true))
      .finally(() => setLoading(false))
  }, [id])

  return (
    <div className="max-w-2xl mx-auto px-4 py-6">
      <button
        onClick={() => navigate(-1)}
        className="flex items-center gap-1.5 text-sm text-lo hover:text-hi transition-colors mb-4"
      >
        <ArrowLeft size={16} /> Back
      </button>

      {loading && <PostSkeleton />}

      {!loading && error && (
        <div className="card p-12 text-center">
          <p className="font-semibold text-hi mb-2">Post not found</p>
          <p className="text-sm text-lo mb-5">
            This post may have been deleted or the link is invalid.
          </p>
          <Link to="/" className="btn-primary inline-flex">Go home</Link>
        </div>
      )}

      {!loading && !error && post && (
        <PostCard post={post} onDelete={() => navigate('/')} />
      )}
    </div>
  )
}

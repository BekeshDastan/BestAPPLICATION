import { Heart, MessageCircle, FileText } from 'lucide-react'
import { formatCount } from '../../lib/utils'

export default function PostThumbnail({ post, onClick }) {
  const hasMedia = post.media_urls?.length > 0
  const isVideo  = hasMedia && /\.(mp4|webm|ogg)$/i.test(post.media_urls[0])

  return (
    <button
      onClick={() => onClick(post)}
      className="relative group w-full overflow-hidden rounded-card block text-left"
      style={{ background: 'var(--surface-high)' }}
    >
      {hasMedia ? (
        isVideo ? (
          <video
            src={post.media_urls[0]}
            className="w-full aspect-square object-cover"
            muted
            preload="metadata"
          />
        ) : (
          <img
            src={post.media_urls[0]}
            alt=""
            className="w-full aspect-square object-cover"
            loading="lazy"
          />
        )
      ) : (
        /* Text-only post */
        <div
          className="aspect-square flex flex-col items-center justify-center p-4 gap-2"
        >
          <FileText size={20} style={{ color: 'var(--text-2)' }} />
          <p className="text-xs text-lo line-clamp-4 text-center leading-relaxed">
            {post.caption}
          </p>
        </div>
      )}

      {/* Hover overlay */}
      <div className="absolute inset-0 bg-black/55 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex items-center justify-center gap-5">
        <span className="flex items-center gap-1.5 text-white text-sm font-semibold drop-shadow">
          <Heart size={16} fill="white" />
          {formatCount(post.likes_count ?? 0)}
        </span>
        <span className="flex items-center gap-1.5 text-white text-sm font-semibold drop-shadow">
          <MessageCircle size={16} fill="white" />
          {formatCount(post.comments_count ?? 0)}
        </span>
      </div>

      {/* Video indicator */}
      {isVideo && (
        <div className="absolute top-2 right-2 opacity-80">
          <div className="w-5 h-5 rounded-full bg-black/60 flex items-center justify-center">
            <span className="text-white text-[8px]">▶</span>
          </div>
        </div>
      )}
    </button>
  )
}

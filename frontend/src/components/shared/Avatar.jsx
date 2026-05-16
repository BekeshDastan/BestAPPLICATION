export default function Avatar({ src, name, size = 36, className = '' }) {
  const initials = name
    ? name.split(' ').map((w) => w[0]).slice(0, 2).join('').toUpperCase()
    : '?'

  if (src) {
    return (
      <img
        src={src}
        alt={name ?? 'avatar'}
        loading="lazy"
        className={`rounded-full object-cover shrink-0 ${className}`}
        style={{ width: size, height: size }}
      />
    )
  }

  return (
    <div
      className={`rounded-full flex items-center justify-center font-semibold text-white shrink-0 select-none ${className}`}
      style={{
        width: size,
        height: size,
        background: 'var(--accent)',
        fontSize: Math.max(Math.round(size * 0.38), 10),
      }}
    >
      {initials}
    </div>
  )
}

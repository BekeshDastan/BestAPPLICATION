import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs) {
  return twMerge(clsx(inputs))
}

export function formatRelativeTime(timestamp) {
  const now = Date.now()
  const ts = typeof timestamp === 'number' ? timestamp * 1000 : new Date(timestamp).getTime()
  const diff = now - ts
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}d ago`
  return new Date(ts).toLocaleDateString()
}

export function formatCount(n) {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}

export function getPasswordStrength(password) {
  let score = 0
  if (password.length >= 8)  score++
  if (password.length >= 12) score++
  if (/[A-Z]/.test(password)) score++
  if (/[0-9]/.test(password)) score++
  if (/[^A-Za-z0-9]/.test(password)) score++
  if (score <= 1) return { level: 'Weak',   color: '#EF4444', width: '20%' }
  if (score === 2) return { level: 'Fair',   color: '#F59E0B', width: '40%' }
  if (score === 3) return { level: 'Good',   color: '#EAB308', width: '60%' }
  if (score === 4) return { level: 'Strong', color: '#22C55E', width: '80%' }
  return { level: 'Very strong', color: '#10B981', width: '100%' }
}

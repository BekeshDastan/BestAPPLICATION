import { useState, useEffect, useRef } from 'react'
import { RefreshCw } from 'lucide-react'
import api from '../../lib/api'
import { formatRelativeTime } from '../../lib/utils'
import { ADMIN_ACCENT } from './AdminLayout'

const SERVICES = [
  { key: 'user',         name: 'User Service',          port: '50051',       group: 'services' },
  { key: 'post',         name: 'Post Service',          port: '50052',       group: 'services' },
  { key: 'chat',         name: 'Chat Service',          port: '50053',       group: 'services' },
  { key: 'story',        name: 'Story Service',         port: '50054',       group: 'services' },
  { key: 'notification', name: 'Notification Service',  port: '50055',       group: 'services' },
  { key: 'gateway',      name: 'API Gateway',           port: '8080',        group: 'services' },
  { key: 'postgres',     name: 'PostgreSQL',            port: '5432',        group: 'infra' },
  { key: 'redis',        name: 'Redis',                 port: '6379',        group: 'infra' },
  { key: 'nats',         name: 'NATS',                  port: '4222 / 8222', group: 'infra' },
  { key: 'minio',        name: 'MinIO',                 port: '9000 / 9001', group: 'infra' },
  { key: 'mailhog',      name: 'MailHog',               port: '1025 / 8025', group: 'infra' },
]

const STATUS_CONFIG = {
  healthy:  { label: 'Healthy',  color: '#10B981' },
  degraded: { label: 'Degraded', color: '#F59E0B' },
  down:     { label: 'Down',     color: '#EF4444' },
  unknown:  { label: 'Unknown',  color: '#71717A' },
}

export default function AdminSystem() {
  const [health,      setHealth]      = useState({})
  const [loading,     setLoading]     = useState(true)
  const [lastChecked, setLastChecked] = useState(null)
  const [refreshing,  setRefreshing]  = useState(false)
  const timerRef = useRef(null)

  async function fetchHealth(manual = false) {
    if (manual) setRefreshing(true)
    try {
      const { data } = await api.get('/admin/system/health')
      // Normalize: data may be { services: { user: 'healthy', ... } } or { services: [...] }
      const raw = data.services ?? data
      if (Array.isArray(raw)) {
        const map = {}
        raw.forEach((s) => { map[s.key ?? s.name?.toLowerCase()] = s.status ?? 'unknown' })
        setHealth(map)
      } else {
        setHealth(raw)
      }
    } catch {
      // If endpoint doesn't exist, mark all as unknown
      const fallback = {}
      SERVICES.forEach((s) => { fallback[s.key] = 'unknown' })
      setHealth(fallback)
    } finally {
      setLoading(false)
      setRefreshing(false)
      setLastChecked(new Date())
    }
  }

  useEffect(() => {
    fetchHealth()
    timerRef.current = setInterval(() => fetchHealth(), 15000)
    return () => clearInterval(timerRef.current)
  }, [])

  const appServices  = SERVICES.filter((s) => s.group === 'services')
  const infraServices = SERVICES.filter((s) => s.group === 'infra')

  const allHealthy  = SERVICES.every((s) => (health[s.key] ?? 'unknown') === 'healthy')
  const anyDown     = SERVICES.some((s) => health[s.key] === 'down')

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-white">System Health</h1>
          {lastChecked && (
            <p className="text-xs mt-0.5" style={{ color: '#52525B' }}>
              Last checked {formatRelativeTime(lastChecked.toISOString())}
            </p>
          )}
        </div>
        <div className="flex items-center gap-3">
          {/* Overall status pill */}
          <span
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium"
            style={{
              background: anyDown ? 'rgba(239,68,68,0.15)' : allHealthy ? `${ADMIN_ACCENT}20` : 'rgba(245,158,11,0.15)',
              color:      anyDown ? '#EF4444' : allHealthy ? ADMIN_ACCENT : '#F59E0B',
            }}
          >
            <span
              className="w-1.5 h-1.5 rounded-full"
              style={{ background: anyDown ? '#EF4444' : allHealthy ? ADMIN_ACCENT : '#F59E0B' }}
            />
            {anyDown ? 'Outage Detected' : allHealthy ? 'All Systems Operational' : 'Degraded Performance'}
          </span>

          <button
            onClick={() => fetchHealth(true)}
            disabled={refreshing}
            className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-btn transition-colors disabled:opacity-50"
            style={{ background: '#1A1A1A', color: '#71717A' }}
            onMouseEnter={(e) => (e.currentTarget.style.color = '#fff')}
            onMouseLeave={(e) => (e.currentTarget.style.color = '#71717A')}
          >
            <RefreshCw size={13} className={refreshing ? 'animate-spin' : ''} />
            Refresh All
          </button>
        </div>
      </div>

      {/* App Services */}
      <div>
        <p className="text-xs font-semibold uppercase tracking-wider mb-3" style={{ color: '#52525B' }}>
          Application Services
        </p>
        <div className="grid grid-cols-3 gap-3">
          {appServices.map((svc) => (
            <ServiceCard
              key={svc.key}
              service={svc}
              status={health[svc.key] ?? (loading ? null : 'unknown')}
              lastChecked={lastChecked}
            />
          ))}
        </div>
      </div>

      {/* Infrastructure */}
      <div>
        <p className="text-xs font-semibold uppercase tracking-wider mb-3" style={{ color: '#52525B' }}>
          Infrastructure
        </p>
        <div className="grid grid-cols-3 gap-3">
          {infraServices.map((svc) => (
            <ServiceCard
              key={svc.key}
              service={svc}
              status={health[svc.key] ?? (loading ? null : 'unknown')}
              lastChecked={lastChecked}
            />
          ))}
        </div>
      </div>

      {/* Legend */}
      <div
        className="rounded-card p-4 flex items-center gap-6"
        style={{ background: '#161616', border: '1px solid #1F1F1F' }}
      >
        <p className="text-xs font-medium" style={{ color: '#52525B' }}>Legend:</p>
        {Object.entries(STATUS_CONFIG).map(([, { label, color }]) => (
          <div key={label} className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full" style={{ background: color }} />
            <span className="text-xs" style={{ color: '#A1A1AA' }}>{label}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

function ServiceCard({ service, status, lastChecked }) {
  const cfg = STATUS_CONFIG[status] ?? STATUS_CONFIG.unknown
  const loading = status === null

  return (
    <div
      className="rounded-card p-4 flex flex-col gap-3 transition-all"
      style={{
        background: '#161616',
        border: `1px solid ${status === 'down' ? 'rgba(239,68,68,0.3)' : status === 'degraded' ? 'rgba(245,158,11,0.3)' : '#1F1F1F'}`,
      }}
    >
      <div className="flex items-start justify-between">
        <p className="text-sm font-semibold text-white leading-tight">{service.name}</p>
        {loading ? (
          <div className="skeleton w-2.5 h-2.5 rounded-full" />
        ) : (
          <span
            className="w-2.5 h-2.5 rounded-full shrink-0 mt-0.5"
            style={{
              background: cfg.color,
              boxShadow: status === 'healthy' ? `0 0 6px ${cfg.color}80` : 'none',
            }}
          />
        )}
      </div>

      <div className="space-y-1">
        <div className="flex items-center gap-2">
          <span className="text-[10px]" style={{ color: '#52525B' }}>Port</span>
          <span
            className="text-[10px] font-mono px-1.5 py-0.5 rounded"
            style={{ background: '#1A1A1A', color: '#A1A1AA' }}
          >
            {service.port}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-[10px]" style={{ color: '#52525B' }}>Status</span>
          {loading ? (
            <div className="skeleton h-3 w-16 rounded" />
          ) : (
            <span className="text-[10px] font-medium" style={{ color: cfg.color }}>
              {cfg.label}
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span className="text-[10px]" style={{ color: '#52525B' }}>Checked</span>
          <span className="text-[10px]" style={{ color: '#52525B' }}>
            {lastChecked ? formatRelativeTime(lastChecked.toISOString()) : '—'}
          </span>
        </div>
      </div>
    </div>
  )
}

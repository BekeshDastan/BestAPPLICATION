import { useEffect, useState, useRef } from 'react'
import { Users, FileImage, BookOpen, MessageSquare, RefreshCw } from 'lucide-react'
import {
  LineChart, Line, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from 'recharts'
import api from '../../lib/api'
import Avatar from '../../components/shared/Avatar'
import { formatRelativeTime, formatCount } from '../../lib/utils'
import { ADMIN_ACCENT } from './AdminLayout'

const ACTIVITY_ICONS = {
  registered:     '👤',
  posted:         '📸',
  followed:       '❤️',
  story_created:  '📖',
  reported:       '🚩',
}

export default function AdminDashboard() {
  const [stats,      setStats]      = useState(null)
  const [regData,    setRegData]    = useState([])
  const [postData,   setPostData]   = useState([])
  const [topUsers,   setTopUsers]   = useState([])
  const [activity,   setActivity]   = useState([])
  const [loading,    setLoading]    = useState(true)
  const activityRef = useRef(null)

  async function fetchAll() {
    try {
      const [sRes, rRes, pRes, uRes, aRes] = await Promise.allSettled([
        api.get('/admin/stats'),
        api.get('/admin/stats/registrations'),
        api.get('/admin/stats/posts'),
        api.get('/admin/users/top'),
        api.get('/admin/activity'),
      ])
      if (sRes.status === 'fulfilled') setStats(sRes.value.data)
      if (rRes.status === 'fulfilled') setRegData(rRes.value.data.data ?? [])
      if (pRes.status === 'fulfilled') setPostData(pRes.value.data.data ?? [])
      if (uRes.status === 'fulfilled') setTopUsers(uRes.value.data.users ?? [])
      if (aRes.status === 'fulfilled') setActivity(aRes.value.data.events ?? [])
    } catch {}
    setLoading(false)
  }

  useEffect(() => {
    fetchAll()
    activityRef.current = setInterval(async () => {
      try {
        const { data } = await api.get('/admin/activity')
        setActivity(data.events ?? [])
      } catch {}
    }, 10000)
    return () => clearInterval(activityRef.current)
  }, [])

  const statCards = [
    { label: 'Total Users',    value: stats?.total_users,    sub: stats?.user_growth,    icon: Users,        color: ADMIN_ACCENT },
    { label: 'Total Posts',    value: stats?.total_posts,    sub: stats?.post_growth,    icon: FileImage,    color: '#6366F1' },
    { label: 'Active Stories', value: stats?.active_stories, sub: 'right now',           icon: BookOpen,     color: '#F59E0B' },
    { label: 'Messages Today', value: stats?.messages_today, sub: 'today',               icon: MessageSquare, color: '#EC4899' },
  ]

  const chartTooltipStyle = {
    contentStyle: { background: '#1A1A1A', border: '1px solid #2E2E2E', borderRadius: 8, color: '#fff' },
    labelStyle:   { color: '#A1A1AA', fontSize: 11 },
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-white">Dashboard</h1>
        <button
          onClick={fetchAll}
          className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-btn transition-colors"
          style={{ color: '#71717A', background: '#1A1A1A' }}
          onMouseEnter={(e) => (e.currentTarget.style.color = '#fff')}
          onMouseLeave={(e) => (e.currentTarget.style.color = '#71717A')}
        >
          <RefreshCw size={13} /> Refresh
        </button>
      </div>

      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-4">
        {statCards.map(({ label, value, sub, icon: Icon, color }) => (
          <div
            key={label}
            className="rounded-card p-5 flex flex-col gap-3"
            style={{ background: '#161616', border: '1px solid #1F1F1F' }}
          >
            <div className="flex items-start justify-between">
              <p className="text-xs font-medium" style={{ color: '#71717A' }}>{label}</p>
              <div
                className="w-8 h-8 rounded-btn flex items-center justify-center"
                style={{ background: `${color}20` }}
              >
                <Icon size={16} style={{ color }} />
              </div>
            </div>
            <div>
              {loading ? (
                <div className="skeleton h-7 w-24 rounded" />
              ) : (
                <p className="text-2xl font-bold text-white">
                  {value !== undefined ? formatCount(value) : '—'}
                </p>
              )}
              {sub && !loading && (
                <p className="text-xs mt-0.5" style={{ color: ADMIN_ACCENT }}>
                  {typeof sub === 'number' ? `↑ ${sub}% /wk` : sub}
                </p>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-2 gap-4">
        <ChartCard title="New Registrations / Day">
          {loading ? <ChartSkeleton /> : regData.length === 0 ? <NoData /> : (
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={regData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1F1F1F" />
                <XAxis dataKey="date" tick={{ fill: '#71717A', fontSize: 10 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fill: '#71717A', fontSize: 10 }} axisLine={false} tickLine={false} />
                <Tooltip {...chartTooltipStyle} />
                <Line
                  type="monotone"
                  dataKey="count"
                  stroke={ADMIN_ACCENT}
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4, fill: ADMIN_ACCENT }}
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </ChartCard>

        <ChartCard title="Posts Created / Day">
          {loading ? <ChartSkeleton /> : postData.length === 0 ? <NoData /> : (
            <ResponsiveContainer width="100%" height={220}>
              <BarChart data={postData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#1F1F1F" />
                <XAxis dataKey="date" tick={{ fill: '#71717A', fontSize: 10 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fill: '#71717A', fontSize: 10 }} axisLine={false} tickLine={false} />
                <Tooltip {...chartTooltipStyle} />
                <Bar dataKey="count" fill={ADMIN_ACCENT} radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </ChartCard>
      </div>

      {/* Bottom row */}
      <div className="grid grid-cols-2 gap-4">
        {/* Top Users */}
        <div
          className="rounded-card overflow-hidden"
          style={{ background: '#161616', border: '1px solid #1F1F1F' }}
        >
          <div className="px-5 py-4 border-b" style={{ borderColor: '#1F1F1F' }}>
            <h2 className="text-sm font-semibold text-white">Top Active Users</h2>
          </div>
          <div className="overflow-x-auto">
            {loading ? (
              <div className="p-4 space-y-3">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <div className="skeleton w-6 h-3 rounded" />
                    <div className="skeleton w-7 h-7 rounded-full" />
                    <div className="skeleton h-3 w-24 rounded" />
                  </div>
                ))}
              </div>
            ) : topUsers.length === 0 ? <NoData /> : (
              <table className="w-full text-xs">
                <thead>
                  <tr style={{ borderBottom: '1px solid #1F1F1F' }}>
                    {['#', '', 'Username', 'Posts', 'Followers', 'Joined'].map((h) => (
                      <th
                        key={h}
                        className="px-4 py-2.5 text-left font-medium"
                        style={{ color: '#52525B' }}
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {topUsers.slice(0, 10).map((u, idx) => (
                    <tr key={u.id} style={{ borderBottom: idx < topUsers.length - 1 ? '1px solid #1F1F1F' : 'none' }}>
                      <td className="px-4 py-2.5" style={{ color: '#52525B' }}>{idx + 1}</td>
                      <td className="px-2 py-2.5">
                        <Avatar src={u.avatar_url} name={u.username} size={28} />
                      </td>
                      <td className="px-2 py-2.5 text-white font-medium">@{u.username}</td>
                      <td className="px-4 py-2.5" style={{ color: '#A1A1AA' }}>{formatCount(u.posts_count ?? 0)}</td>
                      <td className="px-4 py-2.5" style={{ color: '#A1A1AA' }}>{formatCount(u.followers_count ?? 0)}</td>
                      <td className="px-4 py-2.5" style={{ color: '#52525B' }}>
                        {u.created_at ? new Date(u.created_at).toLocaleDateString('en', { month: 'short', year: '2-digit' }) : '—'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </div>

        {/* Activity Feed */}
        <div
          className="rounded-card overflow-hidden flex flex-col"
          style={{ background: '#161616', border: '1px solid #1F1F1F' }}
        >
          <div className="px-5 py-4 border-b flex items-center justify-between" style={{ borderColor: '#1F1F1F' }}>
            <h2 className="text-sm font-semibold text-white">Recent Activity</h2>
            <span className="text-[10px] px-1.5 py-0.5 rounded-full" style={{ background: `${ADMIN_ACCENT}20`, color: ADMIN_ACCENT }}>
              Live
            </span>
          </div>
          <div className="flex-1 overflow-y-auto" style={{ maxHeight: 320 }}>
            {loading ? (
              <div className="p-4 space-y-3">
                {Array.from({ length: 6 }).map((_, i) => (
                  <div key={i} className="flex items-center gap-3">
                    <div className="skeleton w-6 h-6 rounded-full" />
                    <div className="skeleton h-3 w-48 rounded" />
                  </div>
                ))}
              </div>
            ) : activity.length === 0 ? <NoData /> : (
              <div>
                {activity.map((e, idx) => (
                  <div
                    key={e.id ?? idx}
                    className="flex items-start gap-3 px-4 py-3"
                    style={{ borderBottom: idx < activity.length - 1 ? '1px solid #1F1F1F' : 'none' }}
                  >
                    <span className="text-base shrink-0">{ACTIVITY_ICONS[e.type] ?? '🔔'}</span>
                    <div className="flex-1 min-w-0">
                      <p className="text-xs text-white leading-snug truncate">
                        {e.description ?? e.message ?? e.type}
                      </p>
                      <p className="text-[10px] mt-0.5" style={{ color: '#52525B' }}>
                        {formatRelativeTime(e.created_at)}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function ChartCard({ title, children }) {
  return (
    <div
      className="rounded-card p-5"
      style={{ background: '#161616', border: '1px solid #1F1F1F' }}
    >
      <h2 className="text-sm font-semibold text-white mb-4">{title}</h2>
      {children}
    </div>
  )
}

function ChartSkeleton() {
  return <div className="skeleton rounded w-full h-[220px]" />
}

function NoData() {
  return (
    <div className="flex items-center justify-center py-12">
      <p className="text-xs" style={{ color: '#52525B' }}>No data yet.</p>
    </div>
  )
}

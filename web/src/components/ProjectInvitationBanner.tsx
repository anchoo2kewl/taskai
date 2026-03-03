import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { apiClient } from '../lib/api'

export default function NotificationBell() {
  const [count, setCount] = useState(0)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const navigate = useNavigate()

  const fetchCount = async () => {
    try {
      const { count: c } = await apiClient.getMyProjectInvitationCount()
      setCount(c)
    } catch {
      // ignore
    }
  }

  useEffect(() => {
    fetchCount()
    intervalRef.current = setInterval(fetchCount, 60_000)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  return (
    <button
      onClick={() => navigate('/app/settings')}
      className="relative p-1.5 text-dark-text-tertiary hover:text-dark-text-primary hover:bg-dark-bg-tertiary rounded-md transition-colors"
      aria-label={count > 0 ? `${count} pending project invitation${count !== 1 ? 's' : ''}` : 'Notifications'}
      title={count > 0 ? `${count} pending project invitation${count !== 1 ? 's' : ''} — click to view` : 'Notifications'}
    >
      <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
      </svg>
      {count > 0 && (
        <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 px-0.5 bg-danger-500 text-white text-[10px] font-bold rounded-full flex items-center justify-center leading-none">
          {count > 9 ? '9+' : count}
        </span>
      )}
    </button>
  )
}

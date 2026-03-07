import { useState, useEffect } from 'react'
import { apiClient, type FigmaEmbedInfo } from '../lib/api'

const SIZES: Record<string, string> = {
  s: '400px',
  m: '520px',
  l: '720px',
}

interface FigmaEmbedProps {
  url: string
  size?: 's' | 'm' | 'l'
}

export default function FigmaEmbed({ url, size = 'm' }: FigmaEmbedProps) {
  const [info, setInfo] = useState<FigmaEmbedInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [showIframe, setShowIframe] = useState(false)
  const height = SIZES[size] || SIZES.m

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setInfo(null)
    setShowIframe(false)
    apiClient.getFigmaEmbed(url).then((data) => {
      if (!cancelled) {
        setInfo(data)
        setLoading(false)
      }
    }).catch(() => {
      if (!cancelled) setLoading(false)
    })
    return () => { cancelled = true }
  }, [url])

  if (loading) {
    return (
      <div
        className="rounded-lg border border-dark-border-subtle bg-dark-bg-secondary animate-pulse"
        style={{ height: '80px' }}
        aria-label="Loading Figma embed"
      />
    )
  }

  if (!info) return null

  return (
    <div className="my-3 rounded-lg border border-dark-border-subtle overflow-hidden bg-dark-bg-secondary">
      {/* Header row */}
      <div className="flex items-center gap-3 px-3 py-2 border-b border-dark-border-subtle">
        {/* Figma logo icon */}
        <svg width="16" height="16" viewBox="0 0 38 57" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true" className="flex-shrink-0">
          <path d="M19 28.5C19 25.8261 20.0536 23.2609 21.9289 21.3856C23.8043 19.5102 26.3696 18.4566 29.0435 18.4566C31.7174 18.4566 34.2826 19.5102 36.158 21.3856C38.0334 23.2609 39.087 25.8261 39.087 28.5C39.087 31.1739 38.0334 33.7391 36.158 35.6145C34.2826 37.4898 31.7174 38.5435 29.0435 38.5435C26.3696 38.5435 23.8043 37.4898 21.9289 35.6145C20.0536 33.7391 19 31.1739 19 28.5Z" fill="#1ABCFE"/>
          <path d="M-1.08691 47.5435C-1.08691 44.8696 -0.0333414 42.3043 1.84197 40.429C3.71727 38.5536 6.28255 37.5 8.95647 37.5H19.0001V47.5435C19.0001 50.2174 17.9465 52.7827 16.0711 54.658C14.1958 56.5334 11.6305 57.587 8.95647 57.587C6.28255 57.587 3.71727 56.5334 1.84197 54.658C-0.0333414 52.7827 -1.08691 50.2174 -1.08691 47.5435Z" fill="#0ACF83"/>
          <path d="M19 0.413086V18.4566H29.0435C31.7174 18.4566 34.2826 17.403 36.158 15.5277C38.0334 13.6523 39.087 11.0871 39.087 8.41309C39.087 5.73916 38.0334 3.17388 36.158 1.29858C34.2826 -0.576719 31.7174 -1.63027 29.0435 -1.63027H19V0.413086Z" fill="#FF7262"/>
          <path d="M-1.08691 8.41309C-1.08691 11.0871 -0.0333414 13.6523 1.84197 15.5277C3.71727 17.403 6.28255 18.4566 8.95647 18.4566H19.0001V-1.58691H8.95647C6.28255 -1.58691 3.71727 -0.533347 1.84197 1.34195C-0.0333414 3.21725 -1.08691 5.78256 -1.08691 8.41309Z" fill="#F24E1E"/>
          <path d="M-1.08691 28.5C-1.08691 31.1739 -0.0333414 33.7391 1.84197 35.6145C3.71727 37.4898 6.28255 38.5435 8.95647 38.5435H19.0001V18.4566H8.95647C6.28255 18.4566 3.71727 19.5102 1.84197 21.3856C-0.0333414 23.2609 -1.08691 25.8261 -1.08691 28.5Z" fill="#A259FF"/>
        </svg>

        {info.name ? (
          <span className="text-sm font-medium text-dark-text-primary truncate flex-1">{info.name}</span>
        ) : (
          <span className="text-sm text-dark-text-tertiary truncate flex-1">Figma design</span>
        )}

        <div className="flex items-center gap-2 flex-shrink-0">
          <a
            href={url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-primary-400 hover:text-primary-300 transition-colors"
          >
            Open in Figma
          </a>
          <button
            onClick={() => setShowIframe(prev => !prev)}
            className="text-xs px-2 py-1 rounded bg-dark-bg-tertiary text-dark-text-secondary hover:text-dark-text-primary transition-colors"
          >
            {showIframe ? 'Hide' : 'View embed'}
          </button>
        </div>
      </div>

      {/* Thumbnail or iframe */}
      {showIframe ? (
        <iframe
          src={info.embed_url}
          style={{ width: '100%', height, border: 'none', display: 'block' }}
          allow="fullscreen"
          referrerPolicy="strict-origin-when-cross-origin"
          title={info.name ?? 'Figma design'}
        />
      ) : info.thumbnail_url ? (
        <div className="relative">
          <img
            src={info.thumbnail_url}
            alt={info.name ?? 'Figma preview'}
            className="w-full object-cover"
            style={{ maxHeight: height }}
          />
        </div>
      ) : (
        <div
          className="flex items-center justify-center text-dark-text-tertiary text-sm"
          style={{ height: '80px' }}
        >
          {info.configured ? 'Preview unavailable' : 'Connect Figma in Settings to see preview'}
        </div>
      )}
    </div>
  )
}

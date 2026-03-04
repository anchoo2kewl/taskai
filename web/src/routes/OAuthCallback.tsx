import { useEffect, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../state/AuthContext'

/**
 * Handles the OAuth success redirect from the backend.
 *
 * The backend redirects here with ?token=<JWT> on success, or the frontend
 * redirects here from an error with ?error=<msg>&code=<code>.
 *
 * Mounted at: /oauth/callback
 */
export default function OAuthCallback() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { loginWithToken } = useAuth()
  const processed = useRef(false)

  useEffect(() => {
    // Guard against StrictMode double-invocation
    if (processed.current) return
    processed.current = true

    const token = searchParams.get('token')
    const errorMsg = searchParams.get('error')

    if (token) {
      loginWithToken(token)
        .then(() => navigate('/app', { replace: true }))
        .catch(() => navigate('/login?oauth_error=Authentication+failed', { replace: true }))
      return
    }

    if (errorMsg) {
      navigate(`/login?oauth_error=${encodeURIComponent(errorMsg)}`, { replace: true })
      return
    }

    // No token and no error — something went wrong
    navigate('/login?oauth_error=Unexpected+OAuth+response', { replace: true })
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-dark-bg-base to-dark-bg-primary">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-10 w-10 border-b-2 border-primary-400" />
        <p className="text-sm text-dark-text-tertiary">Signing you in…</p>
      </div>
    </div>
  )
}

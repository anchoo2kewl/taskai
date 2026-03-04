import { useState, FormEvent, useEffect } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../state/AuthContext'
import { validateSignupForm } from '../lib/validation'
import { apiClient } from '../lib/api'
import Card, { CardHeader, CardBody } from '../components/ui/Card'
import TextInput from '../components/ui/TextInput'
import Button from '../components/ui/Button'
import FormError from '../components/ui/FormError'

export default function Signup() {
  const [searchParams] = useSearchParams()
  const inviteCodeFromURL = searchParams.get('code') || ''
  const emailFromURL = searchParams.get('email') || ''
  const redirectTo = searchParams.get('redirect')

  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState(emailFromURL)
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [inviteCode, setInviteCode] = useState(inviteCodeFromURL)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const [touched, setTouched] = useState<Record<string, boolean>>({})
  const { signup, error, loading, clearError, user } = useAuth()
  const navigate = useNavigate()

  // Invite validation state
  const [inviteValid, setInviteValid] = useState<boolean | null>(null)
  const [inviterName, setInviterName] = useState('')
  const [inviteMessage, setInviteMessage] = useState('')
  const [validatingInvite, setValidatingInvite] = useState(false)

  // Whether to show the email/password form (collapsed by default when OAuth is available)
  const [showEmailForm, setShowEmailForm] = useState(false)

  // Redirect if already logged in
  useEffect(() => {
    if (user) {
      navigate(redirectTo || '/app', { replace: true })
    }
  }, [user, navigate, redirectTo])

  // Validate invite code from URL on mount
  useEffect(() => {
    if (inviteCodeFromURL) {
      validateInviteCode(inviteCodeFromURL)
    }
  }, [inviteCodeFromURL])

  const validateInviteCode = async (code: string) => {
    if (!code.trim()) {
      setInviteValid(null)
      setInviterName('')
      setInviteMessage('')
      return
    }

    setValidatingInvite(true)
    try {
      const result = await apiClient.validateInvite(code.trim())
      setInviteValid(result.valid)
      setInviterName(result.inviter_name || '')
      setInviteMessage(result.message || '')
    } catch {
      setInviteValid(false)
      setInviteMessage('Failed to validate invite code')
    } finally {
      setValidatingInvite(false)
    }
  }

  const handleBlur = (field: string) => {
    setTouched({ ...touched, [field]: true })
    const validation = validateSignupForm(email, password, confirmPassword)
    setFieldErrors(validation.errors)
  }

  const handleInviteBlur = () => {
    if (inviteCode.trim()) {
      validateInviteCode(inviteCode)
    }
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    clearError()

    setTouched({ email: true, password: true, confirmPassword: true })

    const validation = validateSignupForm(email, password, confirmPassword)
    setFieldErrors(validation.errors)

    if (!validation.isValid) return

    if (!firstName.trim()) {
      setFieldErrors(prev => ({ ...prev, firstName: 'First name is required' }))
      return
    }

    if (!inviteCode.trim()) {
      setFieldErrors(prev => ({ ...prev, inviteCode: 'Invite code is required' }))
      return
    }

    try {
      await signup({ email, password, invite_code: inviteCode.trim(), first_name: firstName.trim(), last_name: lastName.trim() })
    } catch {
      // Error is handled by AuthContext
    }
  }

  const oauthReady = inviteValid === true && inviteCode

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-dark-bg-base to-dark-bg-primary px-4 relative">
      {/* Back to home */}
      <Link
        to="/"
        className="absolute top-6 left-6 text-sm text-dark-text-tertiary hover:text-dark-text-primary flex items-center gap-2 transition-colors duration-150"
      >
        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        Back
      </Link>

      <Card className="max-w-md w-full">
        <CardHeader>
          <div className="text-center">
            <img
              src="/logo.svg"
              alt="TaskAI"
              className="mx-auto h-16 w-16 mb-4"
            />
            <h2 className="text-xl font-semibold text-dark-text-primary tracking-tight">
              Create your account
            </h2>
            <p className="mt-2 text-xs text-dark-text-tertiary">
              TaskAI is invite-only. You need a referral to create an account.
            </p>
          </div>
        </CardHeader>

        <CardBody>
          {/* Step 1: Invite code */}
          <div className="space-y-4">
            <TextInput
              id="invite-code"
              name="invite-code"
              type="text"
              label="Invite Code"
              required
              value={inviteCode}
              onChange={(e) => {
                setInviteCode(e.target.value)
                setInviteValid(null)
              }}
              onBlur={handleInviteBlur}
              error={fieldErrors.inviteCode}
              placeholder="Paste your invite code"
              disabled={loading}
              helpText={validatingInvite ? 'Validating...' : undefined}
            />

            {/* Valid invite banner */}
            {inviteValid === true && (
              <div className="p-3 bg-success-500/10 border border-success-500/30 rounded-lg">
                <div className="flex items-center gap-2">
                  <svg className="w-4 h-4 text-success-400 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
                  </svg>
                  <span className="text-sm text-success-300">
                    Valid invite{inviterName ? ` from ${inviterName}` : ''}
                  </span>
                </div>
              </div>
            )}

            {/* Invalid invite banner */}
            {inviteValid === false && inviteMessage && (
              <div className="p-3 bg-danger-500/10 border border-danger-500/30 rounded-lg">
                <div className="flex items-center gap-2">
                  <svg className="w-4 h-4 text-danger-400 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
                    <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                  </svg>
                  <span className="text-sm text-danger-300">{inviteMessage}</span>
                </div>
              </div>
            )}
          </div>

          {/* Step 2: OAuth options — shown immediately after invite validates */}
          {oauthReady && (
            <div className="mt-5 space-y-3">
              <a
                href={`/api/auth/google?invite_code=${encodeURIComponent(inviteCode)}`}
                className="flex items-center justify-center gap-3 px-4 py-2.5 rounded-lg border border-dark-border-subtle bg-dark-bg-primary hover:bg-dark-bg-tertiary transition-colors text-sm text-dark-text-secondary"
              >
                <svg className="w-4 h-4" viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
                  <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
                  <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
                  <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
                </svg>
                Continue with Google
              </a>

              <a
                href={`/api/auth/github/login?invite_code=${encodeURIComponent(inviteCode)}`}
                className="flex items-center justify-center gap-3 px-4 py-2.5 rounded-lg border border-dark-border-subtle bg-dark-bg-primary hover:bg-dark-bg-tertiary transition-colors text-sm text-dark-text-secondary"
              >
                <svg className="w-4 h-4" viewBox="0 0 24 24" aria-hidden="true" fill="currentColor">
                  <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0 1 12 6.844a9.59 9.59 0 0 1 2.504.337c1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.02 10.02 0 0 0 22 12.017C22 6.484 17.522 2 12 2z" />
                </svg>
                Continue with GitHub
              </a>

              {/* Toggle to show email/password form */}
              <div className="relative mt-2">
                <div className="absolute inset-0 flex items-center">
                  <div className="w-full border-t border-dark-border-subtle" />
                </div>
                <div className="relative flex justify-center text-xs">
                  <span className="px-2 bg-dark-bg-secondary text-dark-text-quaternary">or</span>
                </div>
              </div>

              {!showEmailForm ? (
                <button
                  type="button"
                  onClick={() => setShowEmailForm(true)}
                  className="w-full text-sm text-dark-text-tertiary hover:text-dark-text-secondary transition-colors text-center py-1"
                >
                  Sign up with email instead
                </button>
              ) : null}
            </div>
          )}

          {/* Email/password form — always shown if no OAuth configured, otherwise toggled */}
          {(!oauthReady || showEmailForm) && (
            <form className={`space-y-6 ${oauthReady ? 'mt-4' : 'mt-4'}`} onSubmit={handleSubmit}>
              <FormError message={error || ''} />

              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-3">
                  <TextInput
                    id="first-name"
                    name="first-name"
                    type="text"
                    label="First Name"
                    autoComplete="given-name"
                    required
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    error={fieldErrors.firstName}
                    placeholder="First"
                    disabled={loading}
                  />
                  <TextInput
                    id="last-name"
                    name="last-name"
                    type="text"
                    label="Last Name"
                    autoComplete="family-name"
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    placeholder="Last"
                    disabled={loading}
                  />
                </div>

                <TextInput
                  id="email"
                  name="email"
                  type="email"
                  label="Email address"
                  autoComplete="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  onBlur={() => handleBlur('email')}
                  error={touched.email ? fieldErrors.email : undefined}
                  placeholder="you@example.com"
                  disabled={loading}
                />

                <TextInput
                  id="password"
                  name="password"
                  type="password"
                  label="Password"
                  autoComplete="new-password"
                  required
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  onBlur={() => handleBlur('password')}
                  error={touched.password ? fieldErrors.password : undefined}
                  helpText="Must be at least 8 characters with a letter and number"
                  placeholder="••••••••"
                  disabled={loading}
                />

                <TextInput
                  id="confirm-password"
                  name="confirm-password"
                  type="password"
                  label="Confirm Password"
                  autoComplete="new-password"
                  required
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  onBlur={() => handleBlur('confirmPassword')}
                  error={touched.confirmPassword ? fieldErrors.confirmPassword : undefined}
                  placeholder="••••••••"
                  disabled={loading}
                />
              </div>

              <Button
                type="submit"
                variant="primary"
                fullWidth
                loading={loading}
              >
                Create account
              </Button>
            </form>
          )}

          <div className="mt-4 text-sm text-center">
            <span className="text-dark-text-quaternary">Already have an account? </span>
            <Link to="/login" className="font-medium text-primary-400 hover:text-primary-300 transition-colors">
              Sign in
            </Link>
          </div>

          {/* Referral info */}
          {!inviteCodeFromURL && (
            <div className="mt-6 p-4 bg-dark-bg-primary border border-dark-border-subtle rounded-lg">
              <div className="flex gap-3">
                <svg className="w-5 h-5 text-primary-400 flex-shrink-0 mt-0.5" fill="currentColor" viewBox="0 0 20 20">
                  <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                </svg>
                <div className="text-xs text-dark-text-tertiary">
                  <p className="font-medium mb-1 text-dark-text-secondary">How do I get an invite?</p>
                  <p>Ask an existing TaskAI user to send you an invite link from their account settings. Each user can invite a limited number of friends.</p>
                </div>
              </div>
            </div>
          )}
        </CardBody>
      </Card>
    </div>
  )
}

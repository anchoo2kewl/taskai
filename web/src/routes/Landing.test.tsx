import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import Landing from './Landing'

vi.mock('react-router-dom', () => ({
  Link: ({ to, children, ...props }: { to: string; children: React.ReactNode; [key: string]: unknown }) => (
    <a href={to} {...props}>{children}</a>
  ),
}))

describe('Landing', () => {
  it('renders the hero heading', () => {
    render(<Landing />)
    expect(screen.getByText('Project management')).toBeInTheDocument()
    expect(screen.getByText('built for AI agents')).toBeInTheDocument()
  })

  it('renders "Sign in" and "Get started free" navigation links', () => {
    render(<Landing />)
    const signInLinks = screen.getAllByText('Sign in')
    expect(signInLinks.length).toBeGreaterThanOrEqual(1)

    const getStartedLinks = screen.getAllByText('Get started free')
    expect(getStartedLinks.length).toBeGreaterThanOrEqual(1)
  })

  it('renders feature sections', () => {
    render(<Landing />)
    expect(screen.getByText('Kanban boards')).toBeInTheDocument()
    expect(screen.getByText('Collaborative wiki')).toBeInTheDocument()
    expect(screen.getByText('Sprint planning')).toBeInTheDocument()
    expect(screen.getByText('Knowledge graph')).toBeInTheDocument()
  })

  it('renders the AI agents section', () => {
    render(<Landing />)
    expect(screen.getByText(/Let your AI agent/)).toBeInTheDocument()
    expect(screen.getByText('Model Context Protocol')).toBeInTheDocument()
  })

  it('renders the footer', () => {
    render(<Landing />)
    expect(screen.getByText('Project management for AI-native teams.')).toBeInTheDocument()
  })

  it('has links to /signup', () => {
    render(<Landing />)
    const signupLinks = screen.getAllByRole('link').filter(
      (link) => link.getAttribute('href') === '/signup'
    )
    expect(signupLinks.length).toBeGreaterThanOrEqual(1)
  })

  it('has links to /login', () => {
    render(<Landing />)
    const loginLinks = screen.getAllByRole('link').filter(
      (link) => link.getAttribute('href') === '/login'
    )
    expect(loginLinks.length).toBeGreaterThanOrEqual(1)
  })
})

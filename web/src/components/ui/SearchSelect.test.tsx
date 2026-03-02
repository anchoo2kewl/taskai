import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import SearchSelect from './SearchSelect'

const options = [
  { value: '1', label: 'Alice', description: 'alice@example.com' },
  { value: '2', label: 'Bob', description: 'bob@example.com' },
  { value: '3', label: 'Charlie' },
]

describe('SearchSelect', () => {
  it('renders with selected value displayed', () => {
    render(<SearchSelect value="1" onChange={() => {}} options={options} />)
    expect(screen.getByDisplayValue('Alice')).toBeInTheDocument()
  })

  it('shows options when button is clicked', async () => {
    const user = userEvent.setup()
    render(<SearchSelect value="" onChange={() => {}} options={options} placeholder="Pick one" />)

    // Click the dropdown button to open options
    const button = screen.getByRole('button')
    await user.click(button)
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.getByText('Charlie')).toBeInTheDocument()
  })

  it('filters options by query', async () => {
    const user = userEvent.setup()
    render(<SearchSelect value="" onChange={() => {}} options={options} />)

    const input = screen.getByRole('combobox')
    await user.clear(input)
    await user.type(input, 'ali')

    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.queryByText('Bob')).not.toBeInTheDocument()
    expect(screen.queryByText('Charlie')).not.toBeInTheDocument()
  })

  it('filters by description', async () => {
    const user = userEvent.setup()
    render(<SearchSelect value="" onChange={() => {}} options={options} />)

    const input = screen.getByRole('combobox')
    await user.clear(input)
    await user.type(input, 'bob@')

    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.queryByText('Alice')).not.toBeInTheDocument()
  })

  it('calls onChange when option is selected', async () => {
    const user = userEvent.setup()
    const handleChange = vi.fn()
    render(<SearchSelect value="" onChange={handleChange} options={options} />)

    const button = screen.getByRole('button')
    await user.click(button)
    await user.click(screen.getByText('Bob'))

    expect(handleChange).toHaveBeenCalledWith('2')
  })

  it('shows "No results" when filter matches nothing', async () => {
    const user = userEvent.setup()
    render(<SearchSelect value="" onChange={() => {}} options={options} />)

    const input = screen.getByRole('combobox')
    await user.clear(input)
    await user.type(input, 'zzzzz')

    expect(screen.getByText('No results')).toBeInTheDocument()
  })

  it('shows description under label', async () => {
    const user = userEvent.setup()
    render(<SearchSelect value="" onChange={() => {}} options={options} />)

    const button = screen.getByRole('button')
    await user.click(button)
    expect(screen.getByText('alice@example.com')).toBeInTheDocument()
  })
})

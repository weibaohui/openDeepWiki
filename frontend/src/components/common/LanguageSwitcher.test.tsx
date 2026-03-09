import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// Mock localStorage
const localStorageMock = {
  getItem: vi.fn(),
  setItem: vi.fn(),
  removeItem: vi.fn(),
  clear: vi.fn(),
}

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
})

// Mock antd Select
vi.mock('antd', async () => {
  const antd = await vi.importActual<typeof import('antd')>('antd')
  return {
    ...antd,
    Select: ({ value, options, onChange }: { value: string; options: { value: string; label: string }[]; onChange?: (val: string) => void }) => (
      <div data-testid="language-select" onClick={() => onChange && onChange(value)}>
        {options.map((opt) => (
          <div key={opt.value} data-label={opt.label}>{opt.label}</div>
        ))}
      </div>
    ),
  }
})

describe('LanguageSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('应该渲染语言切换按钮', async () => {
    const { default: LanguageSwitcher } = await import('./LanguageSwitcher')
    render(<LanguageSwitcher />)

    await waitFor(() => {
      expect(screen.getByTestId('language-select')).toBeInTheDocument()
    })
  })

  it('应该在点击时显示语言菜单', async () => {
    const { default: LanguageSwitcher } = await import('./LanguageSwitcher')
    render(<LanguageSwitcher />)

    const select = screen.getByTestId('language-select')
    await userEvent.click(select)

    await waitFor(() => {
      expect(screen.getByText(/简体中文/)).toBeInTheDocument()
      expect(screen.getByText('English')).toBeInTheDocument()
    })
  })

  it('应该从localStorage读取初始语言', async () => {
    localStorageMock.getItem.mockReturnValue('en-US')

    const { default: LanguageSwitcher } = await import('./LanguageSwitcher')
    render(<LanguageSwitcher />)

    expect(localStorageMock.getItem).toHaveBeenCalledWith('language')
  })
})

describe('LanguageSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('应该渲染语言切换按钮', () => {
    render(<LanguageSwitcher />)

    const button = screen.getByRole('button', { name: /language/i })
    expect(button).toBeInTheDocument()
  })

  it('应该在点击时显示语言菜单', async () => {
    render(<LanguageSwitcher />)

    const button = screen.getByRole('button', { name: /language/i })
    await userEvent.click(button)

    await waitFor(() => {
      expect(screen.getByText('中文')).toBeInTheDocument()
      expect(screen.getByText('English')).toBeInTheDocument()
    })
  })

  it('应该从localStorage读取初始语言', () => {
    localStorageMock.getItem.mockReturnValue('en-US')

    render(<LanguageSwitcher />)

    expect(localStorageMock.getItem).toHaveBeenCalledWith('language')
  })
})

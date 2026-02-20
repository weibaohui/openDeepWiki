import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
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

// Mock AppConfigContext
vi.mock('@/context/AppConfigContext', () => ({
  useAppConfig: () => ({
    t: (key: string, fallback?: string) => fallback || key,
    themeMode: 'light',
    setThemeMode: vi.fn(),
  }),
}))

// Mock react-router-dom
vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
}))

describe('ThemeSwitcher', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('应该渲染主题切换按钮', async () => {
    const { default: ThemeSwitcher } = await import('./ThemeSwitcher')
    render(<ThemeSwitcher />)

    const button = screen.getByRole('button', { name: /theme/i })
    expect(button).toBeInTheDocument()
  })

  it('应该在点击时切换主题', async () => {
    const { default: ThemeSwitcher } = await import('./ThemeSwitcher')
    localStorageMock.getItem.mockReturnValue('light')

    render(<ThemeSwitcher />)

    const button = screen.getByRole('button', { name: /theme/i })
    await userEvent.click(button)

    expect(localStorageMock.setItem).toHaveBeenCalledWith('theme', 'dark')
  })

  it('应该从localStorage读取初始主题', async () => {
    const { default: ThemeSwitcher } = await import('./ThemeSwitcher')
    localStorageMock.getItem.mockReturnValue('dark')

    render(<ThemeSwitcher />)

    expect(localStorageMock.getItem).toHaveBeenCalledWith('theme')
  })
})

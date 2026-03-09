import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import Settings from './Settings'

// Mock components
vi.mock('@/components/settings/APIKeyList', () => ({
  default: () => <div>APIKeyList Component</div>,
}))

vi.mock('@/components/settings/TaskMonitor', () => ({
  default: () => <div>TaskMonitor Component</div>,
}))

vi.mock('@/components/common/ThemeSwitcher', () => ({
  ThemeSwitcher: () => <button data-testid="theme-switcher">Theme Switcher</button>,
}))

vi.mock('@/components/common/LanguageSwitcher', () => ({
  LanguageSwitcher: () => <button data-testid="language-switcher">Language Switcher</button>,
}))

// Mock useAppConfig
vi.mock('@/context/AppConfigContext', () => ({
  useAppConfig: () => ({
    t: (key: string, fallback?: string) => fallback || key,
    themeMode: 'light',
    language: 'zh-CN',
    setLanguage: vi.fn(),
    setThemeMode: vi.fn(),
  }),
}))

function renderWithRouter(component: React.ReactNode) {
  return render(
    <MemoryRouter initialEntries={['/settings']}>
      <Routes>
        <Route path="/settings" element={component} />
      </Routes>
    </MemoryRouter>
  )
}

describe('Settings', () => {
  describe('Tab Navigation', () => {
    it('应该渲染设置页面', () => {
      renderWithRouter(<Settings />)

      expect(screen.getByText(/settings/i)).toBeInTheDocument()
    })

    it('应该显示API Key管理标签', async () => {
      renderWithRouter(<Settings />)

      await waitFor(() => {
        expect(screen.getByText(/api key management/i)).toBeInTheDocument()
      })
    })

    it('应该显示任务运行监控标签', async () => {
      renderWithRouter(<Settings />)

      await waitFor(() => {
        expect(screen.getByText(/task run monitoring/i)).toBeInTheDocument()
      })
    })

    it('应该支持标签切换', async () => {
      renderWithRouter(<Settings />)

      const apiKeyTab = await screen.findByRole('tab', { name: /api key management/i })
      expect(apiKeyTab).toBeInTheDocument()

      await userEvent.click(apiKeyTab)

      expect(screen.getByText('APIKeyList Component')).toBeInTheDocument()
    })
  })

  describe('Navigation Bar', () => {
    it('应该渲染返回按钮', () => {
      renderWithRouter(<Settings />)

      const backButton = screen.getByRole('button', { name: /arrow-left/i })
      expect(backButton).toBeInTheDocument()
    })

    it('应该渲染语言切换器', () => {
      renderWithRouter(<Settings />)

      const languageSwitcher = screen.getByTestId('language-switcher')
      expect(languageSwitcher).toBeInTheDocument()
    })

    it('应该渲染主题切换器', () => {
      renderWithRouter(<Settings />)

      const themeSwitcher = screen.getByTestId('theme-switcher')
      expect(themeSwitcher).toBeInTheDocument()
    })

    it('应该点击返回按钮导航到首页', async () => {
      renderWithRouter(<Settings />)

      const backButton = screen.getByRole('button', { name: /arrow-left/i })
      await userEvent.click(backButton)

      expect(window.location.pathname).toBe('/')
    })
  })
})

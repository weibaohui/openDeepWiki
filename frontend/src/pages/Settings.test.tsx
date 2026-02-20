import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import Settings from './Settings'
import APIKeyList from '@/components/settings/APIKeyList'
import TaskMonitor from '@/components/settings/TaskMonitor'

// Mock components
vi.mock('@/components/settings/APIKeyList', () => ({
  default: () => <div>APIKeyList Component</div>,
}))

vi.mock('@/components/settings/TaskMonitor', () => ({
  default: () => <div>TaskMonitor Component</div>,
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

      await waitFor(() => {
        const apiKeyTab = screen.getByRole('tab', { name: 'api-keys' })
        expect(apiKeyTab).toBeInTheDocument()
      })

      await userEvent.click(apiKeyTab)

      expect(screen.getByText('APIKeyList Component')).toBeInTheDocument()
    })
  })

  describe('Navigation Bar', () => {
    it('应该渲染返回按钮', () => {
      renderWithRouter(<Settings />)

      const backButton = screen.getByRole('button', { name: /back/i })
      expect(backButton).toBeInTheDocument()
    })

    it('应该渲染语言切换器', () => {
      renderWithRouter(<Settings />)

      const languageSwitcher = screen.getByRole('button', { name: /language/i })
      expect(languageSwitcher).toBeInTheDocument()
    })

    it('应该渲染主题切换器', () => {
      renderWithRouter(<Settings />)

      const themeSwitcher = screen.getByRole('button', { name: /theme/i })
      expect(themeSwitcher).toBeInTheDocument()
    })

    it('应该点击返回按钮导航到首页', async () => {
      renderWithRouter(<Settings />)

      const backButton = screen.getByRole('button', { name: /back/i })
      await userEvent.click(backButton)

      expect(window.location.pathname).toBe('/')
    })
  })
})

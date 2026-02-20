/**
 * API Key Manager 组件测试
 * 测试 API Key 管理页面的主要功能
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, within, fireEvent, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { http } from 'msw'
import { setupServer } from 'msw/node'
import userEvent from '@testing-library/user-event'
import APIKeyManager from './APIKeyManager'
import * as api from '../services/api'

// Mock antd message
vi.mock('antd', async () => {
  const antd = await vi.importActual<any>('antd')
  return {
    ...antd,
    message: {
      useMessage: () => [{ error: vi.fn(), success: vi.fn(), warning: vi.fn() }],
    },
  }
})

// Mock ThemeSwitcher
vi.mock('@/components/common/ThemeSwitcher', () => ({
  ThemeSwitcher: () => <div>ThemeSwitcher</div>,
}))

// Mock LanguageSwitcher
vi.mock('@/components/common/LanguageSwitcher', () => ({
  LanguageSwitcher: () => <div>LanguageSwitcher</div>,
}))

// Mock useAppConfig
vi.mock('@/context/AppConfigContext', () => ({
  useAppConfig: () => ({
    t: (key: string, fallback?: string) => fallback || key,
    themeMode: 'light',
    locale: 'en-US',
    setLocale: vi.fn(),
    setThemeMode: vi.fn(),
  }),
}))

// Setup MSW handlers for vitest
import { HttpResponse } from 'msw'

// Mock MSW handlers
const handlers = [
  http.get('/api/api-keys', () => {
    return HttpResponse.json({
      data: mockAPIKeys,
      total: mockAPIKeys.length
    })
  }),
  http.get('/api/api-keys/stats', () => {
    return HttpResponse.json(mockStats)
  }),
]

// Mock MSW 服务器
const server = setupServer(...handlers)

// API 类型定义
interface APIKey {
  id: number
  name: string
  provider: string
  base_url: string
  api_key: string
  model: string
  priority: number
  status: 'enabled' | 'disabled' | 'unavailable'
  request_count: number
  error_count: number
  last_used_at?: string
  rate_limit_reset_at?: string
  created_at: string
  updated_at: string
}

interface APIKeyStats {
  total_count: number
  enabled_count: number
  disabled_count: number
  unavailable_count: number
  total_requests: number
  total_errors: number
}

const mockAPIKeys: APIKey[] = [
  {
    id: 1,
    name: 'openai-primary',
    provider: 'openai',
    base_url: 'https://api.openai.com/v1',
    api_key: 'sk-test123456789',
    model: 'gpt-4',
    priority: 0,
    status: 'enabled',
    request_count: 100,
    error_count: 5,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 2,
    name: 'anthropic-backup',
    provider: 'anthropic',
    base_url: 'https://api.anthropic.com/v1',
    api_key: 'sk-ant123456789',
    model: 'claude-3',
    priority: 10,
    status: 'enabled',
    request_count: 50,
    error_count: 2,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

const mockStats: APIKeyStats = {
  total_count: 2,
  enabled_count: 2,
  disabled_count: 0,
  unavailable_count: 0,
  total_requests: 150,
  total_errors: 7,
}

beforeAll(() => server.listen({
  onUnhandledRequest: 'warn',
}))
beforeEach(() => {
  // 添加额外的处理器用于测试中动态添加的 API
  server.use(
    http.get('/api/api-keys', ({ request }) => {
      console.log('MSW: GET /api/api-keys', request.url)
      return HttpResponse.json({ data: mockAPIKeys, total: mockAPIKeys.length })
    }),
    http.get('/api/api-keys/stats', ({ request }) => {
      console.log('MSW: GET /api/api-keys/stats', request.url)
      return HttpResponse.json(mockStats)
    })
  )
})
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

// Mock React Router
const renderWithRouter = (component: React.ReactNode) => {
  return render(
    <BrowserRouter>{component}</BrowserRouter>
  )
}

describe('APIKeyManager', () => {

  describe('渲染测试', () => {
    it('应该正确渲染页面标题', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByText('API Key Management')).toBeInTheDocument()
      })
    })

    it('应该显示统计卡片', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByText('Total')).toBeInTheDocument()
        expect(screen.getByText('Enabled')).toBeInTheDocument()
        expect(screen.getByText('Requests')).toBeInTheDocument()
        expect(screen.getByText('Errors')).toBeInTheDocument()
      })

      // 验证统计数据
      await waitFor(() => {
        expect(screen.getByText('2')).toBeInTheDocument() // total_count
      })
    })

    it('应该显示 API Key 表格', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByText('openai-primary')).toBeInTheDocument()
        expect(screen.getByText('anthropic-backup')).toBeInTheDocument()
      })
    })

    it('应该显示操作按钮', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /add new/i })).toBeInTheDocument()
        expect(screen.getByRole('button', { name: /refresh/i })).toBeInTheDocument()
      })
    })
  })

  describe('数据展示测试', () => {
    it('应该正确显示 API Key 列表', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(table).toBeInTheDocument()

        // 检查表头
        expect(within(table).getByText('Name')).toBeInTheDocument()
        expect(within(table).getByText('Provider')).toBeInTheDocument()
        expect(within(table).getByText('Model')).toBeInTheDocument()
        expect(within(table).getByText('Priority')).toBeInTheDocument()
        expect(within(table).getByText('Status')).toBeInTheDocument()
      })
    })

    it('应该正确显示 Provider 标签', async () => {
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByText('openai')).toBeInTheDocument()
        expect(screen.getByText('anthropic')).toBeInTheDocument()
      })
    })
  })

  describe('用户交互测试', () => {
    it('应该能够打开创建模态框', async () => {
      const user = userEvent.setup()
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        const addButton = screen.getByRole('button', { name: /add new/i })
        expect(addButton).toBeInTheDocument()
      })

      await user.click(screen.getByRole('button', { name: /add new/i }))

      expect(screen.getByText('Add API Key')).toBeInTheDocument()
      expect(screen.getByLabelText(/name/i)).toBeInTheDocument()
    })

    it('应该能够关闭模态框', async () => {
      const user = userEvent.setup()
      renderWithRouter(<APIKeyManager />)

      // 打开模态框
      await user.click(screen.getByRole('button', { name: /add new/i }))

      expect(screen.getByText('Add API Key')).toBeInTheDocument()

      // 关闭模态框（点击 Cancel 或 ESC）
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      await user.click(cancelButton)

      // 模态框应该关闭
      await waitFor(() => {
        expect(screen.queryByText('Add API Key')).not.toBeInTheDocument()
      })
    })

    it('应该能够刷新数据', async () => {
      const user = userEvent.setup()
      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        expect(screen.getByText('openai-primary')).toBeInTheDocument()
      })

      const refreshButton = screen.getByRole('button', { name: /refresh/i })
      await user.click(refreshButton)

      // 数据应该保持显示
      await waitFor(() => {
        expect(screen.getByText('openai-primary')).toBeInTheDocument()
      })
    })

    it('应该能够切换 API Key 状态', async () => {
      const user = userEvent.setup()

      // Mock 状态更新 API
      server.use(
        http.patch('/api/api-keys/1/status', () => HttpResponse.json({ message: 'status updated successfully' }, { status: 200 }))
      )

      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(table).toBeInTheDocument()
      })

      // 找到第一个 switch 按钮
      const switches = screen.getAllByRole('switch')
      expect(switches.length).toBeGreaterThan(0)

      // 点击第一个 switch
      await user.click(switches[0])

      // 等待 API 调用完成
      await waitFor(() => {
        // 验证状态更新
        expect(screen.getByText('openai-primary')).toBeInTheDocument()
      })
    })
  })

  describe('表单测试', () => {
    it('创建表单应该有必填字段验证', async () => {
      const user = userEvent.setup()
      renderWithRouter(<APIKeyManager />)

      // 打开创建模态框
      await user.click(screen.getByRole('button', { name: /add new/i }))

      // 尝试直接提交（不填写字段）
      const okButton = screen.getByRole('button', { name: /ok/i })
      await user.click(okButton)

      // 应该显示验证错误
      await waitFor(() => {
        const nameInput = screen.getByLabelText(/name/i)
        expect(nameInput).toBeInvalid()
      })
    })

    it('应该能够填写并提交创建表单', async () => {
      const user = userEvent.setup()

      // Mock 创建 API
      server.use(
        http.post('/api/api-keys', () => HttpResponse.json({
              ...mockAPIKeys[0],
              id: 3,
              name: 'new-api-key'
            }, { status: 201 }))
      )

      renderWithRouter(<APIKeyManager />)

      // 打开创建模态框
      await user.click(screen.getByRole('button', { name: /add new/i }))

      // 填写表单
      await user.type(screen.getByLabelText(/name/i), 'new-api-key')
      await user.type(screen.getByLabelText(/model/i), 'gpt-4')

      // 提交表单
      await user.click(screen.getByRole('button', { name: /ok/i }))

      // 模态框应该关闭
      await waitFor(() => {
        expect(screen.queryByText('Add API Key')).not.toBeInTheDocument()
      })
    })
  })

  describe('边界情况测试', () => {
    it('当 API 返回空列表时应该显示空状态', async () => {
      server.use(
        http.get('/api/api-keys', () => HttpResponse.json({
              data: [],
              total: 0
            }, { status: 200 }))
      )

      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        const table = screen.getByRole('table')
        expect(table).toBeInTheDocument()
      })

      // 表格应该存在但不应该显示任何 API Key
      expect(screen.queryByText('openai-primary')).not.toBeInTheDocument()
    })

    it('当 API 返回错误时应该显示错误信息', async () => {
      server.use(
        http.get('/api/api-keys', () => HttpResponse.json({ error: 'Server error' }, { status: 500 }))
      )

      renderWithRouter(<APIKeyManager />)

      await waitFor(() => {
        // 应该显示错误消息
        expect(screen.getByText(/failed to load data/i)).toBeInTheDocument()
      })
    })
  })
})

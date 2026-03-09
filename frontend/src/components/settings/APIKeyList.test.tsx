import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import APIKeyList from './APIKeyList'
import type { APIKey } from '../../types'

// Mock 服务器
const server = setupServer()

beforeAll(() => server.listen())
beforeEach(() => server.resetHandlers())
afterAll(() => server.close())

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

const mockApiKeys: APIKey[] = [
  {
    id: 1,
    name: 'test-key-1',
    provider: 'openai',
    base_url: 'https://api.openai.com/v1',
    api_key: 'sk-test1',
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
    name: 'test-key-2',
    provider: 'anthropic',
    base_url: 'https://api.anthropic.com',
    api_key: 'sk-test2',
    model: 'claude-3',
    priority: 1,
    status: 'disabled',
    request_count: 50,
    error_count: 2,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

const mockStats = {
  total_count: 10,
  enabled_count: 8,
  disabled_count: 2,
  unavailable_count: 0,
  total_requests: 1000,
  total_errors: 50,
}

describe('APIKeyList', () => {
  describe('API Key List', () => {
    it('应该渲染API Key列表', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      await waitFor(() => {
        expect(screen.getByText('test-key-1')).toBeInTheDocument()
        expect(screen.getByText('test-key-2')).toBeInTheDocument()
      })
    })

    it('应该显示API Key脱敏', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      await waitFor(() => {
        expect(screen.queryByText('sk-test1')).not.toBeInTheDocument()
        expect(screen.queryByText('sk-test2')).not.toBeInTheDocument()
        expect(screen.getAllByText('sk-****').length).toBeGreaterThan(0)
      })
    })

    it('应该显示提供商', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      await waitFor(() => {
        expect(screen.getByText('openai')).toBeInTheDocument()
        expect(screen.getByText('anthropic')).toBeInTheDocument()
      })
    })

    it('应该显示模型', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      await waitFor(() => {
        expect(screen.getByText('gpt-4')).toBeInTheDocument()
        expect(screen.getByText('claude-3')).toBeInTheDocument()
      })
    })

    it('应该显示状态', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      await waitFor(() => {
        expect(screen.getByText('enabled')).toBeInTheDocument()
        expect(screen.getByText('disabled')).toBeInTheDocument()
      })
    })
  })

  describe('API Key Stats', () => {
    it('应该显示统计信息', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      await waitFor(() => {
        expect(screen.getByText(/total.*10/i)).toBeInTheDocument()
        expect(screen.getByText(/enabled.*8/i)).toBeInTheDocument()
        expect(screen.getByText(/disabled.*2/i)).toBeInTheDocument()
        expect(screen.getByText(/requests.*1000/i)).toBeInTheDocument()
        expect(screen.getByText(/errors.*50/i)).toBeInTheDocument()
      })
    })
  })

  describe('Add API Key', () => {
    it('应该打开添加模态框', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      const addButton = screen.getByRole('button', { name: /add new/i })
      await userEvent.click(addButton)

      await waitFor(() => {
        expect(screen.getByText(/add api key/i)).toBeInTheDocument()
      })
    })

    it('应该成功添加API Key', async () => {
      const newKey = {
        id: 3,
        name: 'new-key',
        provider: 'openai',
        base_url: 'https://api.openai.com/v1',
        api_key: 'sk-new',
        model: 'gpt-4',
        priority: 0,
        status: 'enabled',
        request_count: 0,
        error_count: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      }

      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        }),
        http.post('/api/api-keys', () => {
          return HttpResponse.json(newKey, { status: 201 })
        })
      )

      render(<APIKeyList />)

      const addButton = screen.getByRole('button', { name: /add new/i })
      await userEvent.click(addButton)

      const nameInput = screen.getByPlaceholderText(/name/i)
      await userEvent.type(nameInput, 'new-key')

      const apiKeyInput = screen.getByPlaceholderText(/api key/i)
      await userEvent.type(apiKeyInput, 'sk-new')

      const okButton = screen.getByRole('button', { name: /ok/i })
      await userEvent.click(okButton)

      await waitFor(() => {
        expect(screen.getByText('new-key')).toBeInTheDocument()
      })
    })
  })

  describe('Edit API Key', () => {
    it('应该打开编辑模态框', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      const editButton = screen.getByRole('button', { name: /edit/i })
      await userEvent.click(editButton)

      await waitFor(() => {
        expect(screen.getByText(/edit api key/i)).toBeInTheDocument()
      })
    })

    it('应该更新API Key', async () => {
      const updatedKey = { ...mockApiKeys[0], name: 'updated-key', priority: 10 }

      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        }),
        http.put('/api/api-keys/1', () => {
          return HttpResponse.json(updatedKey)
        })
      )

      render(<APIKeyList />)

      const editButton = screen.getByRole('button', { name: /edit/i })
      await userEvent.click(editButton)

      const nameInput = screen.getByDisplayValue('test-key-1')
      await userEvent.clear(nameInput)
      await userEvent.type(nameInput, 'updated-key')

      const okButton = screen.getByRole('button', { name: /ok/i })
      await userEvent.click(okButton)

      await waitFor(() => {
        expect(screen.getByText('updated-key')).toBeInTheDocument()
      })
    })
  })

  describe('Delete API Key', () => {
    it('应该删除API Key', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        }),
        http.delete('/api/api-keys/1', () => {
          return HttpResponse.json({ message: 'deleted successfully' })
        })
      )

      render(<APIKeyList />)

      const deleteButton = screen.getByRole('button', { name: /delete/i })
      await userEvent.click(deleteButton)

      await waitFor(() => {
        expect(screen.getByRole('button', { name: /ok/i })).toBeInTheDocument()
      })

      await userEvent.click(screen.getByRole('button', { name: /ok/i }))

      await waitFor(() => {
        expect(screen.queryByText('test-key-1')).not.toBeInTheDocument()
      })
    })
  })

  describe('Update Status', () => {
    it('应该更新API Key状态', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        }),
        http.patch('/api/api-keys/1/status', () => {
          return HttpResponse.json({ message: 'status updated successfully' })
        })
      )

      render(<APIKeyList />)

      const switchButton = screen.getByRole('switch')
      await userEvent.click(switchButton)

      await waitFor(() => {
        expect(screen.getByText('disabled')).toBeInTheDocument()
      })
    })
  })

  describe('Refresh', () => {
    it('应该刷新数据', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ data: mockApiKeys, total: 2 })
        }),
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats)
        })
      )

      render(<APIKeyList />)

      const refreshButton = screen.getByRole('button', { name: /refresh/i })
      await userEvent.click(refreshButton)

      await waitFor(() => {
        expect(screen.getByText('test-key-1')).toBeInTheDocument()
      })
    })
  })
})

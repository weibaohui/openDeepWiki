import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import Sync from './Sync'
import type { Repository, Document, Task } from '../types'

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

// Mock 服务器
const server = setupServer()

beforeAll(() => server.listen())
beforeEach(() => server.resetHandlers())
afterAll(() => server.close())

// Mock useParams
vi.mock('react-router-dom', async () => ({
  ...(await vi.importActual('react-router-dom')),
  useParams: () => ({ id: '1' }),
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

const mockRepository: Repository = {
  id: 1,
  name: 'test-repo',
  url: 'https://github.com/test/repo',
  status: 'ready',
  created_at: new Date().toISOString(),
  size_mb: 10.5,
  clone_branch: 'main',
  clone_commit_id: 'abc123',
}

const mockDocuments: Document[] = [
  {
    id: 1,
    title: 'Document 1',
    content: '# Document 1',
    repository_id: 1,
    path: 'docs/doc1.md',
    task_id: 1,
    version: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

const mockTasks: Task[] = [
  {
    id: 1,
    title: 'Task 1',
    status: 'completed',
    task_type: 'DocWrite',
    writer_name: 'DefaultWriter',
    repository_id: 1,
    doc_id: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

function renderWithRouter(component: React.ReactNode, initialEntries: string[] = ['/sync']) {
  return render(
    <MemoryRouter initialEntries={initialEntries}>
      <Routes>
        <Route path="/sync" element={component} />
      </Routes>
    </MemoryRouter>
  )
}

describe('Sync', () => {
  describe('Local Sync URL', () => {
    it('应该渲染本地同步地址', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      await waitFor(() => {
        expect(screen.getByText(/local sync url/i)).toBeInTheDocument()
      })
    })

    it('应该支持复制同步地址', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      const copyButton = screen.getByRole('button', { name: /copy/i })
      await userEvent.click(copyButton)

      await waitFor(() => {
        expect(screen.getByText(/copy success/i)).toBeInTheDocument()
      })
    })
  })

  describe('Sync Mode Selection', () => {
    it('应该支持推送模式', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/repositories/1/documents', () => {
          return HttpResponse.json(mockDocuments)
        }),
        http.get('/api/repositories/1/tasks', () => {
          return HttpResponse.json(mockTasks)
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      const pushButton = screen.getByRole('button', { name: /push/i })
      await userEvent.click(pushButton)

      await waitFor(() => {
        expect(screen.getByText(/push/i)).toBeInTheDocument()
      })
    })

    it('应该支持拉取模式', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      const pullButton = screen.getByRole('button', { name: /pull/i })
      await userEvent.click(pullButton)

      await waitFor(() => {
        expect(screen.getByText(/pull/i)).toBeInTheDocument()
      })
    })
  })

  describe('Target Server Configuration', () => {
    it('应该保存同步目标', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        }),
        http.post('/api/sync/target-save', () => {
          return HttpResponse.json({ id: 1, url: 'http://target-server/api/sync' })
        })
      )

      renderWithRouter(<Sync />)

      const urlInput = screen.getByPlaceholderText(/target server/i)
      await userEvent.type(urlInput, 'http://target-server/api/sync')

      const saveButton = screen.getByRole('button', { name: /save/i })
      await userEvent.click(saveButton)

      await waitFor(() => {
        expect(screen.getByText(/target saved successfully/i)).toBeInTheDocument()
      })
    })

    it('应该选择已保存的目标', async () => {
      const savedTargets = [
        { id: 1, url: 'http://server1/api/sync' },
        { id: 2, url: 'http://server2/api/sync' },
      ]

      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: savedTargets, total: 2 })
        })
      )

      renderWithRouter(<Sync />)

      const selectButtons = screen.getAllByRole('button', { name: /select/i })
      expect(selectButtons.length).toBe(2)
    })

    it('应该删除已保存的目标', async () => {
      const savedTargets = [
        { id: 1, url: 'http://server1/api/sync' },
      ]

      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: savedTargets, total: 1 })
        }),
        http.post('/api/sync/target-delete', () => {
          return HttpResponse.json({ id: 1, message: 'deleted successfully' })
        })
      )

      renderWithRouter(<Sync />)

      const deleteButton = screen.getByRole('button', { name: /delete/i })
      await userEvent.click(deleteButton)

      await waitFor(() => {
        expect(screen.getByText(/target deleted successfully/i)).toBeInTheDocument()
      })
    })
  })

  describe('Repository Selection', () => {
    it('应该加载仓库列表', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      const repositorySelect = screen.getByPlaceholderText(/select repository/i)
      await userEvent.click(repositorySelect)

      await waitFor(() => {
        expect(screen.getByText('test-repo')).toBeInTheDocument()
      })
    })
  })

  describe('Document Selection', () => {
    it('应该加载文档列表', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/repositories/1/documents', () => {
          return HttpResponse.json(mockDocuments)
        }),
        http.get('/api/repositories/1/tasks', () => {
          return HttpResponse.json(mockTasks)
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      await waitFor(() => {
        expect(screen.getByText('Document 1')).toBeInTheDocument()
      })
    })

    it('应该支持多选文档', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/repositories/1/documents', () => {
          return HttpResponse.json(mockDocuments)
        }),
        http.get('/api/repositories/1/tasks', () => {
          return HttpResponse.json(mockTasks)
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      const documentSelect = screen.getByPlaceholderText(/select documents/i)
      await userEvent.click(documentSelect)

      const checkboxes = screen.getAllByRole('checkbox')
      expect(checkboxes.length).toBeGreaterThan(0)
    })
  })

  describe('Start Sync', () => {
    it('应该启动同步', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/repositories/1/documents', () => {
          return HttpResponse.json(mockDocuments)
        }),
        http.get('/api/repositories/1/tasks', () => {
          return HttpResponse.json(mockTasks)
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        }),
        http.post('/api/sync', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              sync_id: 'sync-123',
              repository_id: 1,
              total_tasks: 10,
              status: 'running',
            },
          })
        })
      )

      renderWithRouter(<Sync />)

      const targetInput = screen.getByPlaceholderText(/target server/i)
      await userEvent.type(targetInput, 'http://target-server/api/sync')

      const repositorySelect = screen.getByPlaceholderText(/select repository/i)
      await userEvent.click(repositorySelect)
      await userEvent.click(screen.getByText('test-repo'))

      const startButton = screen.getByRole('button', { name: /start/i })
      await userEvent.click(startButton)

      await waitFor(() => {
        expect(screen.getByText(/sync started successfully/i)).toBeInTheDocument()
      })
    })
  })

  describe('Sync Progress', () => {
    it('应该显示同步进度', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/repositories/1/documents', () => {
          return HttpResponse.json(mockDocuments)
        }),
        http.get('/api/repositories/1/tasks', () => {
          return HttpResponse.json(mockTasks)
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        }),
        http.get('/api/sync/status/sync-123', () => {
          return HttpResponse.json({
            code: 0,
            data: {
              sync_id: 'sync-123',
              status: 'running',
              total_tasks: 10,
              completed_tasks: 5,
              failed_tasks: 0,
              current_task: 'Processing document 5',
            },
          })
        })
      )

      renderWithRouter(<Sync />)

      await waitFor(() => {
        expect(screen.getByText(/sync status/i)).toBeInTheDocument()
        expect(screen.getByText(/running/i)).toBeInTheDocument()
        expect(screen.getByText(/5\/10/i)).toBeInTheDocument()
      })
    })
  })

  describe('Sync History', () => {
    it('应该打开历史抽屉', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        })
      )

      renderWithRouter(<Sync />)

      const historyButton = screen.getByRole('button', { name: /history/i })
      await userEvent.click(historyButton)

      await waitFor(() => {
        expect(screen.getByText(/sync history/i)).toBeInTheDocument()
      })
    })

    it('应该加载历史记录', async () => {
      const mockHistory = [
        {
          id: 1,
          event_type: 'DocPushed',
          repository_id: 1,
          repository_name: 'test-repo',
          doc_id: 1,
          success: true,
          target_server: 'http://target-server/api/sync',
          created_at: new Date().toISOString(),
        },
      ]

      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([mockRepository])
        }),
        http.get('/api/sync/target-list', () => {
          return HttpResponse.json({ data: [], total: 0 })
        }),
        http.get('/api/sync/event-list', () => {
          return HttpResponse.json({
            code: 0,
            data: mockHistory,
          })
        })
      )

      renderWithRouter(<Sync />)

      const historyButton = screen.getByRole('button', { name: /history/i })
      await userEvent.click(historyButton)

      await waitFor(() => {
        expect(screen.getByText('DocPushed')).toBeInTheDocument()
      })
    })
  })
})

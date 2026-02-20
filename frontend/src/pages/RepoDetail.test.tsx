import { http } from 'msw'
import { HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { MemoryRouter, Route, Routes, useParams } from 'react-router-dom'
import RepoDetail from './RepoDetail'
import type { Task, Repository } from '../types'

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
const server = setupServer(...handlers)

beforeAll(() => server.listen())
beforeEach(() => server.resetHandlers())
afterAll(() => server.close())

// Mock useParams
vi.mock('react-router-dom', async () => ({
  ...(await vi.importActual('react-router-dom')),
  useParams: () => ({ id: '1' }),
}))

const mockRepository: Repository = {
  id: 1,
  name: 'test-repo',
  url: 'https://github.com/test/repo',
  status: 'ready',
  created_at: new Date().toISOString(),
  size_mb: 10.5,
  clone_branch: 'main',
}

const mockTasks: Task[] = [
  {
    id: 1,
    title: 'Task 1',
    status: 'completed',
    task_type: 'DocWrite',
    writer_name: 'DefaultWriter',
    repository_id: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 2,
    title: 'Task 2',
    status: 'pending',
    task_type: 'TocWrite',
    writer_name: 'DefaultWriter',
    repository_id: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
  {
    id: 3,
    title: 'Task 3',
    status: 'failed',
    task_type: 'APIWriter',
    writer_name: 'APIWriter',
    error_msg: 'Failed to generate',
    repository_id: 1,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  },
]

function renderWithRouter(component: React.ReactNode) {
  return render(
    <MemoryRouter initialEntries={['/repo/1']}>
      <Routes>
        <Route path="/repo/:id" element={component} />
      </Routes>
    </MemoryRouter>
  )
}

describe('RepoDetail', () => {
  describe('Task List', () => {
    it('应该渲染任务列表', async () => {
      server.use(
        http.get('/api/repositories/1', () => new Response(JSON.stringify(mockRepository), { headers: { 'Content-Type': 'application/json' }}),
        http.get('/api/repositories/1/tasks', () => new Response(JSON.stringify(mockTasks), { headers: { 'Content-Type': 'application/json' }})
      )

      renderWithRouter(<RepoDetail />)

      await waitFor(() => {
        expect(screen.getByText('Task 1')).toBeInTheDocument()
        expect(screen.getByText('Task 2')).toBeInTheDocument()
        expect(screen.getByText('Task 3')).toBeInTheDocument()
      })
    })

    it('应该显示任务状态标签', async () => {
      server.use(
        http.get('/api/repositories/1', () => new Response(JSON.stringify(mockRepository), { headers: { 'Content-Type': 'application/json' }}),
        http.get('/api/repositories/1/tasks', () => new Response(JSON.stringify(mockTasks), { headers: { 'Content-Type': 'application/json' }})
      )

      renderWithRouter(<RepoDetail />)

      await waitFor(() => {
        expect(screen.getByText('completed')).toBeInTheDocument()
        expect(screen.getByText('pending')).toBeInTheDocument()
        expect(screen.getByText('failed')).toBeInTheDocument()
        expect(screen.getByText(/failed to generate/i)).toBeInTheDocument()
      })
    })
  })

  describe('Task Operations', () => {
    it('应该运行任务', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks),
        http.post('/api/tasks/2/run', () => {
          return Response.json({ message: 'task started', status: 'queued' })
        })
      )

      renderWithRouter(<RepoDetail />)

      await waitFor(() => {
        const runButtons = screen.getAllByRole('button', { name: /run/i })
        expect(runButtons.length).toBeGreaterThan(0)
      })
    })

    it('应该重试任务', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks),
        http.post('/api/tasks/3/retry', () => {
          return Response.json({ message: 'task retry started', status: 'queued' })
        }))
      )

      renderWithRouter(<RepoDetail />)

      await waitFor(() => {
        const retryButtons = screen.getAllByRole('button', { name: /retry/i })
        expect(retryButtons.length).toBeGreaterThan(0)
      })
    })

    it('应该取消任务', async () => {
      const runningTask = { ...mockTasks[1], id: 4, status: 'running' as Task['status'] }
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => new Response(JSON.stringify([runningTask]), { headers: { 'Content-Type': 'application/json' }}),
        http.post('/api/tasks/4/cancel', () => {
          return Response.json({ message: 'task canceled', status: 'canceled' })
        }))
      )

      renderWithRouter(<RepoDetail />)

      await waitFor(() => {
        const cancelButton = screen.getByRole('button', { name: /cancel/i })
        expect(cancelButton).toBeInTheDocument()
      })
    })

    it('应该删除任务', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks),
        http.delete('/api/tasks/1', () => {
          return Response.json({ message: 'task deleted' })
        }))
      )

      renderWithRouter(<RepoDetail />)

      await waitFor(() => {
        const deleteButtons = screen.getAllByRole('button', { name: /delete/i })
        expect(deleteButtons.length).toBeGreaterThan(0)
      })
    })
  })

  describe('Repository Operations', () => {
    it('应该运行所有任务', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks),
        http.post('/api/repositories/1/run-all', () => {
          return Response.json({ message: 'all tasks started' })
        }))
      )

      renderWithRouter(<RepoDetail />)

      const runAllButton = screen.getByRole('button', { name: /run all/i })
      expect(runAllButton).toBeInTheDocument()
    })

    it('应该重新克隆仓库', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks),
        http.post('/api/repositories/1/clone', () => {
          return Response.json({ message: 'clone started' })
        }))
      )

      renderWithRouter(<RepoDetail />)

      const cloneButton = screen.getByRole('button', { name: /clone/i })
      expect(cloneButton).toBeInTheDocument()
    })

    it('应该删除仓库', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks),
        http.delete('/api/repositories/1', () => {
          return Response.json({ message: 'repository deleted' })
        }))
      )

      renderWithRouter(<RepoDetail />)

      const deleteButton = screen.getByRole('button', { name: /delete repository/i })
      expect(deleteButton).toBeInTheDocument()
    })
  })

  describe('Auto Refresh', () => {
    it('应该每3秒自动刷新数据', async () => {
      let callCount = 0

      server.use(
        http.get('/api/repositories/1', (req, res, ctx) => {
          callCount++
          return Response.json(mockRepository)
        }),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks)
      )

      renderWithRouter(<RepoDetail />)

      await waitFor(() => callCount === 1)

      // 等待第二次调用
      await waitFor(() => callCount >= 2, { timeout: 4000 })

      expect(callCount).toBeGreaterThanOrEqual(2)
    })
  })

  describe('Navigation', () => {
    it('应该渲染返回按钮', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks)
      )

      renderWithRouter(<RepoDetail />)

      const backButton = screen.getByRole('button', { name: /back/i })
      expect(backButton).toBeInTheDocument()
    })

    it('应该渲染文档导出按钮', async () => {
      server.use(
        http.get('/api/repositories/1', () => Response.json(mockRepository),
        http.get('/api/repositories/1/tasks', () => Response.json(mockTasks)
      )

      renderWithRouter(<RepoDetail />)

      const exportButton = screen.getByRole('button', { name: /export/i })
      expect(exportButton).toBeInTheDocument()
    })
  })
})

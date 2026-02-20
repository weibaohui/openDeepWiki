import { http } from 'msw'
import { HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import TaskMonitor from './TaskMonitor'
import type { Task } from '../../types'

// Mock 服务器
const server = setupServer(...handlers)

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

const mockMonitorData = {
  active_tasks: [
    {
      id: 1,
      title: 'Active Task 1',
      status: 'running',
      task_type: 'DocWrite',
      writer_name: 'DefaultWriter',
      repository_id: 1,
      repository: { id: 1, name: 'test-repo' } as any,
      started_at: new Date(Date.now() - 10000).toISOString(),
      completed_at: null,
    },
  ],
  recent_tasks: [
    {
      id: 2,
      title: 'Recent Task 1',
      status: 'completed',
      task_type: 'DocWrite',
      writer_name: 'DefaultWriter',
      repository_id: 1,
      repository: { id: 1, name: 'test-repo' } as any,
      started_at: new Date(Date.now() - 100000).toISOString(),
      completed_at: new Date().toISOString(),
      error_msg: '',
    },
  ],
  queue_status: {
    queue_length: 10,
    active_workers: 5,
    priority_length: 3,
    active_repos: 2,
  },
}

describe('TaskMonitor', () => {
  describe('Monitor Data', () => {
    it('应该渲染监控数据', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return Response.json(mockMonitorData)
        })
      )

      render(<TaskMonitor />)

      await waitFor(() => {
        expect(screen.getByText(/queue length/i)).toBeInTheDocument()
        expect(screen.getByText(/active workers/i)).toBeInTheDocument()
        expect(screen.getByText(/priority queue/i)).toBeInTheDocument()
        expect(screen.getByText(/active repos/i)).toBeInTheDocument()
      })
    })
  })

  describe('Active Tasks', () => {
    it('应该渲染活跃任务列表', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return Response.json(mockMonitorData)
        })
      )

      render(<TaskMonitor />)

      await waitFor(() => {
        expect(screen.getByText('Active Task 1')).toBeInTheDocument()
        expect(screen.getByText('running')).toBeInTheDocument()
      })
    })

    it('应该显示任务状态标签', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return Response.json(mockMonitorData)
        })
      )

      render(<TaskMonitor />)

      await waitFor(() => {
        const statusTags = screen.getAllByText('running')
        expect(statusTags.length).toBeGreaterThan(0)
      })
    })

    it('应该显示任务元信息', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return Response.json(mockMonitorData)
        })
      )

      render(<TaskMonitor />)

      await waitFor(() => {
        expect(screen.getByText('test-repo')).toBeInTheDocument()
        expect(screen.getByText('DocWrite')).toBeInTheDocument()
      })
    })

    it('应该支持取消任务', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return Response.json(mockMonitorData)
        }),
        http.post('/api/tasks/1/cancel', () => {
          return Response.json({ message: 'task canceled' })
        }))
      )

      render(<TaskMonitor />)

      await waitFor(async () => {
        const cancelButton = screen.getByRole('button', { name: /cancel/i })
        expect(cancelButton).toBeInTheDocument()
        await userEvent.click(cancelButton)
      })
    })
  })

  describe('Recent Tasks', () => {
    it('应该渲染最近任务列表', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return Response.json(mockMonitorData)
        })
      )

      render(<TaskMonitor />)

      await waitFor(() => {
        expect(screen.getByText('Recent Task 1')).toBeInTheDocument()
        expect(screen.getByText('completed')).toBeInTheDocument()
      })
    })
  })

  describe('Auto Refresh', () => {
    it('应该每5秒自动刷新', async () => {
      let callCount = 0

      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          callCount++
          return Response.json(mockMonitorData)
        })
      )

      render(<TaskMonitor />)

      await waitFor(() => callCount === 1)

      // 等待第二次调用
      await waitFor(() => callCount >= 2, { timeout: 6000 })

      expect(callCount).toBeGreaterThanOrEqual(2)
    })
  })

  describe('Manual Refresh', () => {
    it('应该支持手动刷新', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return Response.json(mockMonitorData)
        })
      )

      render(<TaskMonitor />)

      const refreshButton = screen.getByRole('button', { name: /refresh/i })
      await userEvent.click(refreshButton)

      await waitFor(() => {
        expect(screen.getByText(/queue length/i)).toBeInTheDocument()
      })
    })
  })

  describe('Error Handling', () => {
    it('应该处理获取数据失败', async () => {
      server.use(
        http.get('/api/tasks/monitor', (req, res, ctx) => {
          return res(ctx.status(500).json({ error: 'Server error' }))
        })
      )

      render(<TaskMonitor />)

      // 应该显示错误状态或者重试
      await waitFor(() => {
        const errorState = screen.queryByText(/error/i)
        expect(errorState).toBeInTheDocument()
      })
    })
  })
})

import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import TaskMonitor from './TaskMonitor'

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

const mockMonitorData = {
  active_tasks: [
    {
      id: 1,
      title: 'Active Task 1',
      status: 'running',
      task_type: 'DocWrite',
      writer_name: 'DefaultWriter',
      repository_id: 1,
      repository: { id: 1, name: 'test-repo' } as unknown,
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
      repository: { id: 1, name: 'test-repo' } as unknown,
      started_at: new Date(Date.now() - 100000).toISOString(),
      completed_at: new Date().toISOString(),
    },
  ],
}

function renderWithRouter(component: React.ReactNode) {
  return render(<MemoryRouter>{component}</MemoryRouter>)
}

describe('TaskMonitor', () => {
  describe('Active Tasks', () => {
    it('应该渲染活跃任务列表', async () => {
      server.use(
        http.get('/api/tasks/monitor', () => {
          return HttpResponse.json(mockMonitorData)
        })
      )

      renderWithRouter(<TaskMonitor />)

      await waitFor(() => {
        expect(screen.getByText('Active Task 1')).toBeInTheDocument()
        expect(screen.getByText('Recent Task 1')).toBeInTheDocument()
      })
    })

    it('应该显示任务状态标签', async () => {
      server.use(
        http.get('/api/tasks/monitor', () => {
          return HttpResponse.json(mockMonitorData)
        })
      )

      renderWithRouter(<TaskMonitor />)

      await waitFor(() => {
        expect(screen.getByText('running')).toBeInTheDocument()
        expect(screen.getByText('completed')).toBeInTheDocument()
      })
    })

  })

  describe('Task Operations', () => {
    it('应该取消任务', async () => {
      server.use(
        http.get('/api/tasks/monitor', () => {
          return HttpResponse.json(mockMonitorData)
        }),
        http.post('/api/tasks/1/cancel', () => {
          return HttpResponse.json({ message: 'task canceled' })
        })
      )

      renderWithRouter(<TaskMonitor />)

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
        http.get('/api/tasks/monitor', () => {
          return HttpResponse.json(mockMonitorData)
        })
      )

      renderWithRouter(<TaskMonitor />)

      await waitFor(() => {
        expect(screen.getByText('Recent Task 1')).toBeInTheDocument()
      })
    })
  })

  describe('Auto Refresh', () => {
    it('应该每5秒自动刷新数据', async () => {
      let callCount = 0

      server.use(
        http.get('/api/tasks/monitor', () => {
          callCount++
          return HttpResponse.json(mockMonitorData)
        })
      )

      renderWithRouter(<TaskMonitor />)

      await waitFor(() => callCount === 1)

      // 等待第二次调用
      await waitFor(() => callCount >= 2, { timeout: 6000 })

      expect(callCount).toBeGreaterThanOrEqual(2)
    })
  })

  describe('Navigation', () => {
    it('应该渲染设置按钮', async () => {
      server.use(
        http.get('/api/tasks/monitor', () => {
          return HttpResponse.json(mockMonitorData)
        })
      )

      renderWithRouter(<TaskMonitor />)

      const settingsButton = screen.getByRole('button', { name: /settings/i })
      expect(settingsButton).toBeInTheDocument()
    })
  })
})

import { http } from 'msw'
import { HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { MemoryRouter, Route, Routes, useParams } from 'react-router-dom'
import UserRequestList from './UserRequestList'
import type { UserRequest } from '../types'

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

const mockRequests: UserRequest[] = [
  {
    id: 1,
    content: 'Request 1',
    status: 'pending',
    repository_id: 1,
    created_at: new Date().toISOString(),
  },
  {
    id: 2,
    content: 'Request 2',
    status: 'completed',
    repository_id: 1,
    created_at: new Date().toISOString(),
  },
]

function renderWithRouter(component: React.ReactNode) {
  return render(
    <MemoryRouter initialEntries={['/repo/1/user-requests']}>
      <Routes>
        <Route path="/repo/:id/user-requests" element={component} />
      </Routes>
    </MemoryRouter>
  )
}

describe('UserRequestList', () => {
  describe('Request List', () => {
    it('应该渲染用户需求列表', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: mockRequests,
              total: 2,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      await waitFor(() => {
        expect(screen.getByText('Request 1')).toBeInTheDocument()
        expect(screen.getByText('Request 2')).toBeInTheDocument()
      })
    })

    it('应该显示需求状态', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: mockRequests,
              total: 2,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      await waitFor(() => {
        expect(screen.getByText('pending')).toBeInTheDocument()
        expect(screen.getByText('completed')).toBeInTheDocument()
      })
    })

    it('应该显示创建时间', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: mockRequests,
              total: 2,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      await waitFor(() => {
        const createdAtLabels = screen.getAllByText(/created at/i)
        expect(createdAtLabels.length).toBeGreaterThan(0)
      })
    })
  })

  describe('Delete Request', () => {
    it('应该删除用户需求', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: mockRequests,
              total: 2,
            },
          })
        }),
        http.delete('/api/user-requests/1', () => {
          return Response.json({ code: 0, message: 'deleted successfully' })
        }))
      )

      renderWithRouter(<UserRequestList />)

      await waitFor(() => {
        const deleteButton = screen.getAllByRole('button', { name: /delete/i })
        expect(deleteButton.length).toBeGreaterThan(0)

        await userEvent.click(deleteButton[0])

        expect(screen.getByRole('button', { name: /ok/i })).toBeInTheDocument()
        await userEvent.click(screen.getByRole('button', { name: /ok/i }))

        await waitFor(() => {
          expect(screen.getByText(/deleted successfully/i)).toBeInTheDocument()
        })
      })
    })

    it('应该显示删除确认对话框', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: mockRequests,
              total: 2,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      const deleteButton = screen.getAllByRole('button', { name: /delete/i })
      expect(deleteButton.length).toBeGreaterThan(0)
    })
  })

  describe('Status Filter', () => {
    it('应该支持按状态过滤', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: [mockRequests[0]],
              total: 1,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      const statusFilter = screen.getByPlaceholderText(/filter by status/i)
      await userEvent.click(statusFilter)
      await userEvent.click(screen.getByText('pending'))

      await waitFor(() => {
        expect(screen.getByText('Request 1')).toBeInTheDocument()
      })
    })

    it('应该支持清除过滤', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: mockRequests,
              total: 2,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      const statusFilter = screen.getByPlaceholderText(/filter by status/i)
      await userEvent.click(statusFilter)

      const clearButton = screen.getByRole('button', { name: /clear/i })
      await userEvent.click(clearButton)

      await waitFor(() => {
        expect(screen.getByText('Request 1')).toBeInTheDocument()
        expect(screen.getByText('Request 2')).toBeInTheDocument()
      })
    })
  })

  describe('Pagination', () => {
    it('应该支持分页', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: [mockRequests[0]],
              total: 50,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      await waitFor(() => {
        expect(screen.getByText('Request 1')).toBeInTheDocument()
      })
    })
  })

  describe('Refresh', () => {
    it('应该支持手动刷新', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: mockRequests,
              total: 2,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      const refreshButton = screen.getByRole('button', { name: /refresh/i })
      await userEvent.click(refreshButton)

      await waitFor(() => {
        expect(screen.getByText('Request 1')).toBeInTheDocument()
      })
    })
  })

  describe('Empty State', () => {
    it('应该显示空状态', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: [],
              total: 0,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      await waitFor(() => {
        expect(screen.getByText(/no user requests found/i)).toBeInTheDocument()
      })
    })
  })

  describe('Navigation', () => {
    it('应该渲染返回按钮', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: [],
              total: 0,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      const backButton = screen.getByRole('button', { name: /back/i })
      expect(backButton).toBeInTheDocument()
    })
  })

  describe('Header', () => {
    it('应该渲染标题', async () => {
      server.use(
        http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
          return Response.json({
            code: 0,
            data: {
              list: [],
              total: 0,
            },
          })
        })
      )

      renderWithRouter(<UserRequestList />)

      await waitFor(() => {
        expect(screen.getByText(/user requests/i)).toBeInTheDocument()
      })
    })
  })
})

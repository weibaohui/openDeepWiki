import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { MemoryRouter } from 'react-router-dom'
import Home from './Home'
import type { Repository } from '../types'

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

// 测试工具函数
function renderWithRouter(component: React.ReactNode) {
  return render(<MemoryRouter initialEntries={['/']}>{component}</MemoryRouter>)
}

describe('Home', () => {
  const mockRepositories: Repository[] = [
    {
      id: 1,
      name: 'test-repo-1',
      url: 'https://github.com/test/repo1',
      status: 'ready',
      created_at: new Date().toISOString(),
      size_mb: 10.5,
      clone_branch: 'main',
    },
    {
      id: 2,
      name: 'test-repo-2',
      url: 'https://github.com/test/repo2',
      status: 'pending',
      created_at: new Date().toISOString(),
      size_mb: 5.2,
      clone_branch: 'main',
    },
  ]

  describe('Repository List', () => {
    it('应该渲染空状态', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        })
      )

      renderWithRouter(<Home />)

      await waitFor(() => {
        expect(screen.getByText(/no repositories/i)).toBeInTheDocument()
      })
    })

    it('应该渲染仓库列表', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json(mockRepositories)
        })
      )

      renderWithRouter(<Home />)

      await waitFor(() => {
        expect(screen.getByText('test-repo-1')).toBeInTheDocument()
        expect(screen.getByText('test-repo-2')).toBeInTheDocument()
      })
    })

    it('应该显示仓库状态标签', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json(mockRepositories)
        })
      )

      renderWithRouter(<Home />)

      await waitFor(() => {
        expect(screen.getByText('ready')).toBeInTheDocument()
        expect(screen.getByText('pending')).toBeInTheDocument()
      })
    })
  })

  describe('Add Repository', () => {
    it('应该打开添加模态框', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        })
      )

      renderWithRouter(<Home />)

      const addButton = screen.getByRole('button', { name: /add new/i })
      await userEvent.click(addButton)

      expect(screen.getByText(/add repository/i)).toBeInTheDocument()
    })

    it('应该成功添加仓库', async () => {
      const newRepo = {
        id: 3,
        name: 'new-repo',
        url: 'https://github.com/test/new-repo',
        status: 'pending',
        created_at: new Date().toISOString(),
        size_mb: 0,
        clone_branch: '',
      }

      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        }),
        http.post('/api/repositories', () => {
          return HttpResponse.json(newRepo, { status: 201 })
        })
      )

      renderWithRouter(<Home />)

      const addButton = screen.getByRole('button', { name: /add new/i })
      await userEvent.click(addButton)

      const urlInput = screen.getByPlaceholderText(/github/i)
      await userEvent.type(urlInput, 'https://github.com/test/new-repo')

      const okButton = screen.getByRole('button', { name: /ok/i })
      await userEvent.click(okButton)

      await waitFor(() => {
        expect(screen.getByText('new-repo')).toBeInTheDocument()
      })
    })

    it('应该处理重复URL错误', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        }),
        http.post('/api/repositories', () => {
          return HttpResponse.json({ error: 'repository already exists' }, { status: 409 })
        })
      )

      renderWithRouter(<Home />)

      const addButton = screen.getByRole('button', { name: /add new/i })
      await userEvent.click(addButton)

      const urlInput = screen.getByPlaceholderText(/github/i)
      await userEvent.type(urlInput, 'https://github.com/test/existing')

      const okButton = screen.getByRole('button', { name: /ok/i })
      await userEvent.click(okButton)

      await waitFor(() => {
        expect(screen.getByText(/already exists/i)).toBeInTheDocument()
      })
    })

    it('应该验证必填字段', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        })
      )

      renderWithRouter(<Home />)

      const addButton = screen.getByRole('button', { name: /add new/i })
      await userEvent.click(addButton)

      const okButton = screen.getByRole('button', { name: /ok/i })
      await userEvent.click(okButton)

      await waitFor(() => {
        expect(screen.getByText(/url is required/i)).toBeInTheDocument()
      })
    })
  })

  describe('Search Filter', () => {
    it('应该按名称搜索仓库', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json(mockRepositories)
        })
      )

      renderWithRouter(<Home />)

      const searchInput = screen.getByPlaceholderText(/search/i)
      await userEvent.type(searchInput, 'repo-1')

      await waitFor(() => {
        expect(screen.getByText('test-repo-1')).toBeInTheDocument()
      })
      expect(screen.queryByText('test-repo-2')).not.toBeInTheDocument()
    })

    it('应该显示无搜索结果', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json(mockRepositories)
        })
      )

      renderWithRouter(<Home />)

      const searchInput = screen.getByPlaceholderText(/search/i)
      await userEvent.type(searchInput, 'nonexistent')

      await waitFor(() => {
        expect(screen.getByText(/no matching/i)).toBeInTheDocument()
      })
    })
  })

  describe('Navigation', () => {
    it('应该渲染导航栏', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        })
      )

      renderWithRouter(<Home />)

      expect(screen.getByRole('link', { name: /opendeepwiki/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /sync/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /settings/i })).toBeInTheDocument()
    })

    it('应该渲染语言切换器', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        })
      )

      renderWithRouter(<Home />)

      expect(screen.getByRole('button', { name: /language/i })).toBeInTheDocument()
    })

    it('应该渲染主题切换器', async () => {
      server.use(
        http.get('/api/repositories', () => {
          return HttpResponse.json([])
        })
      )

      renderWithRouter(<Home />)

      expect(screen.getByRole('button', { name: /theme/i })).toBeInTheDocument()
    })
  })
})

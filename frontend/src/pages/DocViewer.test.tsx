import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import DocViewer from './DocViewer'
import type { Document, Task, Repository, TaskUsage } from '../types'

import userEvent from '@testing-library/user-event'

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
  useParams: () => ({ id: '1', docId: '1' }),
}))
const mockRepository: Repository = {
  id: 1,
  name: 'test-repo',
  url: 'https://github.com/test/repo',
  status: 'ready',
  created_at: new Date().toISOString(),
  size_mb: 10.5,
  clone_branch: 'main',
  clone_commit_id: 'abc123def',
}
const mockDocument: Document = {
  id: 1,
  title: 'Test Document',
  content: '# Test Content\n\nThis is a test document.',
  repository_id: 1,
  path: 'docs/test.md',
  task_id: 1,
  version: 1,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
}
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
const mockTokenUsage: TaskUsage = {
  task_id: 1,
  total_tokens: 1000,
  prompt_tokens: 600,
  completion_tokens: 400,
  api_key_name: 'test-key',
}
function renderWithRouter(component: React.ReactNode) {
  return render(
    <MemoryRouter initialEntries={['/repo/1/doc/1']}>
      <Routes>
        <Route path="/repo/:id/doc/:docId" element={component} />
      </Routes>
    </MemoryRouter>
  )
}
describe('DocViewer', () => {
  describe('Document View', () => {
    it('应该渲染文档内容', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 4.5,
          rating_count: 10,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: mockTokenUsage,
        }))
      )
      renderWithRouter(<DocViewer />)
      await waitFor(() => {
        expect(screen.getByText('Test Content')).toBeInTheDocument()
        expect(screen.getByText('This is a test document.')).toBeInTheDocument()
      })
    })
    it('应该显示文档元信息', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        }))
      )
      renderWithRouter(<DocViewer />)
      await waitFor(() => {
        expect(screen.getByText(/created at/i)).toBeInTheDocument()
        expect(screen.getByText(/updated at/i)).toBeInTheDocument()
        expect(screen.getByText('main')).toBeInTheDocument()
        expect(screen.getByText('abc123')).toBeInTheDocument()
      })
    })
  })
  describe('Document Edit', () => {
    it('应该进入编辑模式', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        }))
      )
      renderWithRouter(<DocViewer />)
      const editButton = screen.getByRole('button', { name: /edit/i })
      await userEvent.click(editButton)
      expect(screen.getByDisplayValue('# Test Content\n\nThis is a test document.')).toBeInTheDocument()
    })
    it('应该保存文档', async () => {
      const updatedDoc = { ...mockDocument, content: '# Updated Content' }
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        })),
        http.put('/api/documents/1', () => HttpResponse.json(updatedDoc))
      )
      renderWithRouter(<DocViewer />)
      const editButton = screen.getByRole('button', { name: /edit/i })
      await userEvent.click(editButton)
      const editor = screen.getByDisplayValue('# Test Content\n\nThis is a test document.')
      await userEvent.clear(editor)
      await userEvent.type(editor, '# Updated Content')
      const saveButton = screen.getByRole('button', { name: /save/i })
      await userEvent.click(saveButton)
      await waitFor(() => {
        expect(screen.getByText(/document saved/i)).toBeInTheDocument()
      })
    })
    it('应该取消编辑', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        }))
      )
      renderWithRouter(<DocViewer />)
      const editButton = screen.getByRole('button', { name: /edit/i })
      await userEvent.click(editButton)
      const cancelButton = screen.getByRole('button', { name: /cancel/i })
      await userEvent.click(cancelButton)
      expect(screen.queryByDisplayValue('# Updated Content')).not.toBeInTheDocument()
    })
  })
  describe('Document Rating', () => {
    it('应该显示评分信息', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 4.5,
          rating_count: 10,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: mockTokenUsage,
        }))
      )
      renderWithRouter(<DocViewer />)
      await waitFor(() => {
        expect(screen.getByText(/average rating/i)).toBeInTheDocument()
        expect(screen.getByText(/your rating/i)).toBeInTheDocument()
      })
    })
    it('应该提交评分', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 4.0,
          rating_count: 10,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: mockTokenUsage,
        }),
        http.post('/api/documents/1/ratings', () => HttpResponse.json({ average_score: 4.5, rating_count: 11 }))
      )
      renderWithRouter(<DocViewer />)
      await waitFor(() => {
        const starButtons = screen.getAllByRole('button')
        expect(starButtons.length).toBeGreaterThan(0)
      })
      // 查找并点击5星按钮
      const starButton = screen.getAllByLabelText('5 stars')[0]
      await userEvent.click(starButton)
      await waitFor(() => {
        expect(screen.getByText(/rating submitted/i)).toBeInTheDocument()
      })
    })
  })
  describe('Document Versions', () => {
    it('应该打开版本抽屉', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        }))
      )
      renderWithRouter(<DocViewer />)
      const versionsButton = screen.getByRole('button', { name: /versions/i })
      await userEvent.click(versionsButton)
      expect(screen.getByText(/versions/i)).toBeInTheDocument()
    })
    it('应该显示文档版本列表', async () => {
      const mockVersions = [
        { ...mockDocument, id: 2, version: 2, updated_at: new Date().toISOString() },
        { ...mockDocument, id: 1, version: 1, updated_at: new Date(Date.now() - 86400000).toISOString() },
      ]
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        })),
        http.get('/api/documents/1/versions', () => HttpResponse.json(mockVersions))
      )
      renderWithRouter(<DocViewer />)
      const versionsButton = screen.getByRole('button', { name: /versions/i })
      await userEvent.click(versionsButton)
      await waitFor(() => {
        expect(screen.getByText('Version 2')).toBeInTheDocument()
        expect(screen.getByText('Version 1')).toBeInTheDocument()
      })
    })
  })
  describe('Document Export', () => {
    it('应该导出ZIP', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        })),
        http.get('/api/repositories/1/documents/export', () => {
          return new Response(new Uint8Array([0x50, 0x4b, 0x03, 0x04]), {
            headers: {
              'Content-Type': 'application/zip',
              'Content-Disposition': 'attachment; filename="test-docs.zip"'
            }
          })
        })
      )
      renderWithRouter(<DocViewer />)
      const exportButton = screen.getByRole('button', { name: /export/i })
      expect(exportButton).toBeInTheDocument()
    })
    it('应该导出PDF', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        })),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null,
        })),
        http.get('/api/repositories/1/export-pdf', () => {
          return new Response(new Uint8Array([0x25, 0x50, 0x44, 0x46]), {
            headers: {
              'Content-Type': 'application/pdf',
              'Content-Disposition': 'attachment; filename="test-docs.pdf"'
            }
          })
        })
      )
      renderWithRouter(<DocViewer />)
      const exportMenuButton = screen.getByRole('button', { name: /export/i })
      await userEvent.click(exportMenuButton)
      expect(screen.getByText(/export pdf/i)).toBeInTheDocument()
    })
  })
  describe('Token Usage Display', () => {
    it('应该显示Token用量信息', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        }),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: mockTokenUsage,
        }))
      )
      renderWithRouter(<DocViewer />)
      await waitFor(() => {
        expect(screen.getByText(/total tokens/i)).toBeInTheDocument()
        expect(screen.getByText(/input tokens/i)).toBeInTheDocument()
        expect(screen.getByText(/output tokens/i)).toBeInTheDocument()
        expect(screen.getByText('1000')).toBeInTheDocument()
        expect(screen.getByText('600')).toBeInTheDocument()
        expect(screen.getByText('400')).toBeInTheDocument()
        expect(screen.getByText('test-key')).toBeInTheDocument()
      })
    })
  })
  describe('Sidebar Navigation', () => {
    it('应该渲染文档侧边栏', async () => {
      const mockDocs = [
        mockDocument,
        { ...mockDocument, id: 2, title: 'Document 2' },
      ]
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json(mockDocs)),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        }),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null
        })
      )
      renderWithRouter(<DocViewer />)
      await waitFor(() => {
        expect(screen.getByText('Test Document')).toBeInTheDocument()
        expect(screen.getByText('Document 2')).toBeInTheDocument()
      })
    })
    it('应该渲染返回按钮', async () => {
      server.use(
        http.get('/api/repositories/1', () => HttpResponse.json(mockRepository)),
        http.get('/api/repositories/1/documents', () => HttpResponse.json([mockDocument])),
        http.get('/api/repositories/1/tasks', () => HttpResponse.json(mockTasks)),
        http.get('/api/documents/1', () => HttpResponse.json(mockDocument)),
        http.get('/api/documents/1/ratings/stats', () => HttpResponse.json({
          average_score: 0,
          rating_count: 0,
        }),
        http.get('/api/documents/1/token-usage', () => HttpResponse.json({
          code: 0,
          data: null
        })
      )
      renderWithRouter(<DocViewer />)
      const backButton = screen.getByRole('button', { name: /back/i })
      expect(backButton).toBeInTheDocument()
    })
  })
})

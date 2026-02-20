import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import MarkdownRender from './MarkdownRender'

// Mock @uiw/react-md-editor
vi.mock('@uiw/react-md-editor', () => ({
  default: ({ value, preview }: any) => {
    if (preview) {
      return <div data-testid="markdown-preview" dangerouslySetInnerHTML={{ __html: value }} />
    }
    return <div data-testid="markdown-editor">{value}</div>
  },
}))

// Mock react-router-dom
vi.mock('react-router-dom', () => ({
  useLocation: () => ({ pathname: '/test' }),
}))

// Mock useAppConfig
vi.mock('@/context/AppConfigContext', () => ({
  useAppConfig: () => ({
    t: (key: string, fallback?: string) => fallback || key,
    themeMode: 'light',
    locale: 'zh-CN',
    setLocale: vi.fn(),
    setThemeMode: vi.fn(),
  }),
}))

describe('MarkdownRender', () => {
  const testContent = '# Test Heading\n\nThis is a test paragraph.\n\n- List item 1\n- List item 2'

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('应该渲染Markdown内容', () => {
    render(<MarkdownRender content={testContent} />)

    expect(screen.getByText('Test Heading')).toBeInTheDocument()
    expect(screen.getByText('This is a test paragraph.')).toBeInTheDocument()
    expect(screen.getByText('List item 1')).toBeInTheDocument()
    expect(screen.getByText('List item 2')).toBeInTheDocument()
  })

  it('应该渲染标题', () => {
    render(<MarkdownRender content="# Main Heading" />)

    const heading = screen.getByRole('heading', { level: 1 })
    expect(heading).toBeInTheDocument()
    expect(heading).toHaveTextContent('Main Heading')
  })

  it('应该渲染代码块', () => {
    const codeContent = '```javascript\nconst x = 1;\nconsole.log(x);\n```'
    render(<MarkdownRender content={codeContent} />)

    const codeElement = screen.getByText('const x = 1;')
    expect(codeElement).toBeInTheDocument()
  })

  it('应该渲染链接', () => {
    const linkContent = '[GitHub](https://github.com)'
    render(<MarkdownRender content={linkContent} />)

    const link = screen.getByRole('link', { name: 'GitHub' })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute('href', 'https://github.com')
  })

  it('应该处理空内容', () => {
    render(<MarkdownRender content="" />)

    const container = screen.getByTestId('markdown-preview')
    expect(container).toBeInTheDocument()
  })

  it('应该应用自定义样式', () => {
    const customStyle = { color: 'red' }
    render(<MarkdownRender content={testContent} style={customStyle} />)

    const container = screen.getByTestId('markdown-preview')
    expect(container).toBeInTheDocument()
  })

  it('应该渲染表格', () => {
    const tableContent = '| Header 1 | Header 2 |\n|----------|----------|\n| Cell 1   | Cell 2   |'
    render(<MarkdownRender content={tableContent} />)

    expect(screen.getByText('Header 1')).toBeInTheDocument()
    expect(screen.getByText('Header 2')).toBeInTheDocument()
    expect(screen.getByText('Cell 1')).toBeInTheDocument()
    expect(screen.getByText('Cell 2')).toBeInTheDocument()
  })

  it('应该渲染引用', () => {
    const quoteContent = '> This is a quote'
    render(<MarkdownRender content={quoteContent} />)

    expect(screen.getByText('This is a quote')).toBeInTheDocument()
  })

  it('应该渲染加粗文本', () => {
    const boldContent = '**Bold Text**'
    render(<MarkdownRender content={boldContent} />)

    const boldText = screen.getByText('Bold Text')
    expect(boldText).toBeInTheDocument()
  })

  it('应该渲染斜体文本', () => {
    const italicContent = '*Italic Text*'
    render(<MarkdownRender content={italicContent} />)

    const italicText = screen.getByText('Italic Text')
    expect(italicText).toBeInTheDocument()
  })
})

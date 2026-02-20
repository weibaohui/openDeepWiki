import { describe, it, expect, beforeEach } from 'vitest'
import { http } from 'msw'
import { setupServer } from 'msw/node'
import { repositoryApi, taskApi, documentApi, userRequestApi } from './api'
import type { Repository, Task, Document } from '../types'

// Mock 服务器
const server = setupServer()

beforeAll(() => server.listen())
beforeEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('API - repositoryApi', () => {
  const mockRepos: Repository[] = [
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
      status: 'ready',
      created_at: new Date().toISOString(),
      size_mb: 5.2,
      clone_branch: 'main',
    },
  ]

  it('应该获取仓库列表', async () => {
    server.use(
      http.get('/api/repositories', (req, res, ctx) => {
        return res(ctx.json(mockRepos))
      })
    )

    const response = await repositoryApi.list()
    expect(response.status).toBe(200)
    expect(response.data).toEqual(mockRepos)
  })

  it('应该获取仓库详情', async () => {
    server.use(
      http.get('/api/repositories/1', (req, res, ctx) => {
        return res(ctx.json(mockRepos[0]))
      })
    )

    const response = await repositoryApi.get(1)
    expect(response.status).toBe(200)
    expect(response.data.id).toBe(1)
    expect(response.data.name).toBe('test-repo-1')
  })

  it('应该创建仓库', async () => {
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
      http.post('/api/repositories', (req, res, ctx) => {
        return res(ctx.status(201).json(newRepo))
      })
    )

    const response = await repositoryApi.create('https://github.com/test/new-repo')
    expect(response.status).toBe(201)
    expect(response.data.id).toBe(3)
  })

  it('应该删除仓库', async () => {
    server.use(
      http.delete('/api/repositories/1', (req, res, ctx) => {
        return res(ctx.json({ message: 'deleted successfully' }))
      })
    )

    const response = await repositoryApi.delete(1)
    expect(response.status).toBe(200)
  })
})

describe('API - taskApi', () => {
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
  ]

  it('应该获取仓库任务列表', async () => {
    server.use(
      http.get('/api/repositories/1/tasks', (req, res, ctx) => {
        return res(ctx.json(mockTasks))
      })
    )

    const response = await taskApi.getByRepository(1)
    expect(response.status).toBe(200)
    expect(response.data).toEqual(mockTasks)
  })

  it('应该获取任务统计', async () => {
    const mockStats = {
      completed: 10,
      pending: 5,
      failed: 2,
    }

    server.use(
      http.get('/api/repositories/1/tasks/stats', (req, res, ctx) => {
        return res(ctx.json(mockStats))
      })
    )

    const response = await taskApi.getStats(1)
    expect(response.status).toBe(200)
    expect(response.data.completed).toBe(10)
  })

  it('应该运行任务', async () => {
    server.use(
      http.post('/api/tasks/1/run', (req, res, ctx) => {
        return res(ctx.json({ message: 'task started', status: 'queued' }))
      })
    )

    const response = await taskApi.run(1)
    expect(response.status).toBe(200)
  })

  it('应该重试任务', async () => {
    server.use(
      http.post('/api/tasks/1/retry', (req, res, ctx) => {
        return res(ctx.json({ message: 'task retry started', status: 'queued' }))
      })
    )

    const response = await taskApi.retry(1)
    expect(response.status).toBe(200)
  })

  it('应该取消任务', async () => {
    server.use(
      http.post('/api/tasks/1/cancel', (req, res, ctx) => {
        return res(ctx.json({ message: 'task canceled', status: 'canceled' }))
      })
    )

    const response = await taskApi.cancel(1)
    expect(response.status).toBe(200)
  })

  it('应该删除任务', async () => {
    server.use(
      http.delete('/api/tasks/1', (req, res, ctx) => {
        return res(ctx.json({ message: 'task deleted' }))
      })
    )

    const response = await taskApi.delete(1)
    expect(response.status).toBe(200)
  })

  it('应该获取监控数据', async () => {
    const mockMonitorData = {
      active_tasks: [mockTasks[0]],
      recent_tasks: [mockTasks[0]],
      queue_status: {
        queue_length: 10,
        active_workers: 5,
      },
    }

    server.use(
      http.get('/api/tasks/monitor', (req, res, ctx) => {
        return res(ctx.json(mockMonitorData))
      })
    )

    const response = await taskApi.monitor()
    expect(response.status).toBe(200)
    expect(response.data).toBeDefined()
  })
})

describe('API - documentApi', () => {
  const mockDocs: Document[] = [
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

  it('应该获取仓库文档列表', async () => {
    server.use(
      http.get('/api/repositories/1/documents', (req, res, ctx) => {
        return res(ctx.json(mockDocs))
      })
    )

    const response = await documentApi.getByRepository(1)
    expect(response.status).toBe(200)
    expect(response.data).toEqual(mockDocs)
  })

  it('应该获取文档详情', async () => {
    server.use(
      http.get('/api/documents/1', (req, res, ctx) => {
        return res(ctx.json(mockDocs[0]))
      })
    )

    const response = await documentApi.get(1)
    expect(response.status).toBe(200)
    expect(response.data.id).toBe(1)
  })

  it('应该更新文档', async () => {
    const updatedDoc = { ...mockDocs[0], content: '# Updated Content' }

    server.use(
      http.put('/api/documents/1', (req, res, ctx) => {
        return res(ctx.json(updatedDoc))
      })
    )

    const response = await documentApi.update(1, '# Updated Content')
    expect(response.status).toBe(200)
    expect(response.data.content).toBe('# Updated Content')
  })

  it('应该获取文档版本列表', async () => {
    const mockVersions = [
      mockDocs[0],
      { ...mockDocs[0], id: 2, version: 2 },
    ]

    server.use(
      http.get('/api/documents/1/versions', (req, res, ctx) => {
        return res(ctx.json(mockVersions))
      })
    )

    const response = await documentApi.getVersions(1)
    expect(response.status).toBe(200)
    expect(response.data).toHaveLength(2)
  })

  it('应该提交评分', async () => {
    const mockStats = { average_score: 5.0, rating_count: 1 }

    server.use(
      http.post('/api/documents/1/ratings', (req, res, ctx) => {
        return res(ctx.json(mockStats))
      })
    )

    const response = await documentApi.submitRating(1, 5)
    expect(response.status).toBe(200)
    expect(response.data.average_score).toBe(5.0)
  })

  it('应该获取评分统计', async () => {
    const mockStats = { average_score: 4.5, rating_count: 10 }

    server.use(
      http.get('/api/documents/1/ratings/stats', (req, res, ctx) => {
        return res(ctx.json(mockStats))
      })
    )

    const response = await documentApi.getRatingStats(1)
    expect(response.status).toBe(200)
    expect(response.data.average_score).toBe(4.5)
  })

  it('应该导出ZIP', async () => {
    server.use(
      http.get('/api/repositories/1/documents/export', (req, res, ctx) => {
        return res(
          ctx.set('Content-Type', 'application/zip'),
          ctx.set('Content-Disposition', 'attachment; filename="test-docs.zip"'),
          ctx.body(new Uint8Array([0x50, 0x4b, 0x03, 0x04]))
        )
      })
    )

    const response = await documentApi.export(1)
    expect(response.status).toBe(200)
    expect(response.headers['content-type']).toBe('application/zip')
  })

  it('应该导出PDF', async () => {
    server.use(
      http.get('/api/repositories/1/export-pdf', (req, res, ctx) => {
        return res(
          ctx.set('Content-Type', 'application/pdf'),
          ctx.set('Content-Disposition', 'attachment; filename="test-docs.pdf"'),
          ctx.body(new Uint8Array([0x25, 0x50, 0x44, 0x46]))
        )
      })
    )

    const response = await documentApi.exportPdf(1)
    expect(response.status).toBe(200)
    expect(response.headers['content-type']).toBe('application/pdf')
  })
})

describe('API - userRequestApi', () => {
  it('应该创建用户需求', async () => {
    const mockRequest = {
      id: 1,
      content: 'test request',
      status: 'pending',
      repository_id: 1,
      created_at: new Date().toISOString(),
    }

    server.use(
      http.post('/api/repositories/1/user-requests', (req, res, ctx) => {
        return res(ctx.json(mockRequest))
      })
    )

    const response = await userRequestApi.create(1, 'test request')
    expect(response.status).toBe(200)
    expect(response.data.id).toBe(1)
  })

  it('应该获取用户需求列表', async () => {
    const mockList = {
      list: [
        {
          id: 1,
          content: 'request 1',
          status: 'pending',
          created_at: new Date().toISOString(),
        },
      ],
      total: 1,
    }

    server.use(
      http.get('/api/repositories/1/user-requests', (req, res, ctx) => {
        return res(ctx.json(mockList))
      })
    )

    const response = await userRequestApi.list(1, { page: 1, page_size: 20 })
    expect(response.status).toBe(200)
    expect(response.data.list).toHaveLength(1)
  })

  it('应该删除用户需求', async () => {
    server.use(
      http.delete('/api/user-requests/1', (req, res, ctx) => {
        return res(ctx.json({ message: 'deleted successfully' }))
      })
    )

    const response = await userRequestApi.delete(1)
    expect(response.status).toBe(200)
  })

  it('应该更新需求状态', async () => {
    server.use(
      http.patch('/api/user-requests/1/status', (req, res, ctx) => {
        return res(ctx.json({ message: 'status updated successfully' }))
      })
    )

    const response = await userRequestApi.updateStatus(1, 'completed')
    expect(response.status).toBe(200)
  })
})

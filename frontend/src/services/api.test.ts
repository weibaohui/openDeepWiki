/**
 * API Key API 测试
 * 测试 API Key 相关的 API 调用
 */

import { describe, it, expect, beforeAll, afterEach, afterAll } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import { apiKeyApi } from './api'

// API 类型定义
interface APIKey {
  id: number
  name: string
  provider: string
  base_url: string
  api_key: string
  model: string
  priority: number
  status: 'enabled' | 'disabled' | 'unavailable'
  request_count: number
  error_count: number
  last_used_at?: string
  rate_limit_reset_at?: string
  created_at: string
  updated_at: string
}

interface APIKeyStats {
  total_count: number
  enabled_count: number
  disabled_count: number
  unavailable_count: number
  total_requests: number
  total_errors: number
}

// Mock 服务器
const server = setupServer()

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('apiKeyApi', () => {
  const mockApiKey: APIKey = {
    id: 1,
    name: 'test-key',
    provider: 'openai',
    base_url: 'https://api.openai.com/v1',
    api_key: 'sk-test123456789',
    model: 'gpt-4',
    priority: 10,
    status: 'enabled',
    request_count: 100,
    error_count: 5,
    created_at: new Date().toISOString(),
    updated_at: new Date().toISOString(),
  }

  describe('list', () => {
    it('应该成功获取 API Key 列表', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({
            data: [mockApiKey],
            total: 1
          }, { status: 200 })
        })
      )

      const response = await apiKeyApi.list()

      expect(response.status).toBe(200)
      const data = response.data
      expect(data.data).toHaveLength(1)
      expect(data.total).toBe(1)
      expect(data.data[0].name).toBe('test-key')
    })

    it('当 API 返回错误时应该抛出异常', async () => {
      server.use(
        http.get('/api/api-keys', () => {
          return HttpResponse.json({ error: 'Server error' }, { status: 500 })
        })
      )

      await expect(apiKeyApi.list()).rejects.toThrow()
    })
  })

  describe('get', () => {
    it('应该成功获取指定 ID 的 API Key', async () => {
      server.use(
        http.get('/api/api-keys/1', () => {
          return HttpResponse.json(mockApiKey, { status: 200 })
        })
      )

      const response = await apiKeyApi.get(1)

      expect(response.status).toBe(200)
      expect(response.data.name).toBe('test-key')
      expect(response.data.id).toBe(1)
    })

    it('当 API Key 不存在时应该返回 404', async () => {
      server.use(
        http.get('/api/api-keys/999', () => {
          return HttpResponse.json({ error: 'Not found' }, { status: 404 })
        })
      )

      await expect(apiKeyApi.get(999)).rejects.toThrow()
    })
  })

  describe('create', () => {
    it('应该成功创建 API Key', async () => {
      const newKey = {
        name: 'new-key',
        provider: 'openai',
        base_url: 'https://api.openai.com/v1',
        api_key: 'sk-new123456789',
        model: 'gpt-4',
        priority: 10
      }

      server.use(
        http.post('/api/api-keys', () => {
          return HttpResponse.json({
            ...newKey,
            id: 2,
            status: 'enabled',
            request_count: 0,
            error_count: 0,
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString()
          }, { status: 201 })
        })
      )

      const response = await apiKeyApi.create(newKey)

      expect(response.status).toBe(201)
      expect(response.data.name).toBe('new-key')
      expect(response.data.id).toBe(2)
    })

    it('当参数无效时应该返回 400', async () => {
      const invalidKey = {
        name: '' // 空名称
      }

      server.use(
        http.post('/api/api-keys', () => {
          return HttpResponse.json({ error: 'Name is required' }, { status: 400 })
        })
      )

      await expect(apiKeyApi.create(invalidKey)).rejects.toThrow()
    })
  })

  describe('update', () => {
    it('应该成功更新 API Key', async () => {
      const updateData = {
        name: 'updated-key',
        priority: 20
      }

      server.use(
        http.put('/api/api-keys/1', () => {
          return HttpResponse.json({
            ...mockApiKey,
            ...updateData
          }, { status: 200 })
        })
      )

      const response = await apiKeyApi.update(1, updateData)

      expect(response.status).toBe(200)
      expect(response.data.name).toBe('updated-key')
      expect(response.data.priority).toBe(20)
    })

    it('当 API Key 不存在时应该返回 404', async () => {
      server.use(
        http.put('/api/api-keys/999', () => {
          return HttpResponse.json({ error: 'Not found' }, { status: 404 })
        })
      )

      await expect(apiKeyApi.update(999, { name: 'test' })).rejects.toThrow()
    })
  })

  describe('delete', () => {
    it('应该成功删除 API Key', async () => {
      server.use(
        http.delete('/api/api-keys/1', () => {
          return HttpResponse.json({ message: 'deleted successfully' }, { status: 200 })
        })
      )

      const response = await apiKeyApi.delete(1)

      expect(response.status).toBe(200)
    })

    it('当 API Key 不存在时应该返回 404', async () => {
      server.use(
        http.delete('/api/api-keys/999', () => {
          return HttpResponse.json({ error: 'Not found' }, { status: 404 })
        })
      )

      await expect(apiKeyApi.delete(999)).rejects.toThrow()
    })
  })

  describe('updateStatus', () => {
    it('应该成功更新 API Key 状态', async () => {
      server.use(
        http.patch('/api/api-keys/1/status', () => {
          return HttpResponse.json({ message: 'status updated successfully' }, { status: 200 })
        })
      )

      const response = await apiKeyApi.updateStatus(1, 'disabled')

      expect(response.status).toBe(200)
    })

    it('当状态无效时应该返回错误', async () => {
      server.use(
        http.patch('/api/api-keys/1/status', () => {
          return HttpResponse.json({ error: 'Invalid status' }, { status: 400 })
        })
      )

      await expect(apiKeyApi.updateStatus(1, 'invalid')).rejects.toThrow()
    })
  })

  describe('getStats', () => {
    it('应该成功获取统计信息', async () => {
      const mockStats: APIKeyStats = {
        total_count: 10,
        enabled_count: 8,
        disabled_count: 2,
        unavailable_count: 0,
        total_requests: 1000,
        total_errors: 50
      }

      server.use(
        http.get('/api/api-keys/stats', () => {
          return HttpResponse.json(mockStats, { status: 200 })
        })
      )

      const response = await apiKeyApi.getStats()

      expect(response.status).toBe(200)
      expect(response.data.total_count).toBe(10)
      expect(response.data.enabled_count).toBe(8)
      expect(response.data.total_requests).toBe(1000)
    })
  })
})

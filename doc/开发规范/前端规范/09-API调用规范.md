# API 调用规范

本文档定义 API 服务的封装方式和错误处理。

## API 服务文件结构

```tsx
// services/api.ts
import axios from 'axios';
import type { Repository, Task, Document } from '@/types';

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080/api';

const api = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
});

// 按资源分组
export const repositoryApi = {
  list: () => api.get<Repository[]>('/repositories'),
  get: (id: number) => api.get<Repository>(`/repositories/${id}`),
  create: (url: string) => api.post<Repository>('/repositories', { url }),
  delete: (id: number) => api.delete(`/repositories/${id}`),
};

export const taskApi = {
  run: (id: number) => api.post(`/tasks/${id}/run`),
  reset: (id: number) => api.post(`/tasks/${id}/reset`),
  forceReset: (id: number) => api.post(`/tasks/${id}/force-reset`),
};

export const documentApi = {
  list: (repoId: number) => api.get<Document[]>(`/repositories/${repoId}/documents`),
  get: (id: number) => api.get<Document>(`/documents/${id}`),
};
```

## 错误处理

```tsx
import { toast } from '@/components/ui/use-toast';

// 统一错误处理
const handleApiError = (error: unknown) => {
  if (axios.isAxiosError(error)) {
    const message = error.response?.data?.error || error.message;
    toast({ title: 'Error', description: message, variant: 'destructive' });
  }
};

// 使用示例
try {
  await repositoryApi.create(url);
  toast({ title: 'Success', description: 'Repository created' });
} catch (error) {
  handleApiError(error);
}
```

## 请求拦截器

```tsx
// 请求拦截器（如需添加 token）
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// 响应拦截器
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // 处理未授权
    }
    return Promise.reject(error);
  }
);
```

## 在组件中使用

```tsx
function RepositoryList() {
  const [repos, setRepos] = useState<Repository[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchRepos();
  }, []);

  const fetchRepos = async () => {
    setLoading(true);
    try {
      const res = await repositoryApi.list();
      setRepos(res.data);
    } catch (error) {
      handleApiError(error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await repositoryApi.delete(id);
      toast({ title: 'Success', description: 'Repository deleted' });
      fetchRepos(); // 刷新列表
    } catch (error) {
      handleApiError(error);
    }
  };

  // ...
}
```

## API 命名规范

| 操作 | HTTP 方法 | 命名 | 示例 |
| ---- | --------- | ---- | ---- |
| 获取列表 | GET | list | `repositoryApi.list()` |
| 获取单个 | GET | get | `repositoryApi.get(id)` |
| 创建 | POST | create | `repositoryApi.create(data)` |
| 更新 | PUT/PATCH | update | `repositoryApi.update(id, data)` |
| 删除 | DELETE | delete | `repositoryApi.delete(id)` |
| 自定义操作 | POST | 动词 | `taskApi.run(id)` |

## 相关文档

- [类型定义规范](./10-类型定义规范.md)
- [状态管理规范](./06-状态管理规范.md)

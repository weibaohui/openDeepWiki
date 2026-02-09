import axios from 'axios';
import type { Repository, Task, Document, DocumentRatingStats, APIKey, APIKeyStats, GlobalMonitorData } from '../types';

const API_BASE = import.meta.env.VITE_API_BASE || '/api/';

const api = axios.create({
    baseURL: API_BASE,
    headers: {
        'Content-Type': 'application/json',
    },
});

// Repository APIs
export const repositoryApi = {
    list: () => api.get<Repository[]>('/repositories'),
    get: (id: number) => api.get<Repository>(`/repositories/${id}`),
    create: (url: string) => api.post<Repository>('/repositories', { url }),
    delete: (id: number) => api.delete(`/repositories/${id}`),
    clone: (id: number) => api.post(`/repositories/${id}/clone`),
    purgeLocal: (id: number) => api.post(`/repositories/${id}/purge-local`),
    runAll: (id: number) => api.post(`/repositories/${id}/run-all`),
    analyzeDirectory: (id: number) => api.post<{ tasks: Task[]; message: string }>(`/repositories/${id}/directory-analyze`),
    analyzeDatabaseModel: (id: number) => api.post<{ task: Task; message: string }>(`/repositories/${id}/db-model-analyze`),
    // 触发API接口分析
    analyzeAPI: (id: number) => api.post<{ task: Task; message: string }>(`/repositories/${id}/api-analyze`),
    setReady: (id: number) => api.post(`/repositories/${id}/set-ready`),
    createUserRequest: (id: number, content: string) => api.post<{ code: number; message: string; data: Task }>(`/repositories/${id}/user-requests`, { content }),
};

// Task APIs
export const taskApi = {
    getByRepository: (repoId: number) => api.get<Task[]>(`/repositories/${repoId}/tasks`),
    getStats: (repoId: number) => api.get<Record<string, number>>(`/repositories/${repoId}/tasks/stats`),
    get: (id: number) => api.get<Task>(`/tasks/${id}`),
    run: (id: number) => api.post(`/tasks/${id}/run`),
    enqueue: (id: number) => api.post(`/tasks/${id}/enqueue`),
    retry: (id: number) => api.post(`/tasks/${id}/retry`),
    cancel: (id: number) => api.post(`/tasks/${id}/cancel`),
    reset: (id: number) => api.post(`/tasks/${id}/reset`),
    delete: (id: number) => api.delete(`/tasks/${id}`),
    monitor: () => api.get<GlobalMonitorData>('/tasks/monitor'),
};

// Document APIs
export const documentApi = {
    getByRepository: (repoId: number) => api.get<Document[]>(`/repositories/${repoId}/documents`),
    get: (id: number) => api.get<Document>(`/documents/${id}`),
    getVersions: (id: number) => api.get<Document[]>(`/documents/${id}/versions`),
    update: (id: number, content: string) => api.put<Document>(`/documents/${id}`, { content }),
    submitRating: (id: number, score: number) => api.post<DocumentRatingStats>(`/documents/${id}/ratings`, { score }),
    getRatingStats: (id: number) => api.get<DocumentRatingStats>(`/documents/${id}/ratings/stats`),
    getIndex: (repoId: number) => api.get<{ content: string }>(`/repositories/${repoId}/documents/index`),
    export: (repoId: number) => api.get(`/repositories/${repoId}/documents/export`, { responseType: 'blob' }),
};


// API Key APIs
export const apiKeyApi = {
    list: () => api.get<{ data: APIKey[]; total: number }>('/api-keys'),
    get: (id: number) => api.get<APIKey>(`/api-keys/${id}`),
    create: (data: Partial<APIKey>) => api.post<APIKey>('/api-keys', data),
    update: (id: number, data: Partial<APIKey>) => api.put<APIKey>(`/api-keys/${id}`, data),
    delete: (id: number) => api.delete(`/api-keys/${id}`),
    updateStatus: (id: number, status: string) => api.patch(`/api-keys/${id}/status`, { status }),
    getStats: () => api.get<APIKeyStats>('/api-keys/stats'),
};

export default api;

import axios from 'axios';
import type { Repository, Task, Document, Config, DocumentTemplate, TemplateDetail, TemplateChapter, TemplateDocument, AIAnalysisStatus, APIKey, APIKeyStats } from '../types';

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
    runAll: (id: number) => api.post(`/repositories/${id}/run-all`),
    analyzeDirectory: (id: number) => api.post<{ tasks: Task[]; message: string }>(`/repositories/${id}/directory-analyze`),
    setReady: (id: number) => api.post(`/repositories/${id}/set-ready`),
};

// Task APIs
export const taskApi = {
    getByRepository: (repoId: number) => api.get<Task[]>(`/repositories/${repoId}/tasks`),
    get: (id: number) => api.get<Task>(`/tasks/${id}`),
    run: (id: number) => api.post(`/tasks/${id}/run`),
    enqueue: (id: number) => api.post(`/tasks/${id}/enqueue`),
    retry: (id: number) => api.post(`/tasks/${id}/retry`),
    cancel: (id: number) => api.post(`/tasks/${id}/cancel`),
    reset: (id: number) => api.post(`/tasks/${id}/reset`),
    delete: (id: number) => api.delete(`/tasks/${id}`),
};

// Document APIs
export const documentApi = {
    getByRepository: (repoId: number) => api.get<Document[]>(`/repositories/${repoId}/documents`),
    get: (id: number) => api.get<Document>(`/documents/${id}`),
    getVersions: (id: number) => api.get<Document[]>(`/documents/${id}/versions`),
    update: (id: number, content: string) => api.put<Document>(`/documents/${id}`, { content }),
    getIndex: (repoId: number) => api.get<{ content: string }>(`/repositories/${repoId}/documents/index`),
    export: (repoId: number) => api.get(`/repositories/${repoId}/documents/export`, { responseType: 'blob' }),
};

// Config APIs
export const configApi = {
    get: () => api.get<Config>('/config'),
    update: (config: Partial<Config>) => api.put('/config', config),
};

// AI Analysis APIs
export const aiAnalyzeApi = {
    start: (repoId: number) => api.post<{ task_id: string; status: string; message: string }>(`/repositories/${repoId}/ai-analyze`),
    getStatus: (repoId: number) => api.get<AIAnalysisStatus>(`/repositories/${repoId}/ai-analysis-status`),
    getResult: (repoId: number) => api.get<{ content: string }>(`/repositories/${repoId}/ai-analysis-result`),
};

// Document Template APIs
export const templateApi = {
    list: () => api.get<{ data: DocumentTemplate[] }>('/document-templates'),
    get: (id: number) => api.get<{ data: TemplateDetail }>(`/document-templates/${id}`),
    create: (data: { key: string; name: string; description?: string; sort_order?: number }) =>
        api.post<{ data: DocumentTemplate }>('/document-templates', data),
    update: (id: number, data: { name: string; description?: string; sort_order?: number }) =>
        api.put<{ data: DocumentTemplate }>(`/document-templates/${id}`, data),
    delete: (id: number) => api.delete(`/document-templates/${id}`),
    clone: (id: number, key: string) =>
        api.post<{ data: DocumentTemplate }>(`/document-templates/${id}/clone`, { key }),
    createChapter: (templateId: number, data: { title: string; sort_order?: number }) =>
        api.post<{ data: TemplateChapter }>(`/document-templates/${templateId}/chapters`, data),
    updateChapter: (id: number, data: { title: string; sort_order?: number }) =>
        api.put<{ data: TemplateChapter }>(`/chapters/${id}`, data),
    deleteChapter: (id: number) => api.delete(`/chapters/${id}`),
    createDocument: (chapterId: number, data: { title: string; filename: string; content_prompt?: string; sort_order?: number }) =>
        api.post<{ data: TemplateDocument }>(`/chapters/${chapterId}/documents`, data),
    updateDocument: (id: number, data: { title: string; filename: string; content_prompt?: string; sort_order?: number }) =>
        api.put<{ data: TemplateDocument }>(`/template-documents/${id}`, data),
    deleteDocument: (id: number) => api.delete(`/template-documents/${id}`),
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

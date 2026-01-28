import axios from 'axios';
import type { Repository, Task, Document, Config } from '../types';

const API_BASE = import.meta.env.VITE_API_BASE || '/';

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
};

// Task APIs
export const taskApi = {
    getByRepository: (repoId: number) => api.get<Task[]>(`/repositories/${repoId}/tasks`),
    get: (id: number) => api.get<Task>(`/tasks/${id}`),
    run: (id: number) => api.post(`/tasks/${id}/run`),
    reset: (id: number) => api.post(`/tasks/${id}/reset`),
};

// Document APIs
export const documentApi = {
    getByRepository: (repoId: number) => api.get<Document[]>(`/repositories/${repoId}/documents`),
    get: (id: number) => api.get<Document>(`/documents/${id}`),
    update: (id: number, content: string) => api.put<Document>(`/documents/${id}`, { content }),
    getIndex: (repoId: number) => api.get<{ content: string }>(`/repositories/${repoId}/documents/index`),
    export: (repoId: number) => api.get(`/repositories/${repoId}/documents/export`, { responseType: 'blob' }),
};

// Config APIs
export const configApi = {
    get: () => api.get<Config>('/config'),
    update: (config: Partial<Config>) => api.put('/config', config),
};

export default api;

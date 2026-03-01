import axios from 'axios';
import type { Repository, Task, Document, DocumentRatingStats, APIKey, APIKeyStats, EmbeddingKey, EmbeddingKeyStats, GlobalMonitorData, SyncStartResponse, SyncStatusResponse, TaskUsage, SyncRepositoryListResponse, SyncDocumentListResponse, SyncTargetListResponse, SyncTargetSaveResponse, SyncTargetDeleteResponse, SyncEventListResponse, IncrementalUpdateHistory, UserRequest, UserRequestListResponse } from '../types';

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
    incrementalAnalysis: (id: number) => api.post(`/repositories/${id}/incremental-analysis`),
    getIncrementalHistory: (id: number, limit?: number) => api.get<IncrementalUpdateHistory[]>(`/repositories/${id}/incremental-history`, {
        params: { limit }
    }),
    setReady: (id: number) => api.post(`/repositories/${id}/set-ready`),
};

// User Request APIs
export const userRequestApi = {
    create: (repoId: number, content: string) => api.post<UserRequestListResponse>(`/repositories/${repoId}/user-requests`, { content }),
    list: (repoId: number, params?: { page?: number; page_size?: number; status?: string }) =>
        api.get<UserRequestListResponse>(`/repositories/${repoId}/user-requests`, { params }),
    get: (id: number) => api.get<{ code: number; message: string; data: UserRequest }>(`/user-requests/${id}`),
    delete: (id: number) => api.delete<{ code: number; message: string }>(`/user-requests/${id}`),
};

// Task APIs
export const taskApi = {
    getByRepository: (repoId: number) => api.get<Task[]>(`/repositories/${repoId}/tasks`),
    getStats: (repoId: number) => api.get<Record<string, number>>(`/repositories/${repoId}/tasks/stats`),
    get: (id: number) => api.get<Task>(`/tasks/${id}`),
    run: (id: number) => api.post(`/tasks/${id}/run`),
    enqueue: (id: number) => api.post(`/tasks/${id}/enqueue`),
    retry: (id: number) => api.post(`/tasks/${id}/retry`),
    regen: (id: number) => api.post(`/tasks/${id}/regen`),
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
    getTokenUsage: (id: number) => api.get<{ code: number; message: string; data: TaskUsage | null }>(`/documents/${id}/token-usage`).then(res => res.data),
    getIndex: (repoId: number) => api.get<{ content: string }>(`/repositories/${repoId}/documents/index`),
    export: (repoId: number) => api.get(`/repositories/${repoId}/documents/export`, { responseType: 'blob' }),
    exportPdf: (repoId: number) => api.get(`/repositories/${repoId}/export-pdf`, { responseType: 'blob' }),
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

// Embedding Key APIs
export const embeddingKeyApi = {
    list: () => api.get<EmbeddingKey[]>('/api/embedding-keys'),
    get: (id: number) => api.get<EmbeddingKey>(`/api/embedding-keys/${id}`),
    create: (data: Partial<EmbeddingKey>) => api.post<EmbeddingKey>('/api/embedding-keys', data),
    update: (id: number, data: Partial<EmbeddingKey>) => api.put<EmbeddingKey>(`/api/embedding-keys/${id}`, data),
    delete: (id: number) => api.delete(`/api/embedding-keys/${id}`),
    enable: (id: number) => api.post(`/api/embedding-keys/${id}/enable`),
    disable: (id: number) => api.post(`/api/embedding-keys/${id}/disable`),
    testConnection: (id: number) => api.post<{ success: boolean; error?: string }>(`/api/embedding-keys/${id}/test`),
    getStats: () => api.get<EmbeddingKeyStats>('/api/embedding-keys/stats'),
};

export const syncApi = {
    start: (targetServer: string, repositoryId: number, documentIds?: number[], clearTarget?: boolean) => api.post<SyncStartResponse>('/sync', {
        target_server: targetServer,
        repository_id: repositoryId,
        ...(documentIds && documentIds.length > 0 ? { document_ids: documentIds } : {}),
        ...(clearTarget ? { clear_target: true } : {}),
    }),
    pull: (targetServer: string, repositoryId: number, documentIds?: number[], clearLocal?: boolean) => api.post<SyncStartResponse>('/sync/pull', {
        target_server: targetServer,
        repository_id: repositoryId,
        ...(documentIds && documentIds.length > 0 ? { document_ids: documentIds } : {}),
        ...(clearLocal ? { clear_local: true } : {}),
    }),
    status: (syncId: string) => api.get<SyncStatusResponse>(`/sync/status/${syncId}`),
    remoteRepositoryList: (targetServer: string) => api.get<SyncRepositoryListResponse>(`${targetServer}/repository-list`),
    remoteDocumentList: (targetServer: string, repositoryId: number) => api.get<SyncDocumentListResponse>(`${targetServer}/document-list`, {
        params: { repository_id: repositoryId },
    }),
    targetList: () => api.get<SyncTargetListResponse>('/sync/target-list'),
    targetSave: (url: string) => api.post<SyncTargetSaveResponse>('/sync/target-save', { url }),
    targetDelete: (id: number) => api.post<SyncTargetDeleteResponse>('/sync/target-delete', { id }),
    eventList: (params: { repository_id?: number; mode?: string; limit?: number }) => api.get<SyncEventListResponse>('/sync/event-list', { params }),
};

export default api;

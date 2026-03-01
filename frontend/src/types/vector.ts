// 向量状态统计
export interface VectorStatus {
    total_documents: number;
    vectorized_count: number;
    pending_count: number;
    failed_count: number;
    processing_count: number;
}

// 仓库向量化状态
export interface RepositoryVectorStatus {
    repository_id: number;
    repository_name: string;
    total_documents: number;
    vectorized_count: number;
    status: 'not_started' | 'partial' | 'completed';
}

// 向量任务
export interface VectorTask {
    id: number;
    document_id: number;
    document_title?: string;
    repository_id: number;
    repository_name?: string;
    status: 'pending' | 'processing' | 'completed' | 'failed';
    error_message: string;
    created_at: string;
    started_at: string | null;
    completed_at: string | null;
}

// 向量任务列表响应
export interface VectorTaskListResponse {
    list: VectorTask[];
    total: number;
    page: number;
    page_size: number;
}

// 仓库向量化状态列表响应
export interface RepositoryVectorStatusListResponse {
    list: RepositoryVectorStatus[];
    total: number;
}

export interface Repository {
    id: number;
    name: string;
    url: string;
    local_path: string;
    description: string;
    clone_branch?: string;
    clone_commit_id?: string;
    size_mb?: number;
    status: 'pending' | 'cloning' | 'ready' | 'analyzing' | 'completed' | 'error';
    error_msg: string;
    created_at: string;
    updated_at: string;
    tasks?: Task[];
    documents?: Document[];
}

export interface Task {
    id: number;
    repository_id: number;
    writer_name?: string;
    task_type?: string;
    type: string;
    title: string;
    status: 'pending' | 'running' | 'completed' | 'succeeded' | 'failed' | 'queued' | 'canceled';
    doc_id: string,
    error_msg: string;
    sort_order: number;
    started_at: string | null;
    completed_at: string | null;
    created_at: string;
    updated_at: string;
    repository?: Repository;
}

export interface Document {
    id: number;
    repository_id: number;
    task_id: number;
    title: string;
    filename: string;
    content: string;
    sort_order: number;
    version: number;
    is_latest: boolean;
    created_at: string;
    updated_at: string;
}

export interface DocumentRatingStats {
    average_score: number;
    rating_count: number;
}

export interface TaskUsage {
    id: number;
    task_id: number;
    api_key_name: string;
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
    cached_tokens: number;
    reasoning_tokens: number;
    created_at: string;
}


export type ThemeMode = 'default' | 'dark' | 'compact';

export interface APIKey {
    id: number;
    name: string;
    provider: string;
    base_url: string;
    api_key: string;
    model: string;
    priority: number;
    status: 'enabled' | 'disabled' | 'unavailable';
    request_count: number;
    error_count: number;
    last_used_at: string | null;
    rate_limit_reset_at: string | null;
    created_at: string;
    updated_at: string;
}

export interface APIKeyStats {
    total_count: number;
    enabled_count: number;
    disabled_count: number;
    unavailable_count: number;
    total_requests: number;
    total_errors: number;
}

export interface QueueStatus {
    queue_length: number;
    priority_length: number;
    active_workers: number;
    active_repos: number;
}

export interface GlobalMonitorData {
    queue_status: QueueStatus;
    active_tasks: Task[];
    recent_tasks: Task[];
}

export interface SyncStartData {
    sync_id: string;
    repository_id: number;
    total_tasks: number;
    status: string;
}

export interface SyncStatusData {
    sync_id: string;
    repository_id: number;
    total_tasks: number;
    completed_tasks: number;
    failed_tasks: number;
    status: string;
    current_task: string;
    started_at: string;
    updated_at: string;
}

export interface SyncStartResponse {
    code: string;
    data: SyncStartData;
}

export interface SyncStatusResponse {
    code: string;
    data: SyncStatusData;
}

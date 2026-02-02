export interface Repository {
    id: number;
    name: string;
    url: string;
    local_path: string;
    description: string;
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
    type: string;
    title: string;
    status: 'pending' | 'running' | 'completed' | 'failed' | 'queued' | 'canceled';
    error_msg: string;
    sort_order: number;
    started_at: string | null;
    completed_at: string | null;
    created_at: string;
    updated_at: string;
}

export interface Document {
    id: number;
    repository_id: number;
    task_id: number;
    title: string;
    filename: string;
    content: string;
    sort_order: number;
    created_at: string;
    updated_at: string;
}

export interface Config {
    llm: {
        api_url: string;
        api_key: string;
        model: string;
        max_tokens: number;
    };
    github: {
        token: string;
    };
}

export type ThemeMode = 'default' | 'dark' | 'compact';

// 文档模板类型
export interface DocumentTemplate {
    id: number;
    key: string;
    name: string;
    description: string;
    is_system: boolean;
    sort_order: number;
    created_at: string;
    updated_at: string;
}

export interface TemplateChapter {
    id: number;
    title: string;
    sort_order: number;
    documents: TemplateDocument[];
}

export interface TemplateDocument {
    id: number;
    title: string;
    filename: string;
    content_prompt: string;
    sort_order: number;
}

export interface TemplateDetail extends DocumentTemplate {
    chapters: TemplateChapter[];
}

// AI分析任务类型
export interface AIAnalysisStatus {
    id: number;
    repository_id: number;
    task_id: string;
    status: 'pending' | 'running' | 'completed' | 'failed';
    progress: number;
    output_path: string;
    error_msg: string;
    created_at: string;
    updated_at: string;
    completed_at: string | null;
}

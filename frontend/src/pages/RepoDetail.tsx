import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, Play, RotateCcw, FileText, CheckCircle, Clock, XCircle, Loader2 } from 'lucide-react';
import type { Repository, Task, Document } from '../types';
import { repositoryApi, taskApi, documentApi } from '../services/api';

export default function RepoDetail() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [repository, setRepository] = useState<Repository | null>(null);
    const [tasks, setTasks] = useState<Task[]>([]);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [loading, setLoading] = useState(true);

    const fetchData = async () => {
        if (!id) return;
        try {
            const [repoRes, tasksRes, docsRes] = await Promise.all([
                repositoryApi.get(Number(id)),
                taskApi.getByRepository(Number(id)),
                documentApi.getByRepository(Number(id)),
            ]);
            setRepository(repoRes.data);
            setTasks(tasksRes.data);
            setDocuments(docsRes.data);
        } catch (error) {
            console.error('Failed to fetch data:', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchData();
        const interval = setInterval(fetchData, 3000);
        return () => clearInterval(interval);
    }, [id]);

    const handleRunTask = async (taskId: number) => {
        try {
            await taskApi.run(taskId);
            fetchData();
        } catch (error) {
            console.error('Failed to run task:', error);
        }
    };

    const handleResetTask = async (taskId: number) => {
        try {
            await taskApi.reset(taskId);
            fetchData();
        } catch (error) {
            console.error('Failed to reset task:', error);
        }
    };

    const handleRunAll = async () => {
        if (!id) return;
        try {
            await repositoryApi.runAll(Number(id));
            fetchData();
        } catch (error) {
            console.error('Failed to run all tasks:', error);
        }
    };

    const handleExport = async () => {
        if (!id) return;
        try {
            const response = await documentApi.export(Number(id));
            const blob = new Blob([response.data], { type: 'application/zip' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `${repository?.name || 'docs'}-docs.zip`;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (error) {
            console.error('Failed to export:', error);
        }
    };

    const getTaskIcon = (status: string) => {
        switch (status) {
            case 'completed':
                return <CheckCircle className="w-5 h-5 text-green-500" />;
            case 'running':
                return <Loader2 className="w-5 h-5 text-blue-500 animate-spin" />;
            case 'failed':
                return <XCircle className="w-5 h-5 text-red-500" />;
            default:
                return <Clock className="w-5 h-5 text-gray-400" />;
        }
    };

    const getTaskStatusText = (status: string) => {
        const texts: Record<string, string> = {
            pending: '等待中',
            running: '运行中',
            completed: '已完成',
            failed: '失败',
        };
        return texts[status] || status;
    };

    const getDocumentForTask = (taskId: number) => {
        return documents.find((doc) => doc.task_id === taskId);
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
        );
    }

    if (!repository) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <p className="text-gray-500">仓库不存在</p>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50">
            <header className="bg-white shadow-sm">
                <div className="max-w-7xl mx-auto px-4 py-4 sm:px-6 lg:px-8">
                    <div className="flex items-center gap-4">
                        <button
                            onClick={() => navigate('/')}
                            className="p-2 hover:bg-gray-100 rounded-lg"
                        >
                            <ArrowLeft className="w-5 h-5" />
                        </button>
                        <div className="flex-1">
                            <h1 className="text-xl font-bold text-gray-900">{repository.name}</h1>
                            <p className="text-sm text-gray-500 truncate">{repository.url}</p>
                        </div>
                        <div className="flex gap-2">
                            {documents.length > 0 && (
                                <button
                                    onClick={handleExport}
                                    className="px-4 py-2 text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-lg"
                                >
                                    导出文档
                                </button>
                            )}
                            {(repository.status === 'ready' || repository.status === 'completed') && (
                                <button
                                    onClick={handleRunAll}
                                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                                >
                                    <Play className="w-4 h-4" />
                                    运行全部
                                </button>
                            )}
                        </div>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8">
                <div className="grid gap-8 lg:grid-cols-2">
                    <div>
                        <h2 className="text-lg font-semibold mb-4">任务列表</h2>
                        <div className="bg-white rounded-lg shadow-sm border border-gray-200">
                            {tasks.map((task, index) => (
                                <div
                                    key={task.id}
                                    className={`p-4 flex items-center gap-4 ${index !== tasks.length - 1 ? 'border-b border-gray-100' : ''
                                        }`}
                                >
                                    {getTaskIcon(task.status)}
                                    <div className="flex-1">
                                        <p className="font-medium text-gray-900">{task.title}</p>
                                        <p className="text-sm text-gray-500">{getTaskStatusText(task.status)}</p>
                                        {task.error_msg && (
                                            <p className="text-sm text-red-500 mt-1">{task.error_msg}</p>
                                        )}
                                    </div>
                                    <div className="flex gap-2">
                                        {task.status !== 'running' && (
                                            <>
                                                <button
                                                    onClick={() => handleRunTask(task.id)}
                                                    className="p-2 text-blue-600 hover:bg-blue-50 rounded"
                                                    title="运行"
                                                >
                                                    <Play className="w-4 h-4" />
                                                </button>
                                                {(task.status === 'completed' || task.status === 'failed') && (
                                                    <button
                                                        onClick={() => handleResetTask(task.id)}
                                                        className="p-2 text-gray-600 hover:bg-gray-50 rounded"
                                                        title="重置"
                                                    >
                                                        <RotateCcw className="w-4 h-4" />
                                                    </button>
                                                )}
                                            </>
                                        )}
                                        {getDocumentForTask(task.id) && (
                                            <button
                                                onClick={() => navigate(`/repo/${id}/doc/${getDocumentForTask(task.id)?.id}`)}
                                                className="p-2 text-green-600 hover:bg-green-50 rounded"
                                                title="查看文档"
                                            >
                                                <FileText className="w-4 h-4" />
                                            </button>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div>
                        <h2 className="text-lg font-semibold mb-4">生成的文档</h2>
                        {documents.length === 0 ? (
                            <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center">
                                <FileText className="w-12 h-12 mx-auto text-gray-300 mb-2" />
                                <p className="text-gray-500">还没有生成文档</p>
                                <p className="text-sm text-gray-400">运行任务后将在此显示</p>
                            </div>
                        ) : (
                            <div className="bg-white rounded-lg shadow-sm border border-gray-200">
                                {documents.map((doc, index) => (
                                    <div
                                        key={doc.id}
                                        onClick={() => navigate(`/repo/${id}/doc/${doc.id}`)}
                                        className={`p-4 flex items-center gap-4 cursor-pointer hover:bg-gray-50 ${index !== documents.length - 1 ? 'border-b border-gray-100' : ''
                                            }`}
                                    >
                                        <FileText className="w-5 h-5 text-blue-500" />
                                        <div className="flex-1">
                                            <p className="font-medium text-gray-900">{doc.title}</p>
                                            <p className="text-sm text-gray-500">{doc.filename}</p>
                                        </div>
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            </main>
        </div>
    );
}

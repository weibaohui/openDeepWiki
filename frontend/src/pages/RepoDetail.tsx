import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, Play, RotateCcw, FileText, CheckCircle, Clock, XCircle, Loader2 } from 'lucide-react';
import type { Repository, Task, Document } from '../types';
import { repositoryApi, taskApi, documentApi } from '../services/api';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';

export default function RepoDetail() {
    const { t } = useTranslation();
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
        // eslint-disable-next-line react-hooks/exhaustive-deps
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
                return <CheckCircle className="w-5 h-5 text-green-500 dark:text-green-400" />;
            case 'running':
                return <Loader2 className="w-5 h-5 text-blue-500 dark:text-blue-400 animate-spin" />;
            case 'failed':
                return <XCircle className="w-5 h-5 text-red-500 dark:text-red-400" />;
            default:
                return <Clock className="w-5 h-5 text-muted-foreground" />;
        }
    };

    const getDocumentForTask = (taskId: number) => {
        return documents.find((doc) => doc.task_id === taskId);
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (!repository) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <p className="text-muted-foreground">{t('repository.not_found')}</p>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-background text-foreground">
            <header className="border-b bg-card">
                <div className="max-w-7xl mx-auto px-4 py-4 sm:px-6 lg:px-8">
                    <div className="flex items-center gap-4">
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => navigate('/')}
                        >
                            <ArrowLeft className="w-5 h-5" />
                        </Button>
                        <div className="flex-1">
                            <h1 className="text-xl font-bold">{repository.name}</h1>
                            <p className="text-sm text-muted-foreground truncate">{repository.url}</p>
                        </div>
                        <div className="flex gap-2 items-center">
                            <LanguageSwitcher />
                            <ThemeSwitcher />
                            {documents.length > 0 && (
                                <Button
                                    variant="outline"
                                    onClick={handleExport}
                                >
                                    {t('repository.export_docs')}
                                </Button>
                            )}
                            {(repository.status === 'ready' || repository.status === 'completed') && (
                                <Button
                                    onClick={handleRunAll}
                                    className="gap-2"
                                >
                                    <Play className="w-4 h-4" />
                                    {t('repository.run_all')}
                                </Button>
                            )}
                        </div>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8">
                <div className="grid gap-8 lg:grid-cols-2">
                    <div>
                        <h2 className="text-lg font-semibold mb-4">{t('task.title')}</h2>
                        <Card>
                            {tasks.map((task, index) => (
                                <div
                                    key={task.id}
                                    className={`p-4 flex items-center gap-4 ${index !== tasks.length - 1 ? 'border-b border-border' : ''
                                        }`}
                                >
                                    {getTaskIcon(task.status)}
                                    <div className="flex-1">
                                        <p className="font-medium">{task.title}</p>
                                        <p className="text-sm text-muted-foreground">{t(`task.status.${task.status}`)}</p>
                                        {task.error_msg && (
                                            <p className="text-sm text-destructive mt-1">{task.error_msg}</p>
                                        )}
                                    </div>
                                    <div className="flex gap-2">
                                        {task.status !== 'running' && (
                                            <>
                                                <Button
                                                    variant="ghost"
                                                    size="icon"
                                                    onClick={() => handleRunTask(task.id)}
                                                    title={t('task.run')}
                                                    className="text-blue-600 hover:text-blue-700 hover:bg-blue-50 dark:hover:bg-blue-950"
                                                >
                                                    <Play className="w-4 h-4" />
                                                </Button>
                                                {(task.status === 'completed' || task.status === 'failed') && (
                                                    <Button
                                                        variant="ghost"
                                                        size="icon"
                                                        onClick={() => handleResetTask(task.id)}
                                                        title={t('task.reset')}
                                                    >
                                                        <RotateCcw className="w-4 h-4" />
                                                    </Button>
                                                )}
                                            </>
                                        )}
                                        {getDocumentForTask(task.id) && (
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                onClick={() => navigate(`/repo/${id}/doc/${getDocumentForTask(task.id)?.id}`)}
                                                title={t('repository.view_docs')}
                                                className="text-green-600 hover:text-green-700 hover:bg-green-50 dark:hover:bg-green-950"
                                            >
                                                <FileText className="w-4 h-4" />
                                            </Button>
                                        )}
                                    </div>
                                </div>
                            ))}
                        </Card>
                    </div>

                    <div>
                        <h2 className="text-lg font-semibold mb-4">{t('repository.docs')}</h2>
                        {documents.length === 0 ? (
                            <Card className="p-8 text-center border-dashed">
                                <FileText className="w-12 h-12 mx-auto text-muted-foreground mb-2 opacity-50" />
                                <p className="text-muted-foreground">{t('repository.no_docs')}</p>
                                <p className="text-sm text-muted-foreground/60">{t('repository.no_docs_hint')}</p>
                            </Card>
                        ) : (
                            <Card>
                                {documents.map((doc, index) => (
                                    <div
                                        key={doc.id}
                                        onClick={() => navigate(`/repo/${id}/doc/${doc.id}`)}
                                        className={`p-4 flex items-center gap-4 cursor-pointer hover:bg-accent/50 ${index !== documents.length - 1 ? 'border-b border-border' : ''
                                            }`}
                                    >
                                        <FileText className="w-5 h-5 text-blue-500 dark:text-blue-400" />
                                        <div className="flex-1">
                                            <p className="font-medium">{doc.title}</p>
                                            <p className="text-sm text-muted-foreground">{doc.filename}</p>
                                        </div>
                                    </div>
                                ))}
                            </Card>
                        )}
                    </div>
                </div>
            </main>
        </div>
    );
}

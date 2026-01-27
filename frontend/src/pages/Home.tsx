import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
    Plus,
    Trash2,
    RefreshCw,
    Github,
    Settings,
    Book,
    GitFork,
    Clock,
    ExternalLink,
    ChevronRight,
    Search,
    CheckCircle,
    Loader2,
    AlertTriangle
} from 'lucide-react';
import type { Repository } from '../types';
import { repositoryApi } from '../services/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardHeader, CardTitle, CardContent, CardFooter, CardDescription } from '@/components/ui/card';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from '@/components/ui/dialog';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';
import { Badge } from '@/components/ui/badge';

export default function Home() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const [newRepoUrl, setNewRepoUrl] = useState('');
    const [adding, setAdding] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');

    const fetchRepositories = async () => {
        try {
            const { data } = await repositoryApi.list();
            setRepositories(data);
        } catch (error) {
            console.error('Failed to fetch repositories:', error);
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchRepositories();
        const interval = setInterval(fetchRepositories, 5000);
        return () => clearInterval(interval);
    }, []);

    const handleAddRepository = async () => {
        if (!newRepoUrl.trim()) return;
        setAdding(true);
        try {
            await repositoryApi.create(newRepoUrl.trim());
            setNewRepoUrl('');
            setShowAddModal(false);
            fetchRepositories();
        } catch (error) {
            console.error('Failed to add repository:', error);
            // 这里应该用 Toast，但先保留 console
        } finally {
            setAdding(false);
        }
    };

    const handleDelete = async (id: number, e: React.MouseEvent) => {
        e.stopPropagation();
        if (!confirm(t('repository.delete_confirm'))) return;
        try {
            await repositoryApi.delete(id);
            fetchRepositories();
        } catch (error) {
            console.error('Failed to delete repository:', error);
        }
    };

    const handleRunAll = async (id: number, e: React.MouseEvent) => {
        e.stopPropagation();
        try {
            await repositoryApi.runAll(id);
            fetchRepositories();
        } catch (error) {
            console.error('Failed to run tasks:', error);
        }
    };

    // 使用 shadcn Badge 变体映射
    const getStatusConfig = (status: string) => {
        const configs: Record<string, { variant: "default" | "secondary" | "destructive" | "outline", className?: string }> = {
            pending: { variant: 'secondary', className: 'bg-zinc-100 text-zinc-800 dark:bg-zinc-800 dark:text-zinc-300' },
            cloning: { variant: 'default', className: 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300 hover:bg-blue-200 dark:hover:bg-blue-900/50' },
            ready: { variant: 'outline', className: 'border-green-500 text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-950/20' },
            analyzing: { variant: 'default', className: 'bg-purple-100 text-purple-800 dark:bg-purple-900/30 dark:text-purple-300 animate-pulse' },
            completed: { variant: 'default', className: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300 hover:bg-green-200 dark:hover:bg-green-900/50' },
            error: { variant: 'destructive' },
        };
        return configs[status] || { variant: 'secondary' };
    };

    const getStatusDisplay = (status: string) => {
        const map: Record<string, { label: string, icon: React.ReactNode }> = {
            pending: { label: t('repository.status.pending'), icon: <Clock className="w-3 h-3 mr-1" /> },
            cloning: { label: t('repository.status.cloning'), icon: <GitFork className="w-3 h-3 mr-1" /> },
            analyzing: { label: t('repository.status.analyzing'), icon: <Loader2 className="w-3 h-3 mr-1 animate-spin" /> },
            ready: { label: t('repository.status.ready'), icon: <CheckCircle className="w-3 h-3 mr-1" /> },
            completed: { label: t('repository.status.completed'), icon: <CheckCircle className="w-3 h-3 mr-1" /> },
            error: { label: t('repository.status.error'), icon: <AlertTriangle className="w-3 h-3 mr-1" /> },
        };
        return map[status] || { label: status, icon: null };
    };

    const filteredRepositories = repositories.filter(repo =>
        repo.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        repo.url.toLowerCase().includes(searchQuery.toLowerCase())
    );

    return (
        <div className="min-h-screen bg-background text-foreground flex flex-col">
            <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
                <div className="max-w-7xl mx-auto px-4 h-16 sm:px-6 lg:px-8 flex justify-between items-center">
                    <div className="flex items-center gap-2 transition-transform hover:scale-105 cursor-pointer" onClick={() => navigate('/')}>
                        <div className="bg-primary/10 p-2 rounded-lg">
                            <Book className="w-5 h-5 text-primary" />
                        </div>
                        <h1 className="text-xl font-bold bg-gradient-to-r from-primary to-blue-600 bg-clip-text text-transparent">
                            openDeepWiki
                        </h1>
                    </div>
                    <div className="flex gap-2 items-center">
                        <LanguageSwitcher />
                        <ThemeSwitcher />
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => navigate('/config')}
                            title={t('nav.settings')}
                            className="rounded-full"
                        >
                            <Settings className="w-5 h-5" />
                        </Button>
                    </div>
                </div>
            </header>

            <main className="flex-1 max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8 w-full">
                <div className="mb-8 flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <div>
                        <h2 className="text-3xl font-bold tracking-tight">{t('repository.list_title', 'Repositories')}</h2>
                        <p className="text-muted-foreground mt-1">
                            {t('repository.list_subtitle', 'Manage and generate documentation for your codebases.')}
                        </p>
                    </div>
                    <div className="flex gap-2 items-center">
                        <Dialog open={showAddModal} onOpenChange={setShowAddModal}>
                            <DialogTrigger asChild>
                                <Button className="gap-2 shadow-lg hover:shadow-xl transition-all">
                                    <Plus className="w-4 h-4" />
                                    {t('repository.add')}
                                </Button>
                            </DialogTrigger>
                            <DialogContent className="sm:max-w-md">
                                <DialogHeader>
                                    <DialogTitle>{t('repository.add')}</DialogTitle>
                                    <DialogDescription>
                                        {t('repository.add_hint')}
                                    </DialogDescription>
                                </DialogHeader>
                                <div className="grid gap-4 py-4">
                                    <div className="flex items-center gap-2 bg-muted/50 p-2 rounded border">
                                        <Github className="w-5 h-5 text-muted-foreground ml-2" />
                                        <Input
                                            value={newRepoUrl}
                                            onChange={(e) => setNewRepoUrl(e.target.value)}
                                            placeholder="https://github.com/username/repo"
                                            className="border-0 bg-transparent focus-visible:ring-0 focus-visible:ring-offset-0"
                                            onKeyDown={(e) => e.key === 'Enter' && handleAddRepository()}
                                        />
                                    </div>
                                </div>
                                <DialogFooter>
                                    <Button variant="ghost" onClick={() => setShowAddModal(false)}>
                                        {t('common.cancel')}
                                    </Button>
                                    <Button onClick={handleAddRepository} disabled={adding || !newRepoUrl.trim()}>
                                        {adding ? (
                                            <>
                                                <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                                                {t('common.loading')}
                                            </>
                                        ) : (
                                            t('common.confirm')
                                        )}
                                    </Button>
                                </DialogFooter>
                            </DialogContent>
                        </Dialog>
                    </div>
                </div>

                {repositories.length > 0 && (
                    <div className="mb-6 relative max-w-md">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                        <Input
                            className="pl-9 bg-card/50 backdrop-blur-sm"
                            placeholder={t('common.search', 'Search repositories...')}
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                        />
                    </div>
                )}

                {loading ? (
                    <div className="flex flex-col justify-center items-center h-64 gap-4">
                        <div className="relative">
                            <div className="absolute inset-0 bg-primary/20 rounded-full blur-xl animate-pulse"></div>
                            <RefreshCw className="w-10 h-10 animate-spin text-primary relative z-10" />
                        </div>
                        <p className="text-muted-foreground text-sm">{t('common.loading_data', 'Loading repositories...')}</p>
                    </div>
                ) : filteredRepositories.length === 0 ? (
                    <div className="text-center py-20 bg-card/50 rounded-xl border border-dashed border-muted-foreground/25">
                        <div className="bg-muted/30 w-20 h-20 rounded-full flex items-center justify-center mx-auto mb-6">
                            <Github className="w-10 h-10 text-muted-foreground/60" />
                        </div>
                        <h2 className="text-xl font-semibold mb-2">{searchQuery ? t('common.no_results', 'No matching repositories found') : t('repository.no_repos')}</h2>
                        <p className="text-muted-foreground mb-6 max-w-sm mx-auto">
                            {searchQuery ? t('common.try_different_search', 'Try adjusting your search terms.') : t('repository.add_hint')}
                        </p>
                        {!searchQuery && (
                            <Button onClick={() => setShowAddModal(true)} variant="outline" className="gap-2">
                                <Plus className="w-4 h-4" />
                                {t('repository.add')}
                            </Button>
                        )}
                    </div>
                ) : (
                    <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                        {filteredRepositories.map((repo) => {
                            const statusConfig = getStatusConfig(repo.status);
                            return (
                                <Card
                                    key={repo.id}
                                    onClick={() => navigate(`/repo/${repo.id}`)}
                                    className="group cursor-pointer hover:shadow-lg hover:border-primary/20 transition-all duration-300 flex flex-col overflow-hidden"
                                >
                                    <CardHeader className="pb-3">
                                        <div className="flex justify-between items-start gap-4">
                                            <div className="flex items-center gap-3 overflow-hidden">
                                                <div className="w-10 h-10 rounded-lg bg-primary/5 flex items-center justify-center shrink-0 group-hover:bg-primary/10 transition-colors">
                                                    <Book className="w-5 h-5 text-primary" />
                                                </div>
                                                <div className="overflow-hidden">
                                                    <CardTitle className="truncate text-lg group-hover:text-primary transition-colors" title={repo.name}>
                                                        {repo.name}
                                                    </CardTitle>
                                                    <CardDescription className="flex items-center gap-1 text-xs mt-1">
                                                        <Clock className="w-3 h-3" />
                                                        {new Date(repo.created_at).toLocaleDateString()}
                                                    </CardDescription>
                                                </div>
                                            </div>
                                            <Badge variant={statusConfig.variant} className={statusConfig.className + ' inline-flex items-center'}>
                                                {getStatusDisplay(repo.status).icon}
                                                {getStatusDisplay(repo.status).label}
                                            </Badge>
                                        </div>
                                    </CardHeader>
                                    <CardContent className="flex-1 pb-4">
                                        <div className="flex items-center gap-2 text-sm text-muted-foreground bg-muted/30 p-2 rounded border border-transparent group-hover:border-border/50 transition-colors">
                                            <GitFork className="w-4 h-4 shrink-0" />
                                            <span className="truncate" title={repo.url}>{repo.url.replace('https://github.com/', '')}</span>
                                            <ExternalLink className="w-3 h-3 ml-auto opacity-0 group-hover:opacity-50" />
                                        </div>
                                        <div className="mt-3 grid grid-cols-2 gap-2 text-xs">
                                            <div className="rounded bg-muted/40 p-2 border border-transparent group-hover:border-border/50 transition-colors">
                                                <span className="text-muted-foreground">{t('repository.doc_count', '文档数')}</span>
                                                <span className="ml-2 font-medium">{Array.isArray((repo as any).documents) ? (repo as any).documents.length : 0}</span>
                                            </div>
                                            <div className="rounded bg-muted/40 p-2 border border-transparent group-hover:border-border/50 transition-colors">
                                                <span className="text-muted-foreground">{t('repository.ai_summary', 'AI 概要')}</span>
                                                <span className="ml-2 font-medium">{Array.isArray((repo as any).documents) && (repo as any).documents.length > 0 ? t('common.ready', '已生成') : t('common.not_ready', '未生成')}</span>
                                            </div>
                                        </div>
                                        {repo.status === 'analyzing' && (
                                            <div className="mt-3 h-2 rounded bg-muted/50 overflow-hidden">
                                                <div className="h-full w-1/2 bg-primary/60 animate-pulse"></div>
                                            </div>
                                        )}
                                        {repo.error_msg && (
                                            <div className="mt-3 p-2 bg-destructive/5 border border-destructive/10 rounded text-xs text-destructive flex gap-2 items-start">
                                                <div className="w-1 h-full bg-destructive rounded-full shrink-0 min-h-[12px]"></div>
                                                <p className="line-clamp-2" title={repo.error_msg}>{repo.error_msg}</p>
                                            </div>
                                        )}
                                    </CardContent>
                                    <CardFooter className="pt-2 pb-4 border-t bg-muted/5 flex justify-between items-center">
                                        <div className="flex gap-2">
                                            <Button
                                                variant="default"
                                                size="sm"
                                                className="gap-1"
                                                onClick={(e) => { e.stopPropagation(); navigate(`/repo/${repo.id}`) }}
                                            >
                                                {t('repository.enter_wiki', '进入知识库')}
                                                <ChevronRight className="w-3 h-3" />
                                            </Button>
                                            <Button
                                                variant="secondary"
                                                size="sm"
                                                className="gap-1"
                                                onClick={(e) => handleRunAll(repo.id, e)}
                                                disabled={!(repo.status === 'ready' || repo.status === 'completed')}
                                                title={t('repository.run_all')}
                                            >
                                                {t('repository.rebuild', '重新分析')}
                                                <RefreshCw className="w-3 h-3" />
                                            </Button>
                                        </div>
                                        <Button
                                            variant="destructive"
                                            size="sm"
                                            className="gap-1"
                                            onClick={(e) => handleDelete(repo.id, e)}
                                            title={t('common.delete')}
                                        >
                                            {t('common.delete', '删除')}
                                            <Trash2 className="w-3 h-3" />
                                        </Button>
                                    </CardFooter>
                                </Card>
                            );
                        })}
                    </div>
                )}
            </main>
        </div>
    );
}

import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Plus, Trash2, Play, RefreshCw, Github, Settings } from 'lucide-react';
import type { Repository } from '../types';
import { repositoryApi } from '../services/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card';
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
import { Badge } from '@/components/ui/badge'; // 需要 Badge 组件

export default function Home() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const [newRepoUrl, setNewRepoUrl] = useState('');
    const [adding, setAdding] = useState(false);

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
    const getStatusVariant = (status: string): "default" | "secondary" | "destructive" | "outline" => {
        const variants: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
            pending: 'secondary',
            cloning: 'default', // blue-ish
            ready: 'outline',   // green-ish -> custom style needed or just use default
            analyzing: 'default',
            completed: 'default',
            error: 'destructive',
        };
        return variants[status] || 'secondary';
    };

    return (
        <div className="min-h-screen bg-background text-foreground">
            <header className="border-b bg-card">
                <div className="max-w-7xl mx-auto px-4 py-4 sm:px-6 lg:px-8 flex justify-between items-center">
                    <div className="flex items-center gap-2">
                        <Github className="w-8 h-8" />
                        <h1 className="text-2xl font-bold">openDeepWiki</h1>
                    </div>
                    <div className="flex gap-2 items-center">
                        <LanguageSwitcher />
                        <ThemeSwitcher />
                        <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => navigate('/config')}
                            title={t('nav.settings')}
                        >
                            <Settings className="w-5 h-5" />
                        </Button>

                        <Dialog open={showAddModal} onOpenChange={setShowAddModal}>
                            <DialogTrigger asChild>
                                <Button className="gap-2">
                                    <Plus className="w-4 h-4" />
                                    {t('repository.add')}
                                </Button>
                            </DialogTrigger>
                            <DialogContent>
                                <DialogHeader>
                                    <DialogTitle>{t('repository.add')}</DialogTitle>
                                    <DialogDescription>
                                        {t('repository.add_hint')}
                                    </DialogDescription>
                                </DialogHeader>
                                <Input
                                    value={newRepoUrl}
                                    onChange={(e) => setNewRepoUrl(e.target.value)}
                                    placeholder={t('repository.url_placeholder')}
                                    onKeyDown={(e) => e.key === 'Enter' && handleAddRepository()}
                                />
                                <DialogFooter>
                                    <Button variant="outline" onClick={() => setShowAddModal(false)}>
                                        {t('common.cancel')}
                                    </Button>
                                    <Button onClick={handleAddRepository} disabled={adding || !newRepoUrl.trim()}>
                                        {adding ? t('common.loading') : t('common.confirm')}
                                    </Button>
                                </DialogFooter>
                            </DialogContent>
                        </Dialog>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8">
                {loading ? (
                    <div className="flex justify-center items-center h-64">
                        <RefreshCw className="w-8 h-8 animate-spin text-muted-foreground" />
                    </div>
                ) : repositories.length === 0 ? (
                    <div className="text-center py-16">
                        <Github className="w-16 h-16 mx-auto text-muted-foreground mb-4" />
                        <h2 className="text-xl font-medium mb-2">{t('repository.no_repos')}</h2>
                        <p className="text-muted-foreground mb-4">{t('repository.add_hint')}</p>
                        <Button onClick={() => setShowAddModal(true)} className="gap-2">
                            <Plus className="w-4 h-4" />
                            {t('repository.add')}
                        </Button>
                    </div>
                ) : (
                    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                        {repositories.map((repo) => (
                            <Card
                                key={repo.id}
                                onClick={() => navigate(`/repo/${repo.id}`)}
                                className="cursor-pointer hover:shadow-md transition-shadow"
                            >
                                <CardHeader className="pb-2">
                                    <div className="flex justify-between items-start">
                                        <CardTitle className="truncate flex-1 pr-2" title={repo.name}>
                                            {repo.name}
                                        </CardTitle>
                                        <Badge variant={getStatusVariant(repo.status)}>
                                            {t(`repository.status.${repo.status}`)}
                                        </Badge>
                                    </div>
                                </CardHeader>
                                <CardContent className="pb-2">
                                    <p className="text-sm text-muted-foreground truncate" title={repo.url}>{repo.url}</p>
                                    {repo.error_msg && (
                                        <p className="text-sm text-destructive truncate mt-2" title={repo.error_msg}>{repo.error_msg}</p>
                                    )}
                                </CardContent>
                                <CardFooter className="justify-between pt-2">
                                    <span className="text-xs text-muted-foreground">
                                        {new Date(repo.created_at).toLocaleDateString()}
                                    </span>
                                    <div className="flex gap-2">
                                        {(repo.status === 'ready' || repo.status === 'completed') && (
                                            <Button
                                                variant="ghost"
                                                size="icon"
                                                className="h-8 w-8 text-blue-600 hover:text-blue-700 hover:bg-blue-50 dark:hover:bg-blue-950"
                                                onClick={(e) => handleRunAll(repo.id, e)}
                                                title={t('repository.run_all')}
                                            >
                                                <Play className="w-4 h-4" />
                                            </Button>
                                        )}
                                        <Button
                                            variant="ghost"
                                            size="icon"
                                            className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                                            onClick={(e) => handleDelete(repo.id, e)}
                                            title={t('common.delete')}
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </Button>
                                    </div>
                                </CardFooter>
                            </Card>
                        ))}
                    </div>
                )}
            </main>
        </div>
    );
}

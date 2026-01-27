import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Trash2, Play, RefreshCw, Github, Settings } from 'lucide-react';
import type { Repository } from '../types';
import { repositoryApi } from '../services/api';

export default function Home() {
    const [repositories, setRepositories] = useState<Repository[]>([]);
    const [loading, setLoading] = useState(true);
    const [showAddModal, setShowAddModal] = useState(false);
    const [newRepoUrl, setNewRepoUrl] = useState('');
    const [adding, setAdding] = useState(false);
    const navigate = useNavigate();

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
            alert('Failed to add repository');
        } finally {
            setAdding(false);
        }
    };

    const handleDelete = async (id: number, e: React.MouseEvent) => {
        e.stopPropagation();
        if (!confirm('Are you sure you want to delete this repository?')) return;
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

    const getStatusColor = (status: string) => {
        const colors: Record<string, string> = {
            pending: 'bg-gray-100 text-gray-800',
            cloning: 'bg-blue-100 text-blue-800',
            ready: 'bg-green-100 text-green-800',
            analyzing: 'bg-yellow-100 text-yellow-800',
            completed: 'bg-emerald-100 text-emerald-800',
            error: 'bg-red-100 text-red-800',
        };
        return colors[status] || 'bg-gray-100 text-gray-800';
    };

    const getStatusText = (status: string) => {
        const texts: Record<string, string> = {
            pending: '等待中',
            cloning: '克隆中',
            ready: '就绪',
            analyzing: '分析中',
            completed: '已完成',
            error: '错误',
        };
        return texts[status] || status;
    };

    return (
        <div className="min-h-screen bg-gray-50">
            <header className="bg-white shadow-sm">
                <div className="max-w-7xl mx-auto px-4 py-4 sm:px-6 lg:px-8 flex justify-between items-center">
                    <div className="flex items-center gap-2">
                        <Github className="w-8 h-8 text-gray-800" />
                        <h1 className="text-2xl font-bold text-gray-900">openDeepWiki</h1>
                    </div>
                    <div className="flex gap-2">
                        <button
                            onClick={() => navigate('/config')}
                            className="p-2 text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg"
                            title="Settings"
                        >
                            <Settings className="w-5 h-5" />
                        </button>
                        <button
                            onClick={() => setShowAddModal(true)}
                            className="flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition"
                        >
                            <Plus className="w-5 h-5" />
                            添加仓库
                        </button>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto px-4 py-8 sm:px-6 lg:px-8">
                {loading ? (
                    <div className="flex justify-center items-center h-64">
                        <RefreshCw className="w-8 h-8 animate-spin text-gray-400" />
                    </div>
                ) : repositories.length === 0 ? (
                    <div className="text-center py-16">
                        <Github className="w-16 h-16 mx-auto text-gray-300 mb-4" />
                        <h2 className="text-xl font-medium text-gray-600 mb-2">还没有仓库</h2>
                        <p className="text-gray-500 mb-4">添加一个 GitHub 仓库开始解读</p>
                        <button
                            onClick={() => setShowAddModal(true)}
                            className="inline-flex items-center gap-2 bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
                        >
                            <Plus className="w-5 h-5" />
                            添加仓库
                        </button>
                    </div>
                ) : (
                    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                        {repositories.map((repo) => (
                            <div
                                key={repo.id}
                                onClick={() => navigate(`/repo/${repo.id}`)}
                                className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 cursor-pointer hover:shadow-md transition"
                            >
                                <div className="flex justify-between items-start mb-2">
                                    <h3 className="font-semibold text-lg text-gray-900 truncate flex-1">
                                        {repo.name}
                                    </h3>
                                    <span className={`px-2 py-1 text-xs rounded-full ${getStatusColor(repo.status)}`}>
                                        {getStatusText(repo.status)}
                                    </span>
                                </div>
                                <p className="text-sm text-gray-500 truncate mb-4">{repo.url}</p>
                                {repo.error_msg && (
                                    <p className="text-sm text-red-500 truncate mb-2">{repo.error_msg}</p>
                                )}
                                <div className="flex justify-between items-center">
                                    <span className="text-xs text-gray-400">
                                        {new Date(repo.created_at).toLocaleDateString()}
                                    </span>
                                    <div className="flex gap-2">
                                        {(repo.status === 'ready' || repo.status === 'completed') && (
                                            <button
                                                onClick={(e) => handleRunAll(repo.id, e)}
                                                className="p-1.5 text-blue-600 hover:bg-blue-50 rounded"
                                                title="运行所有任务"
                                            >
                                                <Play className="w-4 h-4" />
                                            </button>
                                        )}
                                        <button
                                            onClick={(e) => handleDelete(repo.id, e)}
                                            className="p-1.5 text-red-600 hover:bg-red-50 rounded"
                                            title="删除"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </div>
                                </div>
                            </div>
                        ))}
                    </div>
                )}
            </main>

            {showAddModal && (
                <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
                    <div className="bg-white rounded-lg shadow-xl max-w-md w-full p-6">
                        <h2 className="text-xl font-semibold mb-4">添加仓库</h2>
                        <input
                            type="text"
                            value={newRepoUrl}
                            onChange={(e) => setNewRepoUrl(e.target.value)}
                            placeholder="https://github.com/username/repo"
                            className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 mb-4"
                            onKeyDown={(e) => e.key === 'Enter' && handleAddRepository()}
                        />
                        <div className="flex justify-end gap-2">
                            <button
                                onClick={() => setShowAddModal(false)}
                                className="px-4 py-2 text-gray-600 hover:bg-gray-100 rounded-lg"
                            >
                                取消
                            </button>
                            <button
                                onClick={handleAddRepository}
                                disabled={adding || !newRepoUrl.trim()}
                                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                            >
                                {adding ? '添加中...' : '添加'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}

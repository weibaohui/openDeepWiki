import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Save, Eye, EyeOff } from 'lucide-react';
import type { Config } from '../types';
import { configApi } from '../services/api';

export default function ConfigPage() {
    const navigate = useNavigate();
    const [_config, setConfig] = useState<Config | null>(null);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [showApiKey, setShowApiKey] = useState(false);
    const [showGithubToken, setShowGithubToken] = useState(false);

    const [formData, setFormData] = useState({
        llm_api_url: '',
        llm_api_key: '',
        llm_model: '',
        llm_max_tokens: 4096,
        github_token: '',
    });

    useEffect(() => {
        const fetchConfig = async () => {
            try {
                const { data } = await configApi.get();
                setConfig(data);
                setFormData({
                    llm_api_url: data.llm.api_url,
                    llm_api_key: data.llm.api_key,
                    llm_model: data.llm.model,
                    llm_max_tokens: data.llm.max_tokens,
                    github_token: data.github.token,
                });
            } catch (error) {
                console.error('Failed to fetch config:', error);
            } finally {
                setLoading(false);
            }
        };
        fetchConfig();
    }, []);

    const handleSave = async () => {
        setSaving(true);
        try {
            await configApi.update({
                llm: {
                    api_url: formData.llm_api_url,
                    api_key: formData.llm_api_key,
                    model: formData.llm_model,
                    max_tokens: formData.llm_max_tokens,
                },
                github: {
                    token: formData.github_token,
                },
            });
            alert('配置已保存');
        } catch (error) {
            console.error('Failed to save config:', error);
            alert('保存失败');
        } finally {
            setSaving(false);
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="animate-spin w-8 h-8 border-4 border-blue-500 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50">
            <header className="bg-white shadow-sm">
                <div className="max-w-3xl mx-auto px-4 py-4 sm:px-6 lg:px-8 flex items-center gap-4">
                    <button
                        onClick={() => navigate('/')}
                        className="p-2 hover:bg-gray-100 rounded-lg"
                    >
                        <ArrowLeft className="w-5 h-5" />
                    </button>
                    <h1 className="text-xl font-bold text-gray-900">设置</h1>
                </div>
            </header>

            <main className="max-w-3xl mx-auto px-4 py-8 sm:px-6 lg:px-8">
                <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
                    <h2 className="text-lg font-semibold mb-6">LLM 配置</h2>

                    <div className="space-y-4">
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                API 地址
                            </label>
                            <input
                                type="text"
                                value={formData.llm_api_url}
                                onChange={(e) => setFormData({ ...formData, llm_api_url: e.target.value })}
                                placeholder="https://api.openai.com/v1"
                                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                API Key
                            </label>
                            <div className="relative">
                                <input
                                    type={showApiKey ? 'text' : 'password'}
                                    value={formData.llm_api_key}
                                    onChange={(e) => setFormData({ ...formData, llm_api_key: e.target.value })}
                                    placeholder="sk-..."
                                    className="w-full px-3 py-2 pr-10 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                                />
                                <button
                                    type="button"
                                    onClick={() => setShowApiKey(!showApiKey)}
                                    className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-gray-600"
                                >
                                    {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                                </button>
                            </div>
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                模型
                            </label>
                            <input
                                type="text"
                                value={formData.llm_model}
                                onChange={(e) => setFormData({ ...formData, llm_model: e.target.value })}
                                placeholder="gpt-4o"
                                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            />
                        </div>

                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-1">
                                最大 Token 数
                            </label>
                            <input
                                type="number"
                                value={formData.llm_max_tokens}
                                onChange={(e) => setFormData({ ...formData, llm_max_tokens: Number(e.target.value) })}
                                className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            />
                        </div>
                    </div>

                    <hr className="my-6" />

                    <h2 className="text-lg font-semibold mb-6">GitHub 配置</h2>

                    <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">
                            GitHub Token (用于访问私有仓库)
                        </label>
                        <div className="relative">
                            <input
                                type={showGithubToken ? 'text' : 'password'}
                                value={formData.github_token}
                                onChange={(e) => setFormData({ ...formData, github_token: e.target.value })}
                                placeholder="ghp_..."
                                className="w-full px-3 py-2 pr-10 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                            />
                            <button
                                type="button"
                                onClick={() => setShowGithubToken(!showGithubToken)}
                                className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-gray-400 hover:text-gray-600"
                            >
                                {showGithubToken ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                            </button>
                        </div>
                    </div>

                    <div className="mt-6 flex justify-end">
                        <button
                            onClick={handleSave}
                            disabled={saving}
                            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                        >
                            <Save className="w-4 h-4" />
                            {saving ? '保存中...' : '保存配置'}
                        </button>
                    </div>
                </div>
            </main>
        </div>
    );
}

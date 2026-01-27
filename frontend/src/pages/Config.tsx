import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, Save, Eye, EyeOff, Loader2 } from 'lucide-react';
import type { Config } from '../types';
import { configApi } from '../services/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card';
import { Separator } from '@/components/ui/separator';
import { ThemeSwitcher } from '@/components/common/ThemeSwitcher';
import { LanguageSwitcher } from '@/components/common/LanguageSwitcher';

export default function ConfigPage() {
    const { t } = useTranslation();
    const navigate = useNavigate();
    const [, setConfig] = useState<Config | null>(null);
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
            alert(t('settings.save_success'));
        } catch (error) {
            console.error('Failed to save config:', error);
            alert(t('settings.save_failed'));
        } finally {
            setSaving(false);
        }
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-background text-foreground">
            <header className="border-b bg-card">
                <div className="max-w-3xl mx-auto px-4 py-4 sm:px-6 lg:px-8 flex items-center gap-4">
                    <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => navigate('/')}
                    >
                        <ArrowLeft className="w-5 h-5" />
                    </Button>
                    <h1 className="text-xl font-bold flex-1">{t('settings.title')}</h1>
                    <LanguageSwitcher />
                    <ThemeSwitcher />
                </div>
            </header>

            <main className="max-w-3xl mx-auto px-4 py-8 sm:px-6 lg:px-8">
                <Card>
                    <CardHeader>
                        <CardTitle>{t('settings.llm_config')}</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        <div className="grid gap-2">
                            <Label htmlFor="api_url">{t('settings.api_url')}</Label>
                            <Input
                                id="api_url"
                                type="text"
                                value={formData.llm_api_url}
                                onChange={(e) => setFormData({ ...formData, llm_api_url: e.target.value })}
                                placeholder="https://api.openai.com/v1"
                            />
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="api_key">{t('settings.api_key')}</Label>
                            <div className="relative">
                                <Input
                                    id="api_key"
                                    type={showApiKey ? 'text' : 'password'}
                                    value={formData.llm_api_key}
                                    onChange={(e) => setFormData({ ...formData, llm_api_key: e.target.value })}
                                    placeholder="sk-..."
                                    className="pr-10"
                                />
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    onClick={() => setShowApiKey(!showApiKey)}
                                    className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                                >
                                    {showApiKey ? <EyeOff className="w-4 h-4 text-muted-foreground" /> : <Eye className="w-4 h-4 text-muted-foreground" />}
                                </Button>
                            </div>
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="model">{t('settings.model')}</Label>
                            <Input
                                id="model"
                                type="text"
                                value={formData.llm_model}
                                onChange={(e) => setFormData({ ...formData, llm_model: e.target.value })}
                                placeholder="gpt-4o"
                            />
                        </div>

                        <div className="grid gap-2">
                            <Label htmlFor="max_tokens">{t('settings.max_tokens')}</Label>
                            <Input
                                id="max_tokens"
                                type="number"
                                value={formData.llm_max_tokens}
                                onChange={(e) => setFormData({ ...formData, llm_max_tokens: Number(e.target.value) })}
                            />
                        </div>
                    </CardContent>

                    <Separator />

                    <CardHeader>
                        <CardTitle>{t('settings.github_config')}</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="grid gap-2">
                            <Label htmlFor="github_token">{t('settings.github_token')}</Label>
                            <div className="relative">
                                <Input
                                    id="github_token"
                                    type={showGithubToken ? 'text' : 'password'}
                                    value={formData.github_token}
                                    onChange={(e) => setFormData({ ...formData, github_token: e.target.value })}
                                    placeholder="ghp_..."
                                    className="pr-10"
                                />
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="icon"
                                    onClick={() => setShowGithubToken(!showGithubToken)}
                                    className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                                >
                                    {showGithubToken ? <EyeOff className="w-4 h-4 text-muted-foreground" /> : <Eye className="w-4 h-4 text-muted-foreground" />}
                                </Button>
                            </div>
                        </div>
                    </CardContent>

                    <CardFooter className="justify-end pt-4">
                        <Button
                            onClick={handleSave}
                            disabled={saving}
                            className="gap-2"
                        >
                            {saving ? (
                                <Loader2 className="w-4 h-4 animate-spin" />
                            ) : (
                                <Save className="w-4 h-4" />
                            )}
                            {saving ? t('settings.saving') : t('common.save')}
                        </Button>
                    </CardFooter>
                </Card>
            </main>
        </div>
    );
}

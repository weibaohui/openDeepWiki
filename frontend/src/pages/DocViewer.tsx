import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { ArrowLeft, FileText, Download, Edit2, Save, X, Loader2 } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import type { Document } from '../types';
import { documentApi } from '../services/api';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';

export default function DocViewer() {
    const { t } = useTranslation();
    const { id, docId } = useParams<{ id: string; docId: string }>();
    const navigate = useNavigate();
    const [document, setDocument] = useState<Document | null>(null);
    const [documents, setDocuments] = useState<Document[]>([]);
    const [loading, setLoading] = useState(true);
    const [editing, setEditing] = useState(false);
    const [editContent, setEditContent] = useState('');

    useEffect(() => {
        const fetchData = async () => {
            if (!id || !docId) return;
            try {
                const [docRes, docsRes] = await Promise.all([
                    documentApi.get(Number(docId)),
                    documentApi.getByRepository(Number(id)),
                ]);
                setDocument(docRes.data);
                setDocuments(docsRes.data);
                setEditContent(docRes.data.content);
            } catch (error) {
                console.error('Failed to fetch document:', error);
            } finally {
                setLoading(false);
            }
        };
        fetchData();
    }, [id, docId]);

    const handleSave = async () => {
        if (!docId) return;
        try {
            const { data } = await documentApi.update(Number(docId), editContent);
            setDocument(data);
            setEditing(false);
        } catch (error) {
            console.error('Failed to save document:', error);
        }
    };

    const handleDownload = () => {
        if (!document) return;
        const blob = new Blob([document.content], { type: 'text/markdown' });
        const url = window.URL.createObjectURL(blob);
        const a = window.document.createElement('a');
        a.href = url;
        a.download = document.filename;
        a.click();
        window.URL.revokeObjectURL(url);
    };

    if (loading) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (!document) {
        return (
            <div className="min-h-screen bg-background flex items-center justify-center">
                <p className="text-muted-foreground">{t('repository.not_found')}</p>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-background flex text-foreground">
            {/* Sidebar */}
            <aside className="w-64 bg-card border-r border-border flex-shrink-0">
                <div className="p-4 border-b border-border">
                    <Button
                        variant="ghost"
                        className="w-full justify-start pl-0 hover:bg-transparent"
                        onClick={() => navigate(`/repo/${id}`)}
                    >
                        <ArrowLeft className="w-4 h-4 mr-2" />
                        {t('repository.title')}
                    </Button>
                </div>
                <nav className="p-2">
                    <p className="px-3 py-2 text-xs font-medium text-muted-foreground uppercase">{t('repository.docs')}</p>
                    {documents.map((doc) => (
                        <Button
                            key={doc.id}
                            variant={doc.id === Number(docId) ? "secondary" : "ghost"}
                            className="w-full justify-start mb-1"
                            onClick={() => navigate(`/repo/${id}/doc/${doc.id}`)}
                        >
                            <FileText className="w-4 h-4 mr-2" />
                            <span className="truncate">{doc.title}</span>
                        </Button>
                    ))}
                </nav>
            </aside>

            {/* Main Content */}
            <main className="flex-1 overflow-auto bg-background">
                <header className="sticky top-0 bg-card/80 backdrop-blur-sm border-b border-border px-6 py-4 flex items-center justify-between z-10">
                    <h1 className="text-xl font-semibold truncate pr-4">{document.title}</h1>
                    <div className="flex gap-2 shrink-0">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={handleDownload}
                        >
                            <Download className="w-4 h-4 mr-2" />
                            {t('common.save')}
                        </Button>
                        {editing ? (
                            <>
                                <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() => {
                                        setEditing(false);
                                        setEditContent(document.content);
                                    }}
                                >
                                    <X className="w-4 h-4 mr-2" />
                                    {t('common.cancel')}
                                </Button>
                                <Button
                                    size="sm"
                                    onClick={handleSave}
                                >
                                    <Save className="w-4 h-4 mr-2" />
                                    {t('common.save')}
                                </Button>
                            </>
                        ) : (
                            <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setEditing(true)}
                            >
                                <Edit2 className="w-4 h-4 mr-2" />
                                {t('common.edit')}
                            </Button>
                        )}
                    </div>
                </header>

                <div className="p-6 max-w-4xl mx-auto">
                    {editing ? (
                        <textarea
                            value={editContent}
                            onChange={(e) => setEditContent(e.target.value)}
                            className="w-full h-[calc(100vh-200px)] p-4 font-mono text-sm border border-input bg-transparent rounded-lg focus:ring-2 focus:ring-ring focus:outline-none resize-none"
                        />
                    ) : (
                        <Card className="p-8 border-none shadow-none bg-transparent">
                            <article className="prose prose-slate dark:prose-invert max-w-none">
                                <ReactMarkdown>{document.content}</ReactMarkdown>
                            </article>
                        </Card>
                    )}
                </div>
            </main>
        </div>
    );
}

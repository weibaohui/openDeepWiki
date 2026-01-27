import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { ArrowLeft, FileText, Download, Edit2, Save, X } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import type { Document } from '../types';
import { documentApi } from '../services/api';

export default function DocViewer() {
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
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <div className="animate-spin w-8 h-8 border-4 border-blue-500 border-t-transparent rounded-full"></div>
            </div>
        );
    }

    if (!document) {
        return (
            <div className="min-h-screen bg-gray-50 flex items-center justify-center">
                <p className="text-gray-500">文档不存在</p>
            </div>
        );
    }

    return (
        <div className="min-h-screen bg-gray-50 flex">
            {/* Sidebar */}
            <aside className="w-64 bg-white border-r border-gray-200 flex-shrink-0">
                <div className="p-4 border-b border-gray-200">
                    <button
                        onClick={() => navigate(`/repo/${id}`)}
                        className="flex items-center gap-2 text-gray-600 hover:text-gray-900"
                    >
                        <ArrowLeft className="w-4 h-4" />
                        返回仓库
                    </button>
                </div>
                <nav className="p-2">
                    <p className="px-3 py-2 text-xs font-medium text-gray-500 uppercase">文档目录</p>
                    {documents.map((doc) => (
                        <button
                            key={doc.id}
                            onClick={() => navigate(`/repo/${id}/doc/${doc.id}`)}
                            className={`w-full flex items-center gap-2 px-3 py-2 text-sm rounded-lg transition ${doc.id === Number(docId)
                                    ? 'bg-blue-50 text-blue-700'
                                    : 'text-gray-700 hover:bg-gray-100'
                                }`}
                        >
                            <FileText className="w-4 h-4" />
                            {doc.title}
                        </button>
                    ))}
                </nav>
            </aside>

            {/* Main Content */}
            <main className="flex-1 overflow-auto">
                <header className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
                    <h1 className="text-xl font-semibold text-gray-900">{document.title}</h1>
                    <div className="flex gap-2">
                        <button
                            onClick={handleDownload}
                            className="flex items-center gap-2 px-3 py-1.5 text-gray-600 hover:bg-gray-100 rounded-lg"
                        >
                            <Download className="w-4 h-4" />
                            下载
                        </button>
                        {editing ? (
                            <>
                                <button
                                    onClick={() => {
                                        setEditing(false);
                                        setEditContent(document.content);
                                    }}
                                    className="flex items-center gap-2 px-3 py-1.5 text-gray-600 hover:bg-gray-100 rounded-lg"
                                >
                                    <X className="w-4 h-4" />
                                    取消
                                </button>
                                <button
                                    onClick={handleSave}
                                    className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                                >
                                    <Save className="w-4 h-4" />
                                    保存
                                </button>
                            </>
                        ) : (
                            <button
                                onClick={() => setEditing(true)}
                                className="flex items-center gap-2 px-3 py-1.5 text-gray-600 hover:bg-gray-100 rounded-lg"
                            >
                                <Edit2 className="w-4 h-4" />
                                编辑
                            </button>
                        )}
                    </div>
                </header>

                <div className="p-6 max-w-4xl mx-auto">
                    {editing ? (
                        <textarea
                            value={editContent}
                            onChange={(e) => setEditContent(e.target.value)}
                            className="w-full h-[calc(100vh-200px)] p-4 font-mono text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                        />
                    ) : (
                        <article className="prose prose-slate max-w-none">
                            <ReactMarkdown>{document.content}</ReactMarkdown>
                        </article>
                    )}
                </div>
            </main>
        </div>
    );
}

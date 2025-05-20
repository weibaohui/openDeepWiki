import React, { useEffect, useRef, useState } from 'react';
import { appendQueryParam, replacePlaceholders } from "@/utils/utils.ts";
import { render as amisRender } from "amis";

// 定义组件的 Props 接口
interface SSEComponentProps {
    url: string;
    data: {
        tailLines?: number;

    };
}

// SSE 组件，使用 forwardRef 让父组件可以手动控制
const SSELogDisplayComponent = React.forwardRef((props: SSEComponentProps, _) => {
    const url = replacePlaceholders(props.url, props.data);
    const params = {
        tailLines: props.data.tailLines,
    };
    // @ts-ignore
    let finalUrl = appendQueryParam(url, params);
    const token = localStorage.getItem('token');
    //拼接url token
    finalUrl = finalUrl + (finalUrl.includes('?') ? '&' : '?') + `token=${token}`;


    const dom = useRef<HTMLDivElement | null>(null);
    const eventSourceRef = useRef<EventSource | null>(null);
    const [errorMessage, setErrorMessage] = useState('');
    const [lines, setLines] = useState<string[]>([]);


    // 连接 SSE 服务器
    const connectSSE = () => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
        }

        eventSourceRef.current = new EventSource(finalUrl);

        eventSourceRef.current.addEventListener('message', (event) => {
            const newLine = event.data;
            setLines((prevLines) => [...prevLines, newLine]);
        });
        eventSourceRef.current.addEventListener('open', (_) => {
            // setErrorMessage('Connected');
        });
        eventSourceRef.current.addEventListener('error', (_) => {
            if (eventSourceRef.current?.readyState === EventSource.CLOSED) {
                // setErrorMessage('连接已关闭');
            } else if (eventSourceRef.current?.readyState === EventSource.CONNECTING) {
                // setErrorMessage('正在尝试重新连接...');
            } else {
                // setErrorMessage('发生未知错误...');
            }
            eventSourceRef.current?.close();
        });
    };

    // 关闭 SSE 连接
    const disconnectSSE = () => {
        if (eventSourceRef.current) {
            eventSourceRef.current.close();
            eventSourceRef.current = null;
        }
    };

    useEffect(() => {
        setLines([]); // 清空日志
        setErrorMessage('');
        connectSSE();
        return () => {
            disconnectSSE();
        };
    }, [finalUrl]);


    // 创建一个转换器实
    const markdownContent = lines.join("");

    // 每次 lines 变化时自动滚动到底部（使用 scrollIntoView 方式增强兼容性）
    const bottomRef = useRef<HTMLDivElement | null>(null);
    useEffect(() => {
        if (bottomRef.current) {
            bottomRef.current.scrollIntoView({ behavior: 'auto' });
        }
    }, [lines]);

    return (
        <div style={{ padding: '4px', borderRadius: '4px', height: 'calc(100vh)', overflow: 'auto' }}>
            <div ref={dom} style={{ whiteSpace: 'pre-wrap', padding: '10px' }}>
                {errorMessage && <div
                    style={{ color: errorMessage == "Connected" ? '#00FF00' : 'red' }}>{errorMessage} 共计：{lines.length}行</div>}

                <pre style={{ whiteSpace: 'pre-wrap' }}>
                    {
                        (amisRender({
                            type: "markdown",
                            value: markdownContent,
                            options: {
                                linkify: true,
                                html: true,
                                breaks: true
                            },
                        }))
                    }
                </pre>
                <div ref={bottomRef}></div>
            </div>
        </div>
    );
});

export default SSELogDisplayComponent;

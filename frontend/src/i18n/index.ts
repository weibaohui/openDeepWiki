import type { Locale } from 'antd/es/locale';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import appZhCN from './locales/zh-CN.json';
import appEnUS from './locales/en-US.json';

export type AppLocale = 'zh-CN' | 'en-US';

export const antdLocales: Record<AppLocale, Locale> = {
    'zh-CN': zhCN,
    'en-US': enUS,
};

export const messages = {
    'zh-CN': appZhCN,
    'en-US': appEnUS,
};

export function getMessage(locale: AppLocale, path: string, fallback?: string) {
    const parts = path.split('.');
    let current: unknown = messages[locale];
    for (const key of parts) {
        if (current && typeof current === 'object') {
            current = (current as Record<string, unknown>)[key];
        } else {
            current = undefined;
        }
    }
    if (typeof current === 'string') {
        return current;
    }
    return fallback ?? path;
}

import { createContext, useContext } from 'react';
import type { AppLocale } from '@/i18n';
import type { ThemeMode } from '@/types';

export interface AppConfigContextValue {
    locale: AppLocale;
    setLocale: (locale: AppLocale) => void;
    themeMode: ThemeMode;
    setThemeMode: (mode: ThemeMode) => void;
    t: (key: string, fallback?: string) => string;
}

export const AppConfigContext = createContext<AppConfigContextValue | null>(null);

export function useAppConfig() {
    const ctx = useContext(AppConfigContext);
    if (!ctx) {
        throw new Error('AppConfigContext 未初始化');
    }
    return ctx;
}

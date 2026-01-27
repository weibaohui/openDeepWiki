import { ConfigProvider, theme as antdTheme } from 'antd';
import type { ReactNode } from 'react';
import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { antdLocales, getMessage, type AppLocale } from '@/i18n';

export type ThemeMode = 'default' | 'dark' | 'compact';

interface AppConfigContextValue {
    locale: AppLocale;
    setLocale: (locale: AppLocale) => void;
    themeMode: ThemeMode;
    setThemeMode: (mode: ThemeMode) => void;
    t: (key: string, fallback?: string) => string;
}

const AppConfigContext = createContext<AppConfigContextValue | null>(null);

interface Props {
    children: ReactNode;
}

export function ThemeProvider({ children }: Props) {
    const [locale, setLocale] = useState<AppLocale>(() => {
        const stored = window.localStorage.getItem('opendeepwiki-locale');
        return stored === 'en-US' ? 'en-US' : 'zh-CN';
    });
    const [themeMode, setThemeMode] = useState<ThemeMode>(() => {
        const stored = window.localStorage.getItem('opendeepwiki-theme-mode');
        return stored === 'dark' || stored === 'compact' ? stored : 'default';
    });

    useEffect(() => {
        window.localStorage.setItem('opendeepwiki-locale', locale);
    }, [locale]);

    useEffect(() => {
        window.localStorage.setItem('opendeepwiki-theme-mode', themeMode);
    }, [themeMode]);

    const algorithm = useMemo(() => {
        if (themeMode === 'dark') return antdTheme.darkAlgorithm;
        if (themeMode === 'compact') return antdTheme.compactAlgorithm;
        return antdTheme.defaultAlgorithm;
    }, [themeMode]);

    const t = useCallback(
        (key: string, fallback?: string) => getMessage(locale, key, fallback),
        [locale]
    );

    const value = useMemo(
        () => ({ locale, setLocale, themeMode, setThemeMode, t }),
        [locale, themeMode, t]
    );

    return (
        <AppConfigContext.Provider value={value}>
            <ConfigProvider locale={antdLocales[locale]} theme={{ algorithm }}>
                {children}
            </ConfigProvider>
        </AppConfigContext.Provider>
    );
}

export function useAppConfig() {
    const ctx = useContext(AppConfigContext);
    if (!ctx) {
        throw new Error('AppConfigContext 未初始化');
    }
    return ctx;
}

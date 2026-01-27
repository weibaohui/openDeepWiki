import { ConfigProvider, theme as antdTheme } from 'antd';
import type { ReactNode } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { antdLocales, getMessage, type AppLocale } from '@/i18n';
import { AppConfigContext } from '@/context/AppConfigContext';
import type { ThemeMode } from '@/types';

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

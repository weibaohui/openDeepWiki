import { Select } from 'antd';
import { useAppConfig } from '@/context/AppConfigContext';
import type { AppLocale } from '@/i18n';

const languageOptions = [
    { value: 'zh-CN', label: '简体中文' },
    { value: 'en-US', label: 'English' },
];

export function LanguageSwitcher() {
    const { locale, setLocale } = useAppConfig();

    return (
        <Select<AppLocale>
            value={locale}
            options={languageOptions}
            onChange={(value: AppLocale) => setLocale(value)}
            style={{ width: 140 }}
        />
    );
}

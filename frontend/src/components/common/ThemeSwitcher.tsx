import { Select } from 'antd';
import { useAppConfig } from '@/context/AppConfigContext';
import type { ThemeMode } from '@/types';

export function ThemeSwitcher() {
    const { themeMode, setThemeMode, t } = useAppConfig();

    const options = [
        { value: 'default', label: t('theme.light', '浅色') },
        { value: 'dark', label: t('theme.dark', '深色') },
        { value: 'compact', label: t('theme.system', '紧凑') },
    ];

    return (
        <Select<ThemeMode>
            value={themeMode}
            options={options}
            onChange={(value: ThemeMode) => setThemeMode(value)}
            style={{ width: 120 }}
        />
    );
}

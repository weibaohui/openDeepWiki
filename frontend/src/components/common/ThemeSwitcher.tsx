import { Select } from 'antd';
import { useAppConfig, type ThemeMode } from '@/providers/ThemeProvider';

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

import { useTranslation } from 'react-i18next';
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from '@/components/ui/select';
import { Languages } from 'lucide-react';

const languages = [
    { code: 'zh-CN', name: '简体中文', flag: '🇨🇳' },
    { code: 'en-US', name: 'English', flag: '🇺🇸' },
];

export function LanguageSwitcher() {
    const { i18n } = useTranslation();

    const handleChange = (value: string) => {
        i18n.changeLanguage(value);
    };

    return (
        <Select value={i18n.language} onValueChange={handleChange}>
            <SelectTrigger className="w-[140px]">
                <Languages className="mr-2 h-4 w-4" />
                <SelectValue />
            </SelectTrigger>
            <SelectContent>
                {languages.map((lang) => (
                    <SelectItem key={lang.code} value={lang.code}>
                        <span className="mr-2">{lang.flag}</span>
                        {lang.name}
                    </SelectItem>
                ))}
            </SelectContent>
        </Select>
    );
}

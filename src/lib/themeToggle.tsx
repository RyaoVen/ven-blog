/** 深浅色切换：纯圆按钮（icon only），悬浮时展开显示文字 */

import { useEffect, useState } from "react";
import { MoonIcon, SunIcon } from "./icons";

const STORAGE_KEY = "ven-theme";

type Theme = "light" | "dark";

function applyTheme(theme: Theme): void {
    document.documentElement.dataset.theme = theme;
}

export function ThemeToggle() {
    // SSR 与水合首帧为 null（渲染默认浅色 moon 图标），挂载后按存储/系统矫正
    const [theme, setTheme] = useState<Theme | null>(null);

    useEffect(() => {
        const stored = localStorage.getItem(STORAGE_KEY) as Theme | null;
        const initial: Theme =
            stored ?? (window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light");
        setTheme(initial);
        applyTheme(initial);
    }, []);

    function toggle() {
        const next: Theme = theme === "dark" ? "light" : "dark";
        setTheme(next);
        applyTheme(next);
        localStorage.setItem(STORAGE_KEY, next);
    }

    const dark = theme === "dark";
    return (
        <button type="button" className="ven-theme-toggle" onClick={toggle} aria-label="切换深浅色主题">
            {dark ? <SunIcon size={15} /> : <MoonIcon size={15} />}
            <span className="ven-theme-label">{dark ? "浅色" : "深色"}</span>
        </button>
    );
}

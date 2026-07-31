/** 深浅色切换：切换 documentElement[data-theme] + localStorage 持久化；无存储值跟随系统偏好 */

import { useEffect, useState } from "react";
import { MoonIcon, SunIcon } from "./icons";

const STORAGE_KEY = "ven-theme";

type Theme = "light" | "dark";

function applyTheme(theme: Theme): void {
    document.documentElement.dataset.theme = theme;
}

export function ThemeToggle() {
    // SSR 与水合首帧为 null（统一渲染"主题"），挂载后按存储/系统矫正，避免 hydration mismatch
    const [theme, setTheme] = useState<Theme | null>(null);

    useEffect(() => {
        // SSR 与水合首帧为 null（统一渲染"主题"），挂载后按存储/系统矫正，避免 hydration mismatch
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

    return (
        <button type="button" className="ven-btn" onClick={toggle} aria-label="切换深浅色主题">
            {theme === null ? (
                "主题"
            ) : theme === "dark" ? (
                <>
                    <SunIcon />
                    浅色
                </>
            ) : (
                <>
                    <MoonIcon />
                    深色
                </>
            )}
        </button>
    );
}

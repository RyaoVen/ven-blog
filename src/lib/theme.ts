/** 设计 token：组件内联样式引用这里，全局 CSS 变量定义见 globalCss.ts */

/** CSS 变量引用（与 globalCss.ts 中 :root 定义一一对应） */
export const v = {
    bg: "var(--bg)",
    bgSubtle: "var(--bg-subtle)",
    bgInset: "var(--bg-inset)",
    border: "var(--border)",
    borderStrong: "var(--border-strong)",
    text: "var(--text)",
    textSecondary: "var(--text-secondary)",
    textMuted: "var(--text-muted)",
    accent: "var(--accent)",
    accentHover: "var(--accent-hover)",
    primary: "var(--primary)",
    primaryHover: "var(--primary-hover)",
    primaryFg: "var(--primary-fg)",
    danger: "var(--danger)",
    shadowCard: "var(--shadow-card)",
    shadowCardHover: "var(--shadow-card-hover)",
} as const;

/** 圆角 */
export const radius = {
    sm: "var(--radius-sm)",
    md: "var(--radius-md)",
    lg: "var(--radius-lg)",
} as const;

/** 布局宽度 */
export const layout = {
    container: 880,
    prose: 760,
} as const;

/** 字体族（与 globalCss.ts 一致） */
export const fontFamily = {
    sans: 'system-ui, -apple-system, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif',
    serif: 'Georgia, "Noto Serif SC", "Songti SC", "SimSun", serif',
    mono: '"JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace',
} as const;

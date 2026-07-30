/** 全局样式：工业极简——单色系、发丝线（hairline）、锐利边角、排版驱动；
 * CSS 变量（浅色落地 + 暗色预留）、元素基线、通用工具类；经 Layout 的 <style> 注入 */

export const globalCss = `
/* ===== 设计变量：浅色主题（工业极简·单色） ===== */
:root {
    --bg: #ffffff;
    --bg-subtle: #fafafa;
    --bg-inset: #f4f4f4;
    --border: #e5e5e5;
    --border-strong: #d4d4d4;
    --text: #111111;
    --text-secondary: #525252;
    --text-muted: #8a8a8a;
    --accent: #111111;
    --accent-hover: #000000;
    --primary: #111111;
    --primary-hover: #333333;
    --primary-fg: #ffffff;
    --danger: #b91c1c;
    --shadow-card: none;
    --shadow-card-hover: none;
    --radius-sm: 2px;
    --radius-md: 2px;
    --radius-lg: 4px;
}

/* ===== 设计变量：暗色主题（预留，接入切换后生效） ===== */
[data-theme="dark"] {
    --bg: #0a0a0a;
    --bg-subtle: #111111;
    --bg-inset: #161616;
    --border: #262626;
    --border-strong: #3a3a3a;
    --text: #ededed;
    --text-secondary: #a3a3a3;
    --text-muted: #737373;
    --accent: #ededed;
    --accent-hover: #ffffff;
    --primary: #ededed;
    --primary-hover: #ffffff;
    --primary-fg: #111111;
    --danger: #f87171;
}

/* ===== 元素基线 ===== */
*, *::before, *::after { box-sizing: border-box; }
html, body { margin: 0; padding: 0; }
body {
    background: var(--bg);
    color: var(--text);
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
    font-size: 15px;
    line-height: 1.7;
    -webkit-font-smoothing: antialiased;
}
a {
    color: var(--text);
    text-decoration: underline;
    text-decoration-color: var(--border-strong);
    text-underline-offset: 3px;
}
a:hover { text-decoration-color: var(--text); }
h1, h2, h3, h4, h5 { line-height: 1.25; font-weight: 650; letter-spacing: -0.02em; margin: 0 0 12px; }
p { margin: 0 0 12px; }
code, pre { font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace; }
::selection { background: var(--text); color: var(--bg); }

/* ===== 通用工具类 ===== */
.ven-btn {
    display: inline-flex; align-items: center; justify-content: center; gap: 6px;
    padding: 7px 16px; font-size: 13px; font-weight: 550; font-family: inherit;
    letter-spacing: 0.02em;
    border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
    background: var(--bg); color: var(--text);
    cursor: pointer; text-decoration: none;
    transition: background 0.12s, border-color 0.12s, color 0.12s;
}
.ven-btn:hover { background: var(--bg-inset); border-color: var(--text); color: var(--text); }
.ven-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.ven-btn-primary { background: var(--primary); border-color: var(--primary); color: var(--primary-fg); }
.ven-btn-primary:hover { background: var(--primary-hover); border-color: var(--primary-hover); color: var(--primary-fg); }
.ven-btn-danger { color: var(--danger); }
.ven-btn-danger:hover { color: var(--danger); border-color: var(--danger); }

.ven-input {
    width: 100%; padding: 8px 12px; font-size: 14px; font-family: inherit;
    border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
    background: var(--bg); color: var(--text);
    transition: border-color 0.12s;
}
.ven-input:focus { outline: none; border-color: var(--text); }
textarea.ven-input { resize: vertical; line-height: 1.7; }

.ven-card {
    background: var(--bg); border: 1px solid var(--border);
    border-radius: var(--radius-md); box-shadow: var(--shadow-card);
}
.ven-card-hover { transition: border-color 0.12s, background 0.12s; }
.ven-card-hover:hover { border-color: var(--text); box-shadow: none; }

.ven-chip {
    display: inline-block; font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase;
    line-height: 1.7; padding: 0 8px;
    border-radius: var(--radius-sm); background: var(--bg-subtle);
    border: 1px solid var(--border); color: var(--text-secondary);
}

/* 元信息标签：等宽 + 大写 + 宽字距（工业感的来源之一） */
.ven-meta {
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 12px; letter-spacing: 0.08em; text-transform: uppercase;
    color: var(--text-muted);
}
`;

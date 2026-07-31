/** 全局样式：暖色纸感设计系统——暖调中性色、衬线标题、玉青强调、胶片噪点、柔和微交互；
 * CSS 变量（浅色落地 + 暗色完整）、元素基线、通用工具类；经 Layout 的 <style> 注入 */

export const globalCss = `
@import url("https://fonts.googleapis.com/css2?family=Noto+Serif+SC:wght@600;700&family=JetBrains+Mono:wght@400;600&display=swap");

/* ===== 设计变量：暖纸（浅色） ===== */
:root {
    --bg: #fafaf9;
    --bg-subtle: #f5f4f2;
    --bg-inset: #efedeb;
    --border: #e7e5e4;
    --border-strong: #d6d3d1;
    --text: #1c1917;
    --text-secondary: #78716c;
    --text-muted: #a8a29e;
    --accent: #0d9488;
    --accent-hover: #14b8a6;
    --primary: #0d9488;
    --primary-hover: #14b8a6;
    --primary-fg: #fafaf9;
    --danger: #dc2626;
    --success: #22c55e;
    --ring: rgba(13, 148, 136, 0.45);
    --shadow-soft: 0 2px 8px rgba(28, 25, 23, 0.05);
    --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
    --radius-sm: 3px;
    --radius-md: 3px;
    --radius-lg: 6px;
}

/* ===== 设计变量：暗色（完整支持） ===== */
[data-theme="dark"] {
    --bg: #0c0a09;
    --bg-subtle: #1c1917;
    --bg-inset: #292524;
    --border: #292524;
    --border-strong: #44403c;
    --text: #e7e5e4;
    --text-secondary: #a8a29e;
    --text-muted: #78716c;
    --accent: #5eead4;
    --accent-hover: #2dd4bf;
    --primary: #14b8a6;
    --primary-hover: #2dd4bf;
    --primary-fg: #0c0a09;
    --danger: #f87171;
    --success: #4ade80;
    --ring: rgba(94, 234, 212, 0.45);
    --shadow-soft: 0 2px 8px rgba(0, 0, 0, 0.35);
}

/* ===== 元素基线 ===== */
*, *::before, *::after { box-sizing: border-box; }
html, body { margin: 0; padding: 0; }
body {
    background-color: var(--bg);
    background-image: linear-gradient(90deg, rgba(128, 128, 128, 0.035) 1px, transparent 1px);
    background-size: 28px 28px;
    color: var(--text);
    font-family: system-ui, -apple-system, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
    font-size: 15px;
    line-height: 1.7;
    -webkit-font-smoothing: antialiased;
}

/* 胶片噪点（soft-light 叠层；暗色 invert + 降 opacity）。z-index 999：压内容、让弹窗（1000） */
body::after {
    content: "";
    position: fixed; inset: 0; z-index: 999; pointer-events: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='240' height='240'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.8' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
    mix-blend-mode: soft-light;
    opacity: 0.2;
}
[data-theme="dark"] body::after { filter: invert(1); opacity: 0.1; }

/* 字体三分工：大标题衬线（情感），正文系统栈，元信息等宽 */
h1, h2, .ven-serif {
    font-family: Georgia, "Noto Serif SC", "Songti SC", "SimSun", serif;
    letter-spacing: -0.01em;
}
h1, h2, h3, h4, h5 { line-height: 1.25; font-weight: 650; margin: 0 0 12px; }
p { margin: 0 0 12px; }
a {
    color: var(--text);
    text-decoration: underline;
    text-decoration-color: var(--border-strong);
    text-underline-offset: 3px;
    transition: text-decoration-color 0.22s var(--ease-out), color 0.22s var(--ease-out);
}
a:hover { text-decoration-color: var(--accent); }
code, pre { font-family: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
::selection { background: rgba(13, 148, 136, 0.22); }

/* 细滚动条 4px */
::-webkit-scrollbar { width: 4px; height: 4px; }
::-webkit-scrollbar-thumb { background: var(--border-strong); border-radius: 2px; }
::-webkit-scrollbar-track { background: transparent; }

/* 焦点 */
:focus-visible { outline: 2px solid var(--ring); outline-offset: 2px; border-radius: var(--radius-sm); }

/* ===== 通用工具类 ===== */
.ven-btn {
    display: inline-flex; align-items: center; justify-content: center; gap: 6px;
    padding: 7px 16px; font-size: 13px; font-weight: 550; font-family: inherit;
    letter-spacing: 0.02em;
    border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
    background: var(--bg); color: var(--text);
    cursor: pointer; text-decoration: none;
    transition: background 0.22s var(--ease-out), border-color 0.22s var(--ease-out), color 0.22s var(--ease-out), transform 0.26s var(--ease-out);
}
.ven-btn:hover { background: var(--bg-inset); border-color: var(--text-secondary); color: var(--text); }
.ven-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.ven-btn-primary { background: var(--primary); border-color: var(--primary); color: var(--primary-fg); }
.ven-btn-primary:hover { background: var(--primary-hover); border-color: var(--primary-hover); color: var(--primary-fg); }
.ven-btn-danger { color: var(--danger); }
.ven-btn-danger:hover { color: var(--danger); border-color: var(--danger); }

.ven-input {
    width: 100%; padding: 8px 12px; font-size: 14px; font-family: inherit;
    border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
    background: var(--bg); color: var(--text);
    transition: border-color 0.22s var(--ease-out), box-shadow 0.22s var(--ease-out);
}
.ven-input:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--ring); }
textarea.ven-input { resize: vertical; line-height: 1.7; }

/* 卡片：ring 描边 + 鼠标跟随光斑 + 上浮 1px + 极淡影 */
.ven-card {
    position: relative;
    background: var(--bg); border: 1px solid var(--border);
    border-radius: var(--radius-md); box-shadow: none;
}
.ven-card::after {
    content: ""; position: absolute; inset: 0; border-radius: inherit; pointer-events: none;
    opacity: 0; transition: opacity 0.22s var(--ease-out);
    background: radial-gradient(240px circle at var(--mouse-x, 50%) var(--mouse-y, 50%), rgba(20, 184, 166, 0.14), transparent 72%);
}
.ven-card:hover::after { opacity: 1; }
.ven-card-hover { transition: transform 0.26s var(--ease-out), box-shadow 0.26s var(--ease-out), border-color 0.22s var(--ease-out); }
.ven-card-hover:hover { transform: translateY(-1px); box-shadow: var(--shadow-soft); border-color: var(--border-strong); }

.ven-chip {
    display: inline-block; font-family: "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace;
    font-size: 11px; letter-spacing: 0.06em; text-transform: uppercase;
    line-height: 1.7; padding: 0 10px;
    border-radius: 999px; background: var(--bg-subtle);
    border: 1px solid var(--border); color: var(--text-secondary);
}

/* 元信息标签：等宽 + 大写 + 宽字距 */
.ven-meta {
    font-family: "JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    font-size: 12px; letter-spacing: 0.08em; text-transform: uppercase;
    color: var(--text-muted);
}

/* 荧光笔链接：下方 30% 盖半透明强调色，hover 向上刷满 */
.ven-hl {
    text-decoration: none;
    background: linear-gradient(rgba(20, 184, 166, 0.32), rgba(20, 184, 166, 0.32)) no-repeat 0 100% / 100% 30%;
    transition: background-size 0.26s var(--ease-out), color 0.22s var(--ease-out);
}
.ven-hl:hover { background-size: 100% 100%; color: var(--text); }

/* 列表行左侧强调竖线（hover 生长） */
.ven-accent-item { position: relative; }
.ven-accent-item::before {
    content: ""; position: absolute; left: 0; top: 50%; transform: translateY(-50%);
    width: 2px; height: 0; border-radius: 1px;
    background: linear-gradient(var(--accent), var(--accent-hover));
    transition: height 0.26s var(--ease-out);
}
.ven-accent-item:hover::before { height: 24px; }

/* 区块标题下划线 scaleX 展开 */
.ven-section-title { position: relative; display: inline-block; }
.ven-section-title::after {
    content: ""; position: absolute; left: 0; bottom: -4px; height: 1px; width: 100%;
    background: var(--accent); transform: scaleX(0); transform-origin: left;
    transition: transform 0.26s var(--ease-out);
}
.ven-section-title:hover::after { transform: scaleX(1); }

/* 玩味细节：特定区块十字准星 */
.ven-crosshair { cursor: crosshair; }

/* 打字机光标 */
.ven-caret { color: var(--accent); animation: ven-blink 1s steps(1) infinite; }
@keyframes ven-blink { 50% { opacity: 0; } }

/* 辉光管时钟：玉青辉光 + 偶发微闪 */
.ven-nixie {
    font-family: "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace;
    font-size: 38px; font-weight: 600; letter-spacing: 0.08em;
    color: var(--accent);
    text-shadow:
        0 0 6px rgba(20, 184, 166, 0.55),
        0 0 18px rgba(20, 184, 166, 0.32),
        0 0 42px rgba(20, 184, 166, 0.18);
    animation: ven-nixie-flicker 5s infinite;
}
@keyframes ven-nixie-flicker {
    0%, 100% { opacity: 1; }
    91% { opacity: 1; }
    92% { opacity: 0.84; }
    93% { opacity: 1; }
    96% { opacity: 0.93; }
    97% { opacity: 1; }
}

/* /posts 左栏分类框响应式 */
@media (max-width: 720px) {
    .ven-posts-grid { grid-template-columns: 1fr !important; }
    .ven-tagbox { position: static !important; }
}

/* 主题切换纯圆按钮：svg 居中；悬浮向左展开显字（固定容器不挤动兄弟组件） */
.ven-theme-wrap { position: relative; width: 32px; height: 32px; flex-shrink: 0; }
.ven-theme-toggle {
    position: absolute; right: 0; top: 0;
    display: inline-flex; align-items: center; justify-content: center;
    width: 32px; height: 32px; padding: 0; gap: 0;
    border-radius: 999px; border: 1px solid var(--border-strong);
    background: var(--bg); color: var(--text);
    cursor: pointer; overflow: hidden; white-space: nowrap;
    transition: width 0.26s var(--ease-out), padding 0.26s var(--ease-out), background 0.22s var(--ease-out), border-color 0.22s var(--ease-out);
}
.ven-theme-toggle:hover { width: auto; padding: 0 12px; background: var(--bg-inset); }
.ven-theme-label {
    font-size: 12px; font-family: inherit;
    max-width: 0; opacity: 0; overflow: hidden;
    transition: max-width 0.26s var(--ease-out), opacity 0.22s var(--ease-out), margin-left 0.26s var(--ease-out);
}
.ven-theme-toggle:hover .ven-theme-label { max-width: 48px; opacity: 1; margin-left: 6px; }

/* 弹窗遮罩毛玻璃 */
.ven-modal-overlay {
    background: rgba(250, 250, 249, 0.68);
    backdrop-filter: blur(22px) saturate(132%);
    -webkit-backdrop-filter: blur(22px) saturate(132%);
}
[data-theme="dark"] .ven-modal-overlay {
    background: rgba(12, 10, 9, 0.6);
}

/* hero 区布局：左文案右动态 SVG（窄屏收起 SVG） */
.ven-hero { display: flex; align-items: center; justify-content: space-between; gap: 32px; }
@media (max-width: 720px) {
    .ven-hero-art { display: none; }
}

/* 顶部导航：三段式（左品牌 / 中搜索 / 右操作），窄屏搜索折行 */
.ven-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.ven-header-search { flex: 1; max-width: 380px; margin: 0 auto; }
.ven-header-search .ven-input { padding: 6px 12px; font-size: 13px; }
@media (max-width: 720px) {
    .ven-header-search { order: 3; flex-basis: 100%; max-width: none; margin: 10px 0 0; }
}

/* ===== 首页整屏板块（滚动磁吸仅首页注入，离开页面解除） ===== */
html.ven-home-snap { scroll-snap-type: y proximity; }
.ven-panel {
    min-height: calc(100vh - 96px);
    scroll-snap-align: start;
    display: flex;
    flex-direction: column;
    justify-content: center;
    position: relative;
}
.ven-panel-chevron {
    position: absolute; bottom: 20px; left: 50%; transform: translateX(-50%);
    animation: ven-bob 1.6s ease-in-out infinite;
}
@keyframes ven-bob {
    0%, 100% { transform: translate(-50%, 0); }
    50% { transform: translate(-50%, 8px); }
}
.ven-panel-nav {
    position: fixed; right: 20px; top: 50%; transform: translateY(-50%);
    display: flex; flex-direction: column; gap: 12px; z-index: 50;
}
.ven-panel-nav a {
    width: 8px; height: 8px; background: var(--border-strong); border-radius: 50%;
    transition: background 0.22s var(--ease-out), transform 0.26s var(--ease-out);
}
.ven-panel-nav a:hover { background: var(--accent); }
.ven-panel-nav a.active { background: var(--accent); transform: scale(1.35); }
@media (max-width: 900px) {
    .ven-panel-nav { display: none; }
    .ven-panel { min-height: auto; padding: 48px 0; }
    .ven-panel-chevron { display: none; }
}
`;

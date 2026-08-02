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

/* 触控兜底：≤720px 触屏设备上所有 ven-btn（含 padding 3px 的小按钮）最小触控高度 40px；
   保持原有 padding 语义，仅加触控下限，桌面端不受影响 */
@media (max-width: 720px) {
    .ven-btn { min-height: 40px; }
}

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
    user-select: none;
}
.ven-clickable { cursor: pointer; }
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

/* 评论即时上屏 slide-in */
@keyframes ven-slide-in {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
}
.ven-comment-enter { animation: ven-slide-in 0.3s var(--ease-out); }

/* 导航链接 hover 下划线滑动 */
.ven-nav-link { position: relative; }
.ven-nav-link::after {
    content: ""; position: absolute; left: 0; bottom: -3px; height: 1px; width: 100%;
    background: var(--accent); transform: scaleX(0); transform-origin: left;
    transition: transform 0.26s var(--ease-out);
}
.ven-nav-link:hover::after { transform: scaleX(1); }

/* 文章卡片封面 hover 轻微放大 */
.ven-card-hover > img { transition: transform 0.4s var(--ease-out); }
.ven-card-hover:hover > img { transform: scale(1.03); }

/* 裱框卡：外框 + 内衬垫 + SVG 四角括线（hover 括线转玉青） */
.ven-frame {
    position: relative;
    background: var(--bg); border: 1px solid var(--border);
    border-radius: 3px; padding: 12px;
    transition: border-color 0.22s var(--ease-out), transform 0.26s var(--ease-out), box-shadow 0.26s var(--ease-out);
}
.ven-frame:hover { border-color: var(--border-strong); transform: translateY(-1px); box-shadow: var(--shadow-soft); }
.ven-frame-inner {
    border: 1px solid var(--border); border-radius: 2px;
    background: var(--bg-subtle); padding: 18px 20px; height: 100%;
    display: flex; flex-direction: column;
}
.ven-frame-corners { position: absolute; inset: 5px; width: calc(100% - 10px); height: calc(100% - 10px); pointer-events: none; }
.ven-frame-corners path { stroke: var(--border-strong); transition: stroke 0.22s var(--ease-out); }
.ven-frame:hover .ven-frame-corners path { stroke: var(--accent); }

/* 打字机光标 */
.ven-caret { color: var(--accent); animation: ven-blink 1s steps(1) infinite; }
@keyframes ven-blink { 50% { opacity: 0; } }

/* 缓慢旋转（hero 作者卡头像环：外环顺时针 / 内环逆时针） */
@keyframes ven-rotate { to { transform: rotate(360deg); } }
.ven-spin-slow { animation: ven-rotate 16s linear infinite; }
.ven-spin-rev { animation: ven-rotate 26s linear infinite reverse; }

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

/* 后台侧边栏：窄屏折叠为顶栏——品牌+返回/注销一行，导航独占一行横向滚动（避免 320px 下 8 个链接多行占屏） */
@media (max-width: 900px) {
    .ven-admin-side {
        position: static !important; width: 100% !important; bottom: auto !important;
        flex-direction: row !important; align-items: center !important; flex-wrap: wrap !important;
        row-gap: 6px !important;
        border-right: none !important; border-bottom: 1px solid var(--border);
        padding: 10px 16px !important;
    }
    .ven-admin-side > a:first-child { margin-bottom: 0 !important; }
    .ven-admin-side nav {
        flex-direction: row !important; flex: 1 1 100% !important;
        flex-wrap: nowrap !important; overflow-x: auto !important;
        scrollbar-width: none !important;
    }
    .ven-admin-side nav::-webkit-scrollbar { display: none !important; }
    .ven-admin-side nav > div { margin-top: 0 !important; }
    .ven-admin-side nav a { white-space: nowrap !important; flex-shrink: 0 !important; }
    .ven-admin-side > div:last-child {
        flex-direction: row !important; border-top: none !important; padding-top: 0 !important;
        margin-left: auto !important; white-space: nowrap !important;
    }
    .ven-admin-main { margin-left: 0 !important; padding: 24px 20px 48px !important; }
}

/* 后台文章管理行：≤720px 标题独占一行，日期/统计/按钮折为副行（纵向卡片） */
@media (max-width: 720px) {
    .ven-admin-post-row { flex-wrap: wrap !important; }
    .ven-admin-post-row > :first-child { flex-basis: 100% !important; }
}

/* 后台审核/动态行：≤480px 内容独占一行，操作按钮换行到内容下方 */
@media (max-width: 480px) {
    .ven-admin-row { flex-wrap: wrap !important; }
    .ven-admin-row > div:first-child { flex-basis: 100% !important; }
}

/* /posts 左栏分类框响应式 */
@media (max-width: 720px) {
    .ven-posts-grid { grid-template-columns: 1fr !important; }
    .ven-tagbox { position: static !important; }
}

/* 移动端触控：小屏按钮与行内文字操作（点赞/回复/删除等）目标高度 ≥40px */
@media (max-width: 720px) {
    .ven-btn { min-height: 40px; }
    .ven-inline-action { min-height: 40px; }
}

/* 作者页友链双行磁吸：小屏退化为普通横滚列表（卡片横滚查看，不整页溢出） */
@media (max-width: 720px) {
    .ven-links { height: auto !important; }
    .ven-links-sticky { position: static !important; }
    .ven-links-row { overflow-x: auto !important; }
}

/* 主题切换纯圆按钮：svg 居中；悬浮向左展开显字（固定容器不挤动兄弟组件）。
 * hover 展开仅桌面（hover: hover）启用，避免触屏点按后标签卡在展开态 */
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
.ven-theme-label {
    font-size: 12px; font-family: inherit;
    max-width: 0; opacity: 0; overflow: hidden;
    transition: max-width 0.26s var(--ease-out), opacity 0.22s var(--ease-out), margin-left 0.26s var(--ease-out);
}
@media (hover: hover) {
    .ven-theme-toggle:hover { width: auto; padding: 0 12px; background: var(--bg-inset); }
    .ven-theme-toggle:hover .ven-theme-label { max-width: 48px; opacity: 1; margin-left: 6px; }
}
/* 触控小屏：主题钮放大到 40px 目标区 */
@media (max-width: 900px) {
    .ven-theme-wrap, .ven-theme-toggle { width: 40px; height: 40px; }
}

/* 弹窗遮罩毛玻璃 */
.ven-modal-overlay {
    background: rgba(250, 250, 249, 0.68);
    backdrop-filter: blur(22px) saturate(132%);
    -webkit-backdrop-filter: blur(22px) saturate(132%);
}
[data-theme="dark"] .ven-modal-overlay {
    background: rgba(12, 10, 9, 0.6);
}

/* hero 区布局：左文案右作者卡（.ven-hero-art 命中作者卡外层，窄屏改纵向排布、卡片自适应不溢出） */
.ven-hero { display: flex; align-items: center; justify-content: space-between; gap: 32px; }
.ven-hero-art { width: 330px; flex-shrink: 0; }
@media (max-width: 720px) {
    .ven-hero { flex-direction: column; align-items: stretch; gap: 28px; }
    .ven-hero-art { width: 100%; }
}

/* 仪表盘卡片行（最新文章 + Nixie 时钟）：≤480px 单列，避免时钟被挤压 */
@media (max-width: 480px) {
    .ven-dash-grid { grid-template-columns: 1fr !important; }
}

/* 顶部导航：三列 grid——左右贴边、搜索真居中不重叠；≤1100px 搜索折到第二行；
 * ≤720px 导航/账号操作收进汉堡抽屉（.ven-menu-btn） */
.ven-header { display: grid; grid-template-columns: 1fr auto 1fr; align-items: center; gap: 16px; }
.ven-header > *:first-child { justify-self: start; }
.ven-header > *:last-child { justify-self: end; }
.ven-header-search { width: min(380px, 36vw); justify-self: center; transition: width 0.26s var(--ease-out); }
.ven-header-search:focus-within { width: min(460px, 46vw); }
.ven-header-search .ven-input { padding: 6px 12px; font-size: 13px; }
@media (max-width: 1100px) {
    .ven-header { grid-template-columns: 1fr auto; }
    .ven-header-search { grid-column: 1 / -1; order: 3; width: 100%; max-width: 380px; margin: 8px auto 0; justify-self: center; }
}

/* ===== 移动端导航：汉堡按钮 + 滑出抽屉 ===== */
.ven-menu-btn {
    display: none;
    align-items: center; justify-content: center;
    width: 40px; height: 40px; padding: 0;
    border: 1px solid var(--border-strong); border-radius: 999px;
    background: var(--bg); color: var(--text);
    cursor: pointer;
    transition: background 0.22s var(--ease-out), border-color 0.22s var(--ease-out);
}
.ven-menu-btn:hover { background: var(--bg-inset); border-color: var(--text-secondary); }
@media (max-width: 720px) {
    .ven-header-nav,
    .ven-header-actions > :not(.ven-menu-btn) { display: none; }
    .ven-menu-btn { display: inline-flex; }
}

/* 抽屉遮罩：毛玻璃（与弹窗同款）；抽屉本体右侧滑入，z-index 压住导航、低于弹窗 */
@keyframes ven-fade-in { from { opacity: 0; } to { opacity: 1; } }
@keyframes ven-drawer-in { from { transform: translateX(100%); } to { transform: translateX(0); } }
.ven-drawer-overlay {
    position: fixed; inset: 0; z-index: 500;
    background: rgba(250, 250, 249, 0.68);
    backdrop-filter: blur(22px) saturate(132%);
    -webkit-backdrop-filter: blur(22px) saturate(132%);
    animation: ven-fade-in 0.24s var(--ease-out);
}
[data-theme="dark"] .ven-drawer-overlay { background: rgba(12, 10, 9, 0.6); }
.ven-drawer {
    position: fixed; top: 0; right: 0; bottom: 0; z-index: 501;
    width: min(320px, 84vw);
    display: flex; flex-direction: column;
    background: var(--bg);
    border-left: 1px solid var(--border);
    box-shadow: -12px 0 32px rgba(28, 25, 23, 0.1);
    animation: ven-drawer-in 0.28s var(--ease-out);
}
[data-theme="dark"] .ven-drawer { box-shadow: -12px 0 32px rgba(0, 0, 0, 0.45); }
.ven-drawer-head {
    display: flex; align-items: center; justify-content: space-between;
    padding: 14px 18px;
    border-bottom: 1px solid var(--border);
}
.ven-drawer-close {
    display: inline-flex; align-items: center; justify-content: center;
    width: 40px; height: 40px; padding: 0;
    border: 1px solid var(--border-strong); border-radius: 999px;
    background: var(--bg); color: var(--text);
    cursor: pointer;
    transition: background 0.22s var(--ease-out), border-color 0.22s var(--ease-out);
}
.ven-drawer-close:hover { background: var(--bg-inset); border-color: var(--text-secondary); }
.ven-drawer-body { flex: 1; overflow-y: auto; padding: 6px 18px 20px; }
.ven-drawer-nav { display: flex; flex-direction: column; border-bottom: 1px solid var(--border); }
.ven-drawer-nav-link {
    display: flex; align-items: center; gap: 12;
    padding: 13px 0; font-size: 16px; font-weight: 550;
    color: var(--text); text-decoration: none;
    border-bottom: 1px solid var(--border);
    transition: color 0.22s var(--ease-out);
}
.ven-drawer-nav-link:last-child { border-bottom: none; }
.ven-drawer-nav-link:hover { color: var(--accent); }
.ven-drawer-actions { display: flex; flex-direction: column; gap: 12; padding: 16px 0 0; }
.ven-drawer-actions .ven-btn { width: 100%; padding: 11px 16px; font-size: 14px; }
.ven-drawer-profile a {
    width: 100%; padding: 11px 0;
    justify-content: flex-start;
    border-top: 1px solid var(--border);
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

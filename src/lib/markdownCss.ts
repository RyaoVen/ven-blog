/** Markdown 正文排版样式（ven-prose）+ TOC 侧栏 + 详情页双栏布局 + hljs 配色（工业极简） */

export const markdownCss = `
/* ===== 详情页布局：主栏 + TOC 侧栏 ===== */
.ven-post-layout { display: grid; grid-template-columns: minmax(0, 1fr) 200px; gap: 48px; }
@media (max-width: 900px) {
    .ven-post-layout { grid-template-columns: 1fr; }
    .ven-toc { display: none; }
}

/* ===== TOC 侧栏 ===== */
.ven-toc { position: sticky; top: 84px; align-self: start; max-height: calc(100vh - 108px); overflow-y: auto; }
.ven-toc nav { display: flex; flex-direction: column; gap: 2px; border-left: 1px solid var(--border); }
.ven-toc a {
    display: block; padding: 3px 0 3px 12px; font-size: 13px; line-height: 1.5;
    color: var(--text-muted); text-decoration: none;
    border-left: 1px solid transparent; margin-left: -1px;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.ven-toc a:hover { color: var(--text); }
.ven-toc a.active { color: var(--accent); font-weight: 650; border-left-color: var(--accent); }

/* ===== 正文排版 ===== */
.ven-prose { font-size: 15.5px; line-height: 1.8; }
.ven-prose h1, .ven-prose h2, .ven-prose h3, .ven-prose h4, .ven-prose h5 { scroll-margin-top: 80px; margin: 36px 0 14px; }
.ven-prose h1 { font-size: 24px; padding-bottom: 8px; border-bottom: 1px solid var(--border); }
.ven-prose h2 { font-size: 20px; padding-bottom: 6px; border-bottom: 1px solid var(--border); }
.ven-prose h3 { font-size: 17px; }
.ven-prose h4 { font-size: 15.5px; }
.ven-prose h5 { font-size: 14px; letter-spacing: 0; }
.ven-prose p { margin: 0 0 16px; }
.ven-prose ul, .ven-prose ol { margin: 0 0 16px; padding-left: 26px; }
.ven-prose li { margin: 4px 0; }
.ven-prose a { color: var(--text); }
.ven-prose strong { font-weight: 700; }
.ven-prose u { text-underline-offset: 3px; }
.ven-prose hr { border: none; border-top: 1px solid var(--border); margin: 36px 0; }
.ven-prose img { max-width: 100%; border: 1px solid var(--border); border-radius: var(--radius-sm); filter: blur(18px); transition: filter 0.5s var(--ease-out); }
.ven-prose img.ven-img-loaded { filter: blur(0); }
.ven-prose blockquote {
    margin: 0 0 16px; padding: 2px 16px;
    border-left: 2px solid var(--text); color: var(--text-secondary);
}
.ven-prose blockquote p { margin: 8px 0; }

/* 表格 */
.ven-prose table { border-collapse: collapse; width: 100%; margin: 0 0 16px; font-size: 14px; }
.ven-prose th, .ven-prose td { border: 1px solid var(--border-strong); padding: 6px 12px; text-align: left; }
.ven-prose th { background: var(--bg-subtle); font-weight: 650; }

/* 行内代码 */
.ven-prose code {
    background: var(--bg-subtle); border: 1px solid var(--border);
    border-radius: var(--radius-sm); padding: 1px 5px; font-size: 0.88em;
}

/* ===== 结构化代码块（行号/行线/语言标识/复制/默认收起） ===== */
.ven-codeblock {
    border: 1px solid var(--border); border-radius: var(--radius-sm);
    background: var(--bg-subtle); margin: 0 0 16px; overflow: hidden;
}
.ven-codeblock-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 5px 12px; border-bottom: 1px solid var(--border); background: var(--bg);
}
.ven-codeblock-lang {
    font-family: "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace;
    font-size: 11px; letter-spacing: 0.08em; text-transform: uppercase; color: var(--text-muted);
}
.ven-codeblock-actions { display: flex; gap: 8px; }
.ven-codeblock-copy, .ven-codeblock-toggle {
    font-family: "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace;
    font-size: 11px; padding: 2px 10px; cursor: pointer;
    border: 1px solid var(--border-strong); border-radius: var(--radius-sm);
    background: transparent; color: var(--text-secondary);
    transition: color 0.22s var(--ease-out), border-color 0.22s var(--ease-out);
}
.ven-codeblock-copy:hover, .ven-codeblock-toggle:hover { color: var(--accent); border-color: var(--accent); }
.ven-codeblock-body { position: relative; }
.ven-codeblock-body pre {
    display: flex; margin: 0; padding: 0; border: none; background: transparent; overflow-x: auto;
}
.ven-codeblock-gutter {
    flex-shrink: 0; padding: 12px 10px; text-align: right; white-space: pre;
    font-family: "JetBrains Mono", ui-monospace, Menlo, Consolas, monospace;
    font-size: 13px; line-height: 1.65; color: var(--text-muted);
    border-right: 1px solid var(--border); user-select: none;
}
.ven-codeblock-body code {
    flex: 1; padding: 12px 14px; background: transparent; border: none;
    font-size: 13px; line-height: 1.65;
    /* 行线：每行一条发丝线 */
    background-image: linear-gradient(transparent calc(1.65em - 1px), var(--border) calc(1.65em - 1px), var(--border) 1.65em);
    background-size: 100% 1.65em;
}
.ven-codeblock[data-collapsed="true"] .ven-codeblock-body { max-height: 116px; overflow: hidden; }
.ven-codeblock-mask { display: none; }
.ven-codeblock[data-collapsed="true"] .ven-codeblock-mask {
    display: block; position: absolute; left: 0; right: 0; bottom: 0; height: 52px;
    background: linear-gradient(transparent, var(--bg-subtle));
    pointer-events: none;
}

/* admonition：:::warning / :::tip / :::note */.ven-admonition {
    border: 1px solid var(--border); border-left: 3px solid var(--text);
    border-radius: var(--radius-sm); background: var(--bg-subtle);
    padding: 12px 16px; margin: 0 0 16px;
}
.ven-admonition p { margin: 0 0 8px; }
.ven-admonition p:last-child { margin-bottom: 0; }
.ven-admonition-title {
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 12px; font-weight: 700; letter-spacing: 0.08em; text-transform: uppercase;
}
.ven-admonition-warning { border-left-color: #dc2626; }
.ven-admonition-warning .ven-admonition-title { color: #dc2626; }
.ven-admonition-tip { border-left-color: #16a34a; }
.ven-admonition-tip .ven-admonition-title { color: #16a34a; }
.ven-admonition-note { border-left-color: var(--accent); }
.ven-admonition-note .ven-admonition-title { color: var(--accent); }

/* 链接块卡片：:::link URL + 标题/简介/图标 */
.ven-linkcard {
    display: flex; align-items: center; gap: 14px;
    border: 1px solid var(--border); border-radius: var(--radius-sm);
    background: var(--bg-subtle); padding: 14px 16px; margin: 0 0 16px;
    text-decoration: none !important; color: var(--text);
    transition: border-color 0.22s var(--ease-out), transform 0.26s var(--ease-out);
}
.ven-linkcard:hover { border-color: var(--accent); transform: translateY(-1px); }
.ven-linkcard-icon {
    width: 40px; height: 40px; border-radius: 3px; object-fit: cover; flex-shrink: 0;
    border: 1px solid var(--border); background: var(--bg);
}
.ven-linkcard-icon-fallback {
    display: inline-flex; align-items: center; justify-content: center;
    color: var(--text-muted, #a8a29e);
}
.ven-linkcard-main { display: flex; flex-direction: column; gap: 3px; min-width: 0; }
.ven-linkcard-title {
    font-weight: 650; font-size: 14.5px; line-height: 1.4;
    overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.ven-linkcard-desc {
    font-size: 13px; color: var(--text-secondary); line-height: 1.5;
    display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden;
}
.ven-linkcard-host {
    font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
    font-size: 11px; color: var(--text-muted, #a8a29e); letter-spacing: 0.04em;
    transition: color 0.22s ease;
}
.ven-linkcard:hover .ven-linkcard-host { color: var(--accent); }

/* ===== hljs 配色（GitHub light 取向，克制） ===== */
.hljs-comment, .hljs-quote { color: #6e7781; font-style: italic; }
.hljs-keyword, .hljs-selector-tag, .hljs-doctag, .hljs-template-tag { color: #cf222e; }
.hljs-string, .hljs-regexp, .hljs-meta .hljs-string { color: #0a3069; }
.hljs-number, .hljs-literal { color: #0550ae; }
.hljs-title, .hljs-title.function_, .hljs-section { color: #8250df; }
.hljs-title.class_, .hljs-type, .hljs-built_in { color: #953800; }
.hljs-attr, .hljs-attribute, .hljs-variable, .hljs-template-variable { color: #1f2328; }
.hljs-name, .hljs-selector-id, .hljs-selector-class { color: #116329; }
.hljs-symbol, .hljs-bullet, .hljs-link { color: #0550ae; }
.hljs-emphasis { font-style: italic; }
.hljs-strong { font-weight: 700; }

/* ===== hljs 深色调色板（[data-theme="dark"] 覆盖，暖调） ===== */
[data-theme="dark"] .hljs-comment, [data-theme="dark"] .hljs-quote { color: #78716c; font-style: italic; }
[data-theme="dark"] .hljs-keyword, [data-theme="dark"] .hljs-selector-tag, [data-theme="dark"] .hljs-doctag, [data-theme="dark"] .hljs-template-tag { color: #f47067; }
[data-theme="dark"] .hljs-string, [data-theme="dark"] .hljs-regexp, [data-theme="dark"] .hljs-meta .hljs-string { color: #96d0ff; }
[data-theme="dark"] .hljs-number, [data-theme="dark"] .hljs-literal { color: #79c0ff; }
[data-theme="dark"] .hljs-title, [data-theme="dark"] .hljs-title.function_, [data-theme="dark"] .hljs-section { color: #d2a8ff; }
[data-theme="dark"] .hljs-title.class_, [data-theme="dark"] .hljs-type, [data-theme="dark"] .hljs-built_in { color: #ffa657; }
[data-theme="dark"] .hljs-attr, [data-theme="dark"] .hljs-attribute, [data-theme="dark"] .hljs-variable, [data-theme="dark"] .hljs-template-variable { color: #e7e5e4; }
[data-theme="dark"] .hljs-name, [data-theme="dark"] .hljs-selector-id, [data-theme="dark"] .hljs-selector-class { color: #7ee787; }
[data-theme="dark"] .hljs-symbol, [data-theme="dark"] .hljs-bullet, [data-theme="dark"] .hljs-link { color: #79c0ff; }

/* 评论内 Markdown：紧凑排版（比正文小一号、间距收紧） */
.ven-comment-prose { font-size: 14px; line-height: 1.7; }
.ven-comment-prose p { margin: 0 0 8px; }
.ven-comment-prose p:last-child { margin-bottom: 0; }
.ven-comment-prose h1, .ven-comment-prose h2, .ven-comment-prose h3,
.ven-comment-prose h4, .ven-comment-prose h5 { font-size: 14px; margin: 12px 0 6px; border: none; padding: 0; }
.ven-comment-prose pre { padding: 10px 12px; margin: 0 0 8px; }
.ven-comment-prose ul, .ven-comment-prose ol { margin: 0 0 8px; }
.ven-comment-prose blockquote { margin: 0 0 8px; }
.ven-comment-prose table { margin: 0 0 8px; font-size: 13px; }
`;

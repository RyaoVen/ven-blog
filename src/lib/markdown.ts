/**
 * Markdown 渲染管线：markdown-it + highlight.js + admonition 容器 + 自写下划线规则 + TOC 提取。
 * renderMarkdown 在 SSR 与客户端同构执行（同一输入字符串 → 同一输出），保证 hydration 一致。
 * XSS 策略：html:false（源码内嵌 HTML 一律转义），输出仅经 dangerouslySetInnerHTML 使用。
 */

import MarkdownIt from "markdown-it";
import markdownItContainer from "markdown-it-container";
import hljs from "highlight.js/lib/core";

// 只打包常用语言子集，控制 bundle 体积
import bash from "highlight.js/lib/languages/bash";
import c from "highlight.js/lib/languages/c";
import cpp from "highlight.js/lib/languages/cpp";
import css from "highlight.js/lib/languages/css";
import go from "highlight.js/lib/languages/go";
import java from "highlight.js/lib/languages/java";
import javascript from "highlight.js/lib/languages/javascript";
import json from "highlight.js/lib/languages/json";
import markdown from "highlight.js/lib/languages/markdown";
import python from "highlight.js/lib/languages/python";
import rust from "highlight.js/lib/languages/rust";
import sql from "highlight.js/lib/languages/sql";
import typescript from "highlight.js/lib/languages/typescript";
import xml from "highlight.js/lib/languages/xml";
import yaml from "highlight.js/lib/languages/yaml";

hljs.registerLanguage("bash", bash);
hljs.registerLanguage("c", c);
hljs.registerLanguage("cpp", cpp);
hljs.registerLanguage("css", css);
hljs.registerLanguage("go", go);
hljs.registerLanguage("java", java);
hljs.registerLanguage("javascript", javascript);
hljs.registerLanguage("js", javascript);
hljs.registerLanguage("json", json);
hljs.registerLanguage("markdown", markdown);
hljs.registerLanguage("python", python);
hljs.registerLanguage("rust", rust);
hljs.registerLanguage("sql", sql);
hljs.registerLanguage("typescript", typescript);
hljs.registerLanguage("ts", typescript);
hljs.registerLanguage("xml", xml);
hljs.registerLanguage("html", xml);
hljs.registerLanguage("yaml", yaml);
hljs.registerLanguage("yml", yaml);

/** 目录项（h1-h5） */
export interface TocItem {
    level: number;
    text: string;
    id: string;
}

/** 渲染结果：HTML 与目录 */
export interface RenderedMarkdown {
    html: string;
    toc: TocItem[];
}

/** HTML 转义（纯文本回退用） */
function escapeHtml(s: string): string {
    return s
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

/** 代码高亮：指定语言命中子集则高亮，否则按纯文本转义输出 */
function highlight(code: string, lang: string): string {
    const language = lang.trim().toLowerCase();
    if (language && hljs.getLanguage(language)) {
        try {
            return hljs.highlight(code, { language, ignoreIllegals: true }).value;
        } catch {
            /* 回退纯文本 */
        }
    }
    return escapeHtml(code);
}

/** ++下划线++ 内联规则（markdown-it 无内置下划线） */
function underlinePlugin(md: MarkdownIt): void {
    md.inline.ruler.after("emphasis", "underline", (state, silent) => {
        const start = state.pos;
        const PLUS = 0x2b;
        if (state.src.charCodeAt(start) !== PLUS || state.src.charCodeAt(start + 1) !== PLUS) {
            return false;
        }
        const end = state.src.indexOf("++", start + 2);
        if (end < 0 || end === start + 2) {
            return false;
        }
        if (!silent) {
            state.pos = start + 2;
            state.posMax = end;
            state.push("underline_open", "u", 1);
            state.md.inline.tokenize(state);
            state.push("underline_close", "u", -1);
        }
        state.pos = end + 2;
        state.posMax = state.src.length;
        return true;
    });
}

/** 标题锚点：h1-h5 注入 id（h-0/h-1/… 确定性序号），并把目录写入 state.env.toc */
function headingAnchorPlugin(md: MarkdownIt): void {
    md.core.ruler.push("heading_anchor", (state) => {
        const toc: TocItem[] = [];
        let index = 0;
        for (let i = 0; i < state.tokens.length; i++) {
            const token = state.tokens[i];
            if (token.type !== "heading_open") {
                continue;
            }
            const level = Number(token.tag.slice(1));
            if (level < 1 || level > 5) {
                continue;
            }
            const inline = state.tokens[i + 1];
            const text = inline?.content ?? "";
            const id = `h-${index++}`;
            token.attrSet("id", id);
            toc.push({ level, text, id });
        }
        (state.env as { toc?: TocItem[] }).toc = toc;
        return true;
    });
}

/** admonition 容器：:::warning / :::tip / :::note（首行可选自定义标题） */
const ADMONITIONS = ["warning", "tip", "note"] as const;
const ADMONITION_TITLES: Record<string, string> = {
    warning: "警告",
    tip: "提示",
    note: "注意",
};

function admonitionPlugin(md: MarkdownIt): void {
    for (const name of ADMONITIONS) {
        md.use(markdownItContainer, name, {
            validate: (params: string) => params.trim().startsWith(name),
            render: (tokens: { nesting: number; info: string }[], idx: number) => {
                if (tokens[idx].nesting === 1) {
                    const custom = tokens[idx].info.trim().slice(name.length).trim();
                    const title = custom || ADMONITION_TITLES[name];
                    return `<div class="ven-admonition ven-admonition-${name}"><p class="ven-admonition-title">${title}</p>\n`;
                }
                return "</div>\n";
            },
        });
    }
}

/** 共享渲染实例（无状态，toc 经 env 传递） */
const md = new MarkdownIt({
    html: false,
    linkify: true,
    breaks: false,
    highlight,
})
    .use(underlinePlugin)
    .use(headingAnchorPlugin)
    .use(admonitionPlugin);

/** 渲染 Markdown 源码为 HTML 并提取目录（确定性：同输入同输出） */
export function renderMarkdown(source: string): RenderedMarkdown {
    const env = {} as { toc?: TocItem[] };
    const html = md.render(source, env);
    return { html, toc: env.toc ?? [] };
}

/** Markdown 转纯文本（摘要/ excerpt 用）：渲染后剥标签、解码基础实体、空白归一 */
export function plainText(source: string): string {
    const html = md.render(source, {});
    const text = html
        .replace(/<[^>]*>/g, " ")
        .replace(/&amp;/g, "&")
        .replace(/&lt;/g, "<")
        .replace(/&gt;/g, ">")
        .replace(/&quot;/g, '"')
        .replace(/&#39;/g, "'")
        .replace(/\s+/g, " ")
        .trim();
    return text;
}

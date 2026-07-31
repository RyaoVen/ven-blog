/** 打字机轮播：逐字显示 → 停顿 8s → 逐字消失 → 下一句。
 * SSR 静态显示首句全文；点击卡片复制当前句到剪贴板并弹气泡提示。 */

import { useEffect, useState } from "react";
import { CheckIcon } from "./icons";
import { v } from "./theme";

export interface TypewriterItem {
    text: string;
    source: string;
}

export function Typewriter({
    items,
    typeMs = 70,
    eraseMs = 30,
    holdMs = 8000,
}: {
    items: TypewriterItem[];
    typeMs?: number;
    eraseMs?: number;
    holdMs?: number;
}) {
    const [index, setIndex] = useState(0);
    // len 为 null 表示 SSR/未启动（显示全文），挂载后从 0 开始打字
    const [len, setLen] = useState<number | null>(null);
    const [erasing, setErasing] = useState(false);
    const [copied, setCopied] = useState(false);

    useEffect(() => {
        if (items.length === 0) {
            return;
        }
        const full = items[index].text;
        let timer: number | undefined;
        if (len === null) {
            setLen(0);
        } else if (!erasing) {
            if (len < full.length) {
                timer = window.setTimeout(() => setLen(len + 1), typeMs);
            } else {
                timer = window.setTimeout(() => setErasing(true), holdMs);
            }
        } else {
            if (len > 0) {
                timer = window.setTimeout(() => setLen(len - 1), eraseMs);
            } else {
                setErasing(false);
                setIndex((index + 1) % items.length);
            }
        }
        return () => {
            if (timer !== undefined) {
                window.clearTimeout(timer);
            }
        };
    }, [len, erasing, index, items, typeMs, eraseMs, holdMs]);

    async function copy() {
        const current = items[index];
        const payload = `${current.text} —— ${current.source}`;
        try {
            await navigator.clipboard.writeText(payload);
        } catch {
            // 剪贴板 API 不可用（非安全上下文）时回退 execCommand
            const ta = document.createElement("textarea");
            ta.value = payload;
            document.body.appendChild(ta);
            ta.select();
            document.execCommand("copy");
            document.body.removeChild(ta);
        }
        setCopied(true);
        window.setTimeout(() => setCopied(false), 1600);
    }

    if (items.length === 0) {
        return null;
    }
    const current = items[index];
    const shown = len === null ? current.text : current.text.slice(0, len);

    return (
        <blockquote
            className="ven-card ven-clickable"
            onClick={copy}
            title="点击复制"
            style={{ margin: 0, padding: "18px 20px", borderLeft: `2px solid ${v.accent}`, minHeight: 96, position: "relative" }}
        >
            <p className="ven-serif" style={{ margin: "0 0 8px", fontSize: 16, lineHeight: 1.7 }}>
                {shown}
                <span className="ven-caret">|</span>
            </p>
            <cite className="ven-meta" style={{ fontStyle: "normal" }}>
                —— {current.source}
            </cite>
            {copied && (
                <span
                    className="ven-card"
                    style={{
                        position: "absolute",
                        top: -12,
                        right: 16,
                        display: "inline-flex",
                        alignItems: "center",
                        gap: 5,
                        padding: "4px 10px",
                        fontSize: 12,
                        color: v.accent,
                        boxShadow: "var(--shadow-soft)",
                    }}
                >
                    <CheckIcon size={12} />
                    已复制
                </span>
            )}
        </blockquote>
    );
}

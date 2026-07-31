/** 文章目录侧栏：平滑滚动 + 阅读进度环 + 折叠 + h4/h5 按当前阅读分支展开 + scroll-spy
 *
 * 点击用 scrollIntoView + replaceState（而非原生 hash 链接）：
 * 原生 hash 跳转会推 history 条目，Chrome fragment 导航触发 popstate 后
 * 框架 router 会恢复滚动位置（"跳到位置又闪回"的根因）。
 */

import { MouseEvent, useEffect, useMemo, useState } from "react";
import type { TocItem } from "../lib/markdown";

/** 进度环半径与周长 */
const RING_R = 9;
const RING_C = 2 * Math.PI * RING_R;

export function Toc({ items }: { items: TocItem[] }) {
    const [active, setActive] = useState<string | null>(null);
    const [progress, setProgress] = useState(0);
    const [collapsed, setCollapsed] = useState(false);

    // scroll-spy：视口上 30% 区域内命中的标题视为当前节
    useEffect(() => {
        const headings = items
            .map((item) => document.getElementById(item.id))
            .filter((el): el is HTMLElement => el !== null);
        if (headings.length === 0) {
            return;
        }
        const observer = new IntersectionObserver(
            (entries) => {
                for (const entry of entries) {
                    if (entry.isIntersecting) {
                        setActive(entry.target.id);
                    }
                }
            },
            { rootMargin: "0px 0px -70% 0px", threshold: 0 },
        );
        headings.forEach((h) => observer.observe(h));
        return () => observer.disconnect();
    }, [items]);

    // 阅读进度：整页滚动比例
    useEffect(() => {
        const onScroll = () => {
            const doc = document.documentElement;
            const max = doc.scrollHeight - window.innerHeight;
            setProgress(max > 0 ? Math.min(100, Math.round((window.scrollY / max) * 100)) : 100);
        };
        onScroll();
        window.addEventListener("scroll", onScroll, { passive: true });
        return () => window.removeEventListener("scroll", onScroll);
    }, []);

    // 每个标题向上最近的"更小级别"标题即为其父节
    const parents = useMemo(() => {
        const result: (number | null)[] = [];
        for (let i = 0; i < items.length; i++) {
            let parent: number | null = null;
            for (let j = i - 1; j >= 0; j--) {
                if (items[j].level < items[i].level) {
                    parent = j;
                    break;
                }
            }
            result.push(parent);
        }
        return result;
    }, [items]);

    // h4/h5 可见性：其 ≤3 级祖先在当前阅读链上（祖先即 active，或 active 是它的后代）
    const visible = useMemo(() => {
        const activeIdx = items.findIndex((item) => item.id === active);
        // active 的祖先链（含自身）
        const chain = new Set<number>();
        for (let i = activeIdx; i >= 0 && activeIdx >= 0; i = parents[i] ?? -1) {
            if (i < 0) break;
            chain.add(i);
            if (parents[i] === null) break;
        }
        return items.map((item, i) => {
            if (item.level <= 3) {
                return true;
            }
            // 找该 h4/h5 的 ≤3 级祖先
            let ancestor: number | null = i;
            while (ancestor !== null && items[ancestor].level > 3) {
                ancestor = parents[ancestor];
            }
            if (ancestor === null) {
                return true; // 没有上级标题的孤立深级项，照常显示
            }
            return chain.has(ancestor);
        });
    }, [items, parents, active]);

    function jump(event: MouseEvent, id: string) {
        event.preventDefault();
        document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" });
        history.replaceState(null, "", `#${id}`);
    }

    return (
        <aside className="ven-toc">
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
                <p className="ven-meta" style={{ margin: 0 }}>
                    目录
                </p>
                <span style={{ display: "inline-flex", alignItems: "center", gap: 5 }}>
                    <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
                        <circle cx="12" cy="12" r={RING_R} stroke="var(--border)" strokeWidth="2" fill="none" />
                        <circle
                            cx="12"
                            cy="12"
                            r={RING_R}
                            stroke="var(--text)"
                            strokeWidth="2"
                            fill="none"
                            strokeDasharray={RING_C}
                            strokeDashoffset={RING_C * (1 - progress / 100)}
                            transform="rotate(-90 12 12)"
                        />
                    </svg>
                    <span className="ven-meta" style={{ minWidth: 34 }}>
                        {progress}%
                    </span>
                </span>
                <button
                    type="button"
                    className="ven-meta"
                    onClick={() => setCollapsed((c) => !c)}
                    style={{
                        marginLeft: "auto",
                        border: "none",
                        background: "none",
                        padding: 0,
                        cursor: "pointer",
                        textTransform: "uppercase",
                    }}
                >
                    {collapsed ? "展开 +" : "折叠 −"
                    }
                </button>
            </div>
            {!collapsed && (
                <nav>
                    {items.map((item, i) =>
                        visible[i] ? (
                            <a
                                key={item.id}
                                href={`#${item.id}`}
                                onClick={(e) => jump(e, item.id)}
                                className={active === item.id ? "active" : ""}
                                style={{ paddingLeft: 12 + (item.level - 1) * 12 }}
                            >
                                {item.text}
                            </a>
                        ) : null,
                    )}
                </nav>
            )}
        </aside>
    );
}

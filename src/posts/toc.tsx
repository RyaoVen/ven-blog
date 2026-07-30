/** 文章目录侧栏：sticky 定位 + #hash 原生锚点跳转（router 不拦截）+ IntersectionObserver scroll-spy */

import { useEffect, useState } from "react";
import type { TocItem } from "../lib/markdown";

export function Toc({ items }: { items: TocItem[] }) {
    const [active, setActive] = useState<string | null>(null);

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

    return (
        <aside className="ven-toc">
            <p className="ven-meta" style={{ margin: "0 0 10px" }}>
                目录
            </p>
            <nav>
                {items.map((item) => (
                    <a
                        key={item.id}
                        href={`#${item.id}`}
                        className={active === item.id ? "active" : ""}
                        style={{ paddingLeft: 12 + (item.level - 1) * 12 }}
                    >
                        {item.text}
                    </a>
                ))}
            </nav>
        </aside>
    );
}

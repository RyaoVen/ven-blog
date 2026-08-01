/** 文章阅读进度条：详情页顶部 2px 玉青条（滚动驱动；客户端行为，SSR 静态零宽） */

import { useEffect, useRef, useState } from "react";

export function ReadingProgress({ articleRef }: { articleRef: React.RefObject<HTMLElement | null> }) {
    const [progress, setProgress] = useState(0);

    useEffect(() => {
        const onScroll = () => {
            const el = articleRef.current;
            if (!el) {
                return;
            }
            const rect = el.getBoundingClientRect();
            const total = rect.height - window.innerHeight;
            if (total <= 0) {
                setProgress(100);
                return;
            }
            const done = Math.min(Math.max(-rect.top, 0), total);
            setProgress(Math.round((done / total) * 100));
        };
        onScroll();
        window.addEventListener("scroll", onScroll, { passive: true });
        return () => window.removeEventListener("scroll", onScroll);
    }, [articleRef]);

    return (
        <div
            aria-hidden="true"
            style={{
                position: "fixed",
                top: 0,
                left: 0,
                height: 2,
                width: `${progress}%`,
                background: "var(--accent)",
                zIndex: 101,
                transition: "width 0.12s linear",
                pointerEvents: "none",
            }}
        />
    );
}

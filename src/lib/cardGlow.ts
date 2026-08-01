/** 卡片鼠标跟随光斑：document 级委托监听，给 .ven-card 设置 --mouse-x/--mouse-y（前台 Layout 与后台 AdminLayout 共用） */

import { useEffect } from "react";

export function useCardGlow() {
    useEffect(() => {
        const onMove = (e: MouseEvent) => {
            const card = (e.target as Element | null)?.closest?.(".ven-card");
            if (!card) {
                return;
            }
            const el = card as HTMLElement;
            const rect = el.getBoundingClientRect();
            el.style.setProperty("--mouse-x", `${e.clientX - rect.left}px`);
            el.style.setProperty("--mouse-y", `${e.clientY - rect.top}px`);
        };
        document.addEventListener("mousemove", onMove, { passive: true });
        return () => document.removeEventListener("mousemove", onMove);
    }, []);
}

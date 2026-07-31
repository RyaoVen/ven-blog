/** 运营时长滚动计时：N天N小时N分N秒（客户端秒级刷新；SSR 由调用方给静态兜底） */

import { useEffect, useState } from "react";

export function LiveDuration({ since }: { since: string }) {
    const [text, setText] = useState<string | null>(null);

    useEffect(() => {
        const start = new Date(since).getTime();
        if (Number.isNaN(start)) {
            return;
        }
        const tick = () => {
            const total = Math.max(0, Math.floor((Date.now() - start) / 1000));
            const d = Math.floor(total / 86400);
            const h = Math.floor((total % 86400) / 3600);
            const m = Math.floor((total % 3600) / 60);
            const sec = total % 60;
            setText(`${d}天${h}小时${m}分${sec}秒`);
        };
        tick();
        const timer = window.setInterval(tick, 1000);
        return () => window.clearInterval(timer);
    }, [since]);

    return <span>{text ?? "…"}</span>;
}

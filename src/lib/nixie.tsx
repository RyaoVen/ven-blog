/** 辉光管时钟：玉青辉光数字显示当前时间（客户端秒级刷新；SSR 占位防 hydration mismatch） */

import { useEffect, useState } from "react";

function now(): string {
    const d = new Date();
    return [d.getHours(), d.getMinutes(), d.getSeconds()].map((n) => String(n).padStart(2, "0")).join(":");
}

export function NixieClock() {
    const [time, setTime] = useState<string | null>(null);

    useEffect(() => {
        setTime(now());
        const timer = window.setInterval(() => setTime(now()), 1000);
        return () => window.clearInterval(timer);
    }, []);

    return (
        <div className="ven-card" style={{ padding: "18px 20px", textAlign: "center" }}>
            <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                NOW
            </p>
            <div className="ven-nixie">{time ?? "--:--:--"}</div>
        </div>
    );
}

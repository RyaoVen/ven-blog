/** 数字滚动：进入视口后从 0 计数到目标值（SSR 静态输出终值，客户端进入视口时播放） */

import { useEffect, useRef, useState } from "react";
import gsap from "gsap";
import { useInView } from "./reveal";

export function CountUp({ value, format }: { value: number; format?: (n: number) => string }) {
    const { ref, inView } = useInView<HTMLSpanElement>(0.6);
    const [display, setDisplay] = useState<number | null>(null); // null = SSR/未启动，显示终值

    useEffect(() => {
        if (!inView) {
            return;
        }
        const counter = { v: 0 };
        const tween = gsap.to(counter, {
            v: value,
            duration: 1.2,
            ease: "power1.out",
            onUpdate: () => setDisplay(Math.round(counter.v)),
            onComplete: () => setDisplay(null), // 结束回静态终值，避免精度残留
        });
        return () => {
            tween.kill();
        };
    }, [inView, value]);

    const shown = display ?? value;
    return <span ref={ref}>{format ? format(shown) : shown}</span>;
}

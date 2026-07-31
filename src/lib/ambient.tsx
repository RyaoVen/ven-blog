/** 环境动态 SVG 小组件：各板块点缀用（纯循环动画，SSR 静态即完整图形） */

import { useEffect, useRef } from "react";
import gsap from "gsap";

/** ECG 脉冲线（描边行进循环） */
export function PulseLine({ width = 120, height = 28 }: { width?: number; height?: number }) {
    const ref = useRef<SVGSVGElement>(null);

    useEffect(() => {
        const ctx = gsap.context(() => {
            gsap.to(".ven-pulse-path", {
                strokeDashoffset: -96,
                duration: 2.4,
                repeat: -1,
                ease: "none",
            });
        }, ref);
        return () => ctx.revert();
    }, []);

    return (
        <svg ref={ref} width={width} height={height} viewBox="0 0 120 28" fill="none" aria-hidden="true">
            <path
                className="ven-pulse-path"
                d="M0 14 H28 L36 4 L46 24 L54 9 L60 14 H78 L84 8 L92 20 L98 14 H120"
                stroke="var(--border-strong)"
                strokeWidth="1.5"
                strokeDasharray="10 14"
            />
        </svg>
    );
}

/** RSS 同心波（三弧错峰呼吸） */
export function SignalWaves({ size = 44 }: { size?: number }) {
    const ref = useRef<SVGSVGElement>(null);

    useEffect(() => {
        const ctx = gsap.context(() => {
            gsap.to(".ven-wave", {
                opacity: 0.15,
                duration: 1.5,
                repeat: -1,
                yoyo: true,
                ease: "sine.inOut",
                stagger: 0.35,
            });
        }, ref);
        return () => ctx.revert();
    }, []);

    return (
        <svg ref={ref} width={size} height={size} viewBox="0 0 48 48" fill="none" aria-hidden="true">
            <circle cx="10" cy="38" r="3" fill="var(--text)" />
            <path className="ven-wave" d="M10 24 A14 14 0 0 1 24 38" stroke="var(--border-strong)" strokeWidth="2" />
            <path className="ven-wave" d="M10 12 A26 26 0 0 1 36 38" stroke="var(--border-strong)" strokeWidth="2" />
            <path className="ven-wave" d="M10 2 A36 36 0 0 1 46 38" stroke="var(--border-strong)" strokeWidth="2" />
        </svg>
    );
}

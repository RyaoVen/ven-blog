/** hero 区动态 SVG：工业蓝图风环境循环动画（虚线方框行军、虚线圆缓转、呼吸点）。
 * 全部为循环动画——SSR 静态输出即为完整图形，水合后启动循环，无入场闪烁。 */

import { useEffect, useRef } from "react";
import gsap from "gsap";

export function HeroArt({ size = 180 }: { size?: number }) {
    const ref = useRef<SVGSVGElement>(null);

    useEffect(() => {
        const ctx = gsap.context(() => {
            // 虚线圆缓转
            gsap.to(".ven-hero-spin", {
                rotation: 360,
                duration: 40,
                repeat: -1,
                ease: "none",
                transformOrigin: "50% 50%",
            });
            // 方框虚线行军（strokeDashoffset 循环走位）
            gsap.to(".ven-hero-march", {
                strokeDashoffset: -56,
                duration: 6,
                repeat: -1,
                ease: "none",
            });
            // 呼吸点
            gsap.to(".ven-hero-dot", {
                opacity: 0.15,
                duration: 1.4,
                repeat: -1,
                yoyo: true,
                ease: "sine.inOut",
                stagger: 0.45,
            });
        }, ref);
        return () => ctx.revert();
    }, []);

    return (
        <svg
            ref={ref}
            width={size}
            height={size}
            viewBox="0 0 200 200"
            fill="none"
            style={{ color: "var(--border-strong)", flexShrink: 0 }}
            aria-hidden="true"
        >
            <rect
                x="20"
                y="20"
                width="160"
                height="160"
                stroke="currentColor"
                strokeWidth="1"
                strokeDasharray="8 6"
                className="ven-hero-march"
            />
            <circle
                cx="100"
                cy="100"
                r="56"
                stroke="currentColor"
                strokeWidth="1"
                strokeDasharray="4 6"
                className="ven-hero-spin"
            />
            <line x1="100" y1="8" x2="100" y2="28" stroke="currentColor" strokeWidth="1" />
            <line x1="100" y1="172" x2="100" y2="192" stroke="currentColor" strokeWidth="1" />
            <line x1="8" y1="100" x2="28" y2="100" stroke="currentColor" strokeWidth="1" />
            <line x1="172" y1="100" x2="192" y2="100" stroke="currentColor" strokeWidth="1" />
            <rect x="96" y="96" width="8" height="8" fill="var(--text)" className="ven-hero-dot" />
            <circle cx="100" cy="44" r="2" fill="currentColor" className="ven-hero-dot" />
            <circle cx="156" cy="100" r="2" fill="currentColor" className="ven-hero-dot" />
        </svg>
    );
}

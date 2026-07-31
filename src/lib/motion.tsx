/** 动效工具：GSAP 客户端动画。
 * 原则：SSR 输出保持静态完整（不藏内容）；动画只在水合后启动；
 * 首屏跳过入场动画（防"SSR 内容先可见→闪回起点→再播一遍"的闪烁），SPA 后续导航才播放。 */

import { ReactNode, useEffect, useRef, useState } from "react";
import gsap from "gsap";

// 首次水合完成标记（模块级，entry-client 单例生命周期）
let firstHydrated = false;

/**
 * 页面入场动画容器：SPA 导航到本页时，直接子元素依次上浮淡入（stagger）。
 * 首屏水合不播放；动画结束 clearProps 还原内联样式（不污染后续交互态）。
 */
export function PageEnter({ children }: { children: ReactNode }) {
    const ref = useRef<HTMLDivElement>(null);
    const [animate] = useState(() => {
        const should = firstHydrated;
        firstHydrated = true;
        return should;
    });

    useEffect(() => {
        if (!animate || !ref.current) {
            return;
        }
        const ctx = gsap.context(() => {
            gsap.fromTo(
                ref.current!.children,
                { opacity: 0, y: 16, filter: "blur(4px)" },
                {
                    opacity: 1,
                    y: 0,
                    filter: "blur(0px)",
                    duration: 0.5,
                    stagger: 0.07,
                    ease: "power2.out",
                    clearProps: "opacity,transform,filter",
                },
            );
        }, ref);
        return () => ctx.revert();
    }, [animate]);

    return <div ref={ref}>{children}</div>;
}

/** 滚动触发 reveal：进入视口后子元素 stagger 上浮淡入（一次性）。
 * SSR 输出完整内容；动画只在客户端进入视口时启动（首屏已在视口内的由调用方自行决定是否使用）。 */

import { ReactNode, useEffect, useRef, useState } from "react";
import gsap from "gsap";

/** 元素是否进入视口（一次性，触发后保持 true） */
export function useInView<T extends HTMLElement>(threshold = 0.2) {
    const ref = useRef<T>(null);
    const [inView, setInView] = useState(false);

    useEffect(() => {
        const el = ref.current;
        if (!el) {
            return;
        }
        const observer = new IntersectionObserver(
            (entries) => {
                for (const entry of entries) {
                    if (entry.isIntersecting) {
                        setInView(true);
                        observer.disconnect();
                    }
                }
            },
            { threshold },
        );
        observer.observe(el);
        return () => observer.disconnect();
    }, [threshold]);

    return { ref, inView };
}

/** 进入视口时对直接子元素做 stagger 上浮淡入（一次性，clearProps 还原） */
export function Reveal({ children, y = 28 }: { children: ReactNode; y?: number }) {
    const { ref, inView } = useInView<HTMLDivElement>(0.18);
    const played = useRef(false);

    useEffect(() => {
        if (!inView || played.current || !ref.current) {
            return;
        }
        played.current = true;
        const ctx = gsap.context(() => {
            gsap.fromTo(
                ref.current!.children,
                { opacity: 0, y },
                { opacity: 1, y: 0, duration: 0.7, stagger: 0.09, ease: "power2.out", clearProps: "opacity,transform" },
            );
        }, ref);
        return () => ctx.revert();
    }, [inView, y, ref]);

    return <div ref={ref}>{children}</div>;
}

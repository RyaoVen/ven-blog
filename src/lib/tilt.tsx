/** 3D 悬浮容器：鼠标跟随 tilt（GSAP quickTo 平滑回弹）。
 * 仅客户端行为（mousemove/mouseleave），SSR 静态输出不受影响。 */

import { ReactNode, useEffect, useRef } from "react";
import gsap from "gsap";

export function Tilt({ children, max = 8 }: { children: ReactNode; max?: number }) {
    const ref = useRef<HTMLDivElement>(null);

    useEffect(() => {
        const el = ref.current;
        if (!el) {
            return;
        }
        gsap.set(el, { transformPerspective: 600 });
        const rotateXTo = gsap.quickTo(el, "rotationX", { duration: 0.45, ease: "power2.out" });
        const rotateYTo = gsap.quickTo(el, "rotationY", { duration: 0.45, ease: "power2.out" });
        const liftTo = gsap.quickTo(el, "y", { duration: 0.45, ease: "power2.out" });

        const onMove = (e: MouseEvent) => {
            const rect = el.getBoundingClientRect();
            const px = (e.clientX - rect.left) / rect.width - 0.5;
            const py = (e.clientY - rect.top) / rect.height - 0.5;
            rotateYTo(px * max);
            rotateXTo(-py * max);
            liftTo(-4);
        };
        const onLeave = () => {
            rotateXTo(0);
            rotateYTo(0);
            liftTo(0);
        };
        el.addEventListener("mousemove", onMove);
        el.addEventListener("mouseleave", onLeave);
        return () => {
            el.removeEventListener("mousemove", onMove);
            el.removeEventListener("mouseleave", onLeave);
        };
    }, [max]);

    return (
        <div ref={ref} style={{ transformStyle: "preserve-3d", willChange: "transform" }}>
            {children}
        </div>
    );
}

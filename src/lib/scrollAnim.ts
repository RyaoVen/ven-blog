/** 非线性平滑滚动：GSAP ScrollToPlugin（power2.inOut），用于章节间跳转。
 * 程序滚动期间临时摘除 scroll-snap（否则 snap 引擎与补间动画争抢滚动位置造成顿挫），
 * 动画完成或中断后恢复。 */

import gsap from "gsap";
import { ScrollToPlugin } from "gsap/ScrollToPlugin";

let registered = false;
const SNAP_CLASS = "ven-home-snap";

/** 平滑滚动到指定元素（非线性缓动） */
export function scrollToElement(el: Element | null): void {
    if (!el) {
        return;
    }
    if (!registered) {
        gsap.registerPlugin(ScrollToPlugin);
        registered = true;
    }
    const root = document.documentElement;
    const hadSnap = root.classList.contains(SNAP_CLASS);
    if (hadSnap) {
        root.classList.remove(SNAP_CLASS);
    }
    const restore = () => {
        if (hadSnap) {
            root.classList.add(SNAP_CLASS);
        }
    };
    gsap.to(window, {
        scrollTo: { y: el, offsetY: 0 },
        duration: 0.9,
        ease: "power2.inOut",
        overwrite: "auto", // 连点/连续触发时旧补间让位
        onComplete: restore,
        onInterrupt: restore,
    });
}

/** 平滑滚动到同辈下一个 section */
export function scrollToNextSection(current: Element | null): void {
    scrollToElement(current?.nextElementSibling ?? null);
}

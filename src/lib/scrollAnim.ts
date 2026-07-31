/** 非线性平滑滚动：GSAP ScrollToPlugin（power2.inOut），用于章节间跳转 */

import gsap from "gsap";
import { ScrollToPlugin } from "gsap/ScrollToPlugin";

let registered = false;

/** 平滑滚动到指定元素（非线性缓动） */
export function scrollToElement(el: Element | null): void {
    if (!el) {
        return;
    }
    if (!registered) {
        gsap.registerPlugin(ScrollToPlugin);
        registered = true;
    }
    gsap.to(window, {
        scrollTo: { y: el, offsetY: 0 },
        duration: 0.9,
        ease: "power2.inOut",
    });
}

/** 平滑滚动到同辈下一个 section */
export function scrollToNextSection(current: Element | null): void {
    scrollToElement(current?.nextElementSibling ?? null);
}

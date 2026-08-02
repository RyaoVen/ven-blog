/** 固定滚动（整屏切页）：滚轮/触摸一次手势非线性滚动一整屏。
 * 仅首页使用；键盘与滚动条拖动保持原生行为（经 scroll 监听同步当前屏索引）。 */

import { RefObject, useCallback, useEffect, useRef, useState } from "react";
import { scrollToElement } from "./scrollAnim";

/** 手势触发的累积滚动量阈值与切屏锁窗时长 */
const WHEEL_THRESHOLD = 40;
const TOUCH_THRESHOLD = 50;
const LOCK_MS = 950;

export function useFixedSections(containerRef: RefObject<HTMLElement | null>, count: number) {
    const [index, setIndex] = useState(0);
    const indexRef = useRef(0);
    const lockRef = useRef(false);

    const goTo = useCallback(
        (next: number) => {
            const clamped = Math.max(0, Math.min(count - 1, next));
            if (clamped === indexRef.current || lockRef.current) {
                return;
            }
            lockRef.current = true;
            indexRef.current = clamped;
            setIndex(clamped);
            const el = containerRef.current?.querySelectorAll(".ven-panel")[clamped];
            scrollToElement(el ?? null);
            window.setTimeout(() => {
                lockRef.current = false;
            }, LOCK_MS);
        },
        [containerRef, count],
    );

    // 滚轮：累积 delta 超阈值切一屏；锁窗内阻断惯性尾
    useEffect(() => {
        // ≤900px 面板已是普通流式布局（非整屏），禁用 touch/wheel 整屏切换，交还原生滚动
        const mq = window.matchMedia("(max-width: 900px)");
        let acc = 0;
        let accTimer: number | null = null;
        const resetAcc = () => {
            acc = 0;
            accTimer = null;
        };
        const onWheel = (e: WheelEvent) => {
            if (mq.matches) {
                return;
            }
            if (lockRef.current) {
                e.preventDefault();
                return;
            }
            acc += e.deltaY;
            if (accTimer === null) {
                accTimer = window.setTimeout(resetAcc, 200);
            }
            if (Math.abs(acc) > WHEEL_THRESHOLD) {
                const next = indexRef.current + (acc > 0 ? 1 : -1);
                if (next >= 0 && next < count) {
                    e.preventDefault();
                    goTo(next);
                }
                // 边缘屏（首屏向上/末屏向下）放行原生滚动（页脚/页眉之外区域）
                resetAcc();
            }
        };

        // 触摸：纵向滑动超阈值切一屏
        let touchY = 0;
        const onTouchStart = (e: TouchEvent) => {
            touchY = e.touches[0].clientY;
        };
        const onTouchEnd = (e: TouchEvent) => {
            if (mq.matches) {
                return;
            }
            const dy = touchY - e.changedTouches[0].clientY;
            if (Math.abs(dy) > TOUCH_THRESHOLD) {
                goTo(indexRef.current + (dy > 0 ? 1 : -1));
            }
        };

        window.addEventListener("wheel", onWheel, { passive: false });
        window.addEventListener("touchstart", onTouchStart, { passive: true });
        window.addEventListener("touchend", onTouchEnd, { passive: true });
        return () => {
            window.removeEventListener("wheel", onWheel);
            window.removeEventListener("touchstart", onTouchStart);
            window.removeEventListener("touchend", onTouchEnd);
            if (accTimer !== null) {
                window.clearTimeout(accTimer);
            }
        };
    }, [goTo]);

    // 手动滚动（滚动条/键盘）时同步当前屏索引
    useEffect(() => {
        const onScroll = () => {
            if (lockRef.current) {
                return;
            }
            const panels = Array.from(containerRef.current?.querySelectorAll(".ven-panel") ?? []);
            const mid = window.scrollY + window.innerHeight / 2;
            const idx = panels.findIndex((p) => {
                const el = p as HTMLElement;
                return mid >= el.offsetTop && mid < el.offsetTop + el.offsetHeight;
            });
            if (idx >= 0 && idx !== indexRef.current) {
                indexRef.current = idx;
                setIndex(idx);
            }
        };
        window.addEventListener("scroll", onScroll, { passive: true });
        return () => window.removeEventListener("scroll", onScroll);
    }, [containerRef]);

    return { index, goTo };
}

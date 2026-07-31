/** 通用弹窗：遮罩 + ESC/点击外部关闭 + GSAP 弹入；ConfirmModal 确认对话框。
 * 仅客户端渲染（open 才挂载），SSR 输出无差异。 */

import { ReactNode, useEffect, useRef } from "react";
import gsap from "gsap";
import { XIcon } from "./icons";
import { v } from "./theme";

const overlayStyle = {
    position: "fixed",
    inset: 0,
    background: "rgba(0, 0, 0, 0.35)",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    padding: 24,
    zIndex: 1000,
} as const;

export function Modal({
    open,
    onClose,
    children,
    width = 520,
}: {
    open: boolean;
    onClose: () => void;
    children: ReactNode;
    width?: number;
}) {
    const cardRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        if (!open) {
            return;
        }
        const onKey = (e: KeyboardEvent) => {
            if (e.key === "Escape") {
                onClose();
            }
        };
        document.addEventListener("keydown", onKey);
        document.body.style.overflow = "hidden";
        if (cardRef.current) {
            gsap.fromTo(
                cardRef.current,
                { opacity: 0, y: 14, scale: 0.98 },
                { opacity: 1, y: 0, scale: 1, duration: 0.22, ease: "power2.out", clearProps: "all" },
            );
        }
        return () => {
            document.removeEventListener("keydown", onKey);
            document.body.style.overflow = "";
        };
    }, [open, onClose]);

    if (!open) {
        return null;
    }
    return (
        <div style={overlayStyle} onClick={onClose} role="dialog" aria-modal="true">
            <div
                ref={cardRef}
                className="ven-card"
                style={{ width: "100%", maxWidth: width, maxHeight: "82vh", overflowY: "auto", padding: "22px 24px" }}
                onClick={(e) => e.stopPropagation()}
            >
                <button
                    type="button"
                    onClick={onClose}
                    aria-label="关闭"
                    style={{
                        position: "absolute",
                        top: 14,
                        right: 14,
                        border: "none",
                        background: "none",
                        cursor: "pointer",
                        color: "var(--text-muted)",
                        padding: 4,
                        display: "inline-flex",
                    }}
                >
                    <XIcon size={16} />
                </button>
                {children}
            </div>
        </div>
    );
}

/** 确认对话框（替代浏览器 confirm） */
export function ConfirmModal({
    open,
    title,
    message,
    confirmText = "确认",
    danger = false,
    onCancel,
    onConfirm,
}: {
    open: boolean;
    title: string;
    message: string;
    confirmText?: string;
    danger?: boolean;
    onCancel: () => void;
    onConfirm: () => void;
}) {
    return (
        <Modal open={open} onClose={onCancel} width={400}>
            <h3 style={{ margin: "0 0 8px", fontSize: 16 }}>{title}</h3>
            <p style={{ margin: "0 0 20px", fontSize: 14, color: v.textSecondary }}>{message}</p>
            <div style={{ display: "flex", justifyContent: "flex-end", gap: 10 }}>
                <button type="button" className="ven-btn" onClick={onCancel}>
                    取消
                </button>
                <button
                    type="button"
                    className={danger ? "ven-btn ven-btn-danger" : "ven-btn ven-btn-primary"}
                    onClick={onConfirm}
                >
                    {confirmText}
                </button>
            </div>
        </Modal>
    );
}

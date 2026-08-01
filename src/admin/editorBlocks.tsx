/** 结构化表单行编辑器共享组件（设置页与个人主页编辑页共用） */

import { ReactNode } from "react";
import { v } from "../lib/theme";

/** 编辑器块：标题 + 添加按钮 + 行列表 */
export function EditorBlock({
    title,
    addLabel,
    onAdd,
    children,
}: {
    title: string;
    addLabel: string;
    onAdd: () => void;
    children: ReactNode;
}) {
    return (
        <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 10 }}>
                <span style={{ fontSize: 14, fontWeight: 650 }}>{title}</span>
                <button type="button" className="ven-btn" style={{ padding: "3px 12px", fontSize: 12 }} onClick={onAdd}>
                    {addLabel}
                </button>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
                {children.length === 0 ? (
                    <p style={{ color: v.textMuted, fontSize: 13, margin: 0 }}>暂无条目，点击右上「{addLabel}」。</p>
                ) : (
                    children
                )}
            </div>
        </div>
    );
}

/** 表单行外壳（右侧移除按钮） */
export function RowShell({ onRemove, children }: { onRemove: () => void; children: ReactNode }) {
    return (
        <div style={{ display: "flex", gap: 8, alignItems: "flex-start" }}>
            <div style={{ flex: 1, display: "flex", gap: 8, flexWrap: "wrap" }}>{children}</div>
            <button
                type="button"
                onClick={onRemove}
                className="ven-btn ven-btn-danger"
                style={{ padding: "3px 10px", fontSize: 12, flexShrink: 0 }}
                aria-label="移除"
            >
                ×
            </button>
        </div>
    );
}

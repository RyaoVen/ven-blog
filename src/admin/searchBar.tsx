/** 管理端搜索筛选小部件：搜索输入 + 可选状态下拉 */

import { ReactNode } from "react";
import { SearchIcon } from "../lib/icons";

export function SearchBar({
    keyword,
    onKeyword,
    placeholder = "搜索…",
    children,
}: {
    keyword: string;
    onKeyword: (kw: string) => void;
    placeholder?: string;
    children?: ReactNode;
}) {
    return (
        <div style={{ display: "flex", gap: "8px 10px", marginBottom: 18, alignItems: "center", flexWrap: "wrap" }}>
            <div style={{ position: "relative", flex: 1, maxWidth: 320 }}>
                <SearchIcon
                    size={13}
                    style={{ position: "absolute", left: 10, top: "50%", transform: "translateY(-50%)", color: "var(--text-muted)", pointerEvents: "none" }}
                />
                <input
                    className="ven-input"
                    style={{ paddingLeft: 30 }}
                    value={keyword}
                    onChange={(e) => onKeyword(e.target.value)}
                    placeholder={placeholder}
                />
            </div>
            {children}
        </div>
    );
}

/** 小尺寸筛选下拉 */
export function FilterSelect({
    value,
    onChange,
    options,
}: {
    value: string;
    onChange: (v: string) => void;
    options: { value: string; label: string }[];
}) {
    return (
        <select className="ven-input" style={{ width: "auto", flexShrink: 0 }} value={value} onChange={(e) => onChange(e.target.value)}>
            {options.map((o) => (
                <option key={o.value} value={o.value}>
                    {o.label}
                </option>
            ))}
        </select>
    );
}

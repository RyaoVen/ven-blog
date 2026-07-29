/** 登录态辅助：读取 JS 可读的 ven_role cookie（后端鉴权仍以 HttpOnly 的 ven_auth 为准） */

import { useEffect, useState } from "react";

/** 读取当前角色；SSR 阶段（无 document）返回 null */
export function currentRole(): string | null {
    if (typeof document === "undefined") {
        return null;
    }
    const match = document.cookie.match(/(?:^|;\s*)ven_role=([^;]*)/);
    return match ? decodeURIComponent(match[1]) : null;
}

/**
 * 组件内获取当前角色的 Hook。
 * 首渲染（含 SSR）恒为 null，挂载后读取 cookie 更新，避免 hydration mismatch。
 */
export function useRole(): string | null {
    const [role, setRole] = useState<string | null>(null);
    useEffect(() => {
        setRole(currentRole());
    }, []);
    return role;
}

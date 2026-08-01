/** viewer 身份：登录后经 /api/me 拉取当前用户（userId/role/username），游客为 null */

import { useEffect, useState } from "react";
import { useRole } from "./role";

export interface Viewer {
    userId: string;
    role: string;
    username: string;
    avatarUrl: string;
}

export function useViewer(): Viewer | null {
    const role = useRole();
    const [viewer, setViewer] = useState<Viewer | null>(null);

    useEffect(() => {
        if (role === null) {
            setViewer(null);
            return;
        }
        let cancelled = false;
        fetch("/api/me")
            .then((r) => (r.ok ? r.json() : null))
            .then((data) => {
                if (!cancelled && data) {
                    setViewer({
                        userId: data.userId,
                        role: data.role,
                        username: data.username ?? "",
                        avatarUrl: data.avatarUrl ?? "",
                    });
                }
            })
            .catch(() => {});
        return () => {
            cancelled = true;
        };
    }, [role]);

    return viewer;
}

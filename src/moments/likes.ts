/** 动态点赞状态：计数初值来自 ISR initialState，viewer 已赞列表挂载后拉取，切换后本地即时更新 */

import { useEffect, useState } from "react";
import { useViewer } from "../lib/viewer";

export function useMomentLikes(initialCounts: Record<string, number>) {
    const viewer = useViewer();
    const [counts, setCounts] = useState(initialCounts);
    const [likedIds, setLikedIds] = useState<Set<string>>(new Set());

    // ISR 再生 / SSE 推送后计数同步
    useEffect(() => setCounts(initialCounts), [initialCounts]);

    useEffect(() => {
        if (viewer === null) {
            setLikedIds(new Set());
            return;
        }
        let cancelled = false;
        fetch("/api/moments/interactions")
            .then((r) => (r.ok ? r.json() : null))
            .then((data) => {
                if (!cancelled && data) {
                    setLikedIds(new Set(data.liked ?? []));
                }
            })
            .catch(() => {});
        return () => {
            cancelled = true;
        };
    }, [viewer]);

    async function toggleLike(id: string) {
        if (viewer === null) {
            window.location.href = "/login?next=/moments";
            return;
        }
        const resp = await fetch(`/api/moments/${id}/like`, { method: "POST" });
        if (!resp.ok) {
            return;
        }
        const data = await resp.json();
        setCounts((c) => ({ ...c, [id]: data.likeCount }));
        setLikedIds((prev) => {
            const next = new Set(prev);
            if (data.liked) {
                next.add(id);
            } else {
                next.delete(id);
            }
            return next;
        });
    }

    return { counts, likedIds, toggleLike };
}

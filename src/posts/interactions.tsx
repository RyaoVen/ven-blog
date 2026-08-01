/** 文章互动条：点赞/收藏按钮 + viewer 状态管理
 * 计数初值来自 ISR initialState（公开数据）；viewer 状态挂载后经 /api 拉取（游客不拉）。
 * 切换后本地即时更新（SSE 推送随后校准）。
 */

import { useEffect, useRef, useState } from "react";
import gsap from "gsap";
import { BookmarkIcon, HeartIcon } from "../lib/icons";
import { useRole } from "../lib/role";

/** viewer 互动状态 */
export interface ViewerState {
    userId: string | null;
    liked: boolean;
    favorited: boolean;
    likeCount: number;
    favoriteCount: number;
}

/** 拉取并维护 viewer 互动状态 */
export function usePostViewer(postId: string, initialLikeCount: number, initialFavoriteCount: number) {
    const role = useRole();
    const [viewer, setViewer] = useState<ViewerState>({
        userId: null,
        liked: false,
        favorited: false,
        likeCount: initialLikeCount,
        favoriteCount: initialFavoriteCount,
    });

    // ISR 再生 / SSE 推送后计数初值变化，同步进本地状态
    useEffect(() => {
        setViewer((v) => ({ ...v, likeCount: initialLikeCount, favoriteCount: initialFavoriteCount }));
    }, [initialLikeCount, initialFavoriteCount]);

    useEffect(() => {
        if (role === null) {
            return;
        }
        let cancelled = false;
        fetch(`/api/posts/${postId}/interactions`)
            .then((r) => (r.ok ? r.json() : null))
            .then((data) => {
                if (cancelled || !data) {
                    return;
                }
                setViewer((v) => ({ ...v, userId: data.userId, liked: data.liked, favorited: data.favorited }));
            })
            .catch(() => {});
        return () => {
            cancelled = true;
        };
    }, [role, postId]);

    async function toggle(kind: "like" | "favorite") {
        if (role === null) {
            window.location.href = `/login?next=${encodeURIComponent(`/posts/${postId}`)}`;
            return;
        }
        const resp = await fetch(`/api/posts/${postId}/${kind}`, { method: "POST" });
        if (!resp.ok) {
            return;
        }
        const data = await resp.json();
        if (kind === "like") {
            setViewer((v) => ({ ...v, liked: data.liked, likeCount: data.likeCount }));
        } else {
            setViewer((v) => ({ ...v, favorited: data.favorited, favoriteCount: data.favoriteCount }));
        }
    }

    return { viewer, toggle };
}

/** 点赞/收藏按钮条 */
export function InteractionBar({
    viewer,
    onToggle,
}: {
    viewer: ViewerState;
    onToggle: (kind: "like" | "favorite") => void;
}) {
    const likeIconRef = useRef<HTMLSpanElement>(null);
    const favIconRef = useRef<HTMLSpanElement>(null);
    const prevLiked = useRef(viewer.liked);
    const prevFav = useRef(viewer.favorited);

    // 状态翻转瞬间图标弹性缩放 pop
    useEffect(() => {
        if (prevLiked.current !== viewer.liked && likeIconRef.current) {
            gsap.fromTo(likeIconRef.current, { scale: 0.5 }, { scale: 1, duration: 0.4, ease: "back.out(3)" });
        }
        prevLiked.current = viewer.liked;
    }, [viewer.liked]);
    useEffect(() => {
        if (prevFav.current !== viewer.favorited && favIconRef.current) {
            gsap.fromTo(favIconRef.current, { scale: 0.5 }, { scale: 1, duration: 0.4, ease: "back.out(3)" });
        }
        prevFav.current = viewer.favorited;
    }, [viewer.favorited]);

    return (
        <div style={{ display: "flex", gap: 12, margin: "36px 0 8px" }}>
            <button
                type="button"
                className={viewer.liked ? "ven-btn ven-btn-primary" : "ven-btn"}
                style={{ minWidth: 118, justifyContent: "center" }}
                onClick={() => onToggle("like")}
            >
                <span ref={likeIconRef} style={{ display: "inline-flex" }}>
                    <HeartIcon filled={viewer.liked} />
                </span>
                {viewer.liked ? "已点赞" : "点赞"} {viewer.likeCount}
            </button>
            <button
                type="button"
                className={viewer.favorited ? "ven-btn ven-btn-primary" : "ven-btn"}
                style={{ minWidth: 118, justifyContent: "center" }}
                onClick={() => onToggle("favorite")}
            >
                <span ref={favIconRef} style={{ display: "inline-flex" }}>
                    <BookmarkIcon filled={viewer.favorited} />
                </span>
                {viewer.favorited ? "已收藏" : "收藏"} {viewer.favoriteCount}
            </button>
        </div>
    );
}

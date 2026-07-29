/** 文章列表项渲染（首页与列表页共用） */

import { formatDateTime } from "../lib/format";
import type { Post } from "./types";

export function PostList({ posts }: { posts: Post[] }) {
    if (posts.length === 0) {
        return <p>还没有文章。</p>;
    }
    return (
        <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "flex", flexDirection: "column", gap: 16 }}>
            {posts.map((p) => (
                <li key={p.id}>
                    <a
                        href={`/posts/${p.id}`}
                        style={{ fontSize: 18, fontWeight: 600, color: "#0969da", textDecoration: "none" }}
                    >
                        {p.title}
                    </a>
                    <div style={{ fontSize: 13, color: "#57606a" }}>{formatDateTime(p.createdAt)}</div>
                </li>
            ))}
        </ul>
    );
}

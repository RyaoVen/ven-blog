/** 作者主页（公开动态页）：作者信息头 + TA 的文章列表 */

import type { PageAppProps } from "../../app/pageApp";
import { Layout } from "../../lib/layout";
import { formatDateTime } from "../../lib/format";
import { v } from "../../lib/theme";
import { PostList } from "../../posts/list";
import { LetterAvatar } from "../../profiles/avatar";
import type { AuthorProfileState } from "../../profiles/types";

export default function AuthorProfilePage({ bootstrap }: PageAppProps) {
    const state = bootstrap.initialState as AuthorProfileState | null;
    const author = state?.author;
    if (!state || !author) {
        return (
            <Layout>
                <p style={{ color: v.textSecondary }}>作者不存在。</p>
            </Layout>
        );
    }
    return (
        <Layout>
            <header
                style={{
                    display: "flex",
                    gap: 24,
                    alignItems: "flex-start",
                    paddingBottom: 28,
                    marginBottom: 32,
                    borderBottom: `1px solid ${v.border}`,
                }}
            >
                <LetterAvatar name={author.username} />
                <div style={{ flex: 1, minWidth: 0 }}>
                    <p className="ven-meta" style={{ margin: 0 }}>
                        AUTHOR
                    </p>
                    <h1 style={{ fontSize: 28, margin: "8px 0 0" }}>{author.username}</h1>
                    {author.bio && <p style={{ color: v.textSecondary, margin: "10px 0 0" }}>{author.bio}</p>}
                    <p className="ven-meta" style={{ margin: "12px 0 0" }}>
                        注册于 {formatDateTime(author.createdAt)}
                    </p>
                </div>
            </header>
            <section>
                <h2 style={{ fontSize: 20, marginBottom: 20 }}>TA 的文章</h2>
                <PostList posts={state.posts} />
            </section>
        </Layout>
    );
}

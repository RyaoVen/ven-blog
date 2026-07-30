/** 用户个人页（公开动态页）：头像/用户名/角色/简介/注册时间 + 文章·评论统计；作者附作者主页入口 */

import type { PageAppProps } from "../../app/pageApp";
import { Layout } from "../../lib/layout";
import { formatDateTime } from "../../lib/format";
import { v } from "../../lib/theme";
import { LetterAvatar } from "../../profiles/avatar";
import type { UserProfileState } from "../../profiles/types";

export default function UserProfilePage({ bootstrap }: PageAppProps) {
    const state = bootstrap.initialState as UserProfileState | null;
    const user = state?.user;
    if (!state || !user) {
        return (
            <Layout>
                <p style={{ color: v.textSecondary }}>用户不存在。</p>
            </Layout>
        );
    }
    return (
        <Layout>
            <section style={{ display: "flex", gap: 24, alignItems: "flex-start" }}>
                <LetterAvatar name={user.username} />
                <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
                        <h1 style={{ fontSize: 28, margin: 0 }}>{user.username}</h1>
                        <span className="ven-chip">{user.role}</span>
                    </div>
                    {user.bio && <p style={{ color: v.textSecondary, margin: "10px 0 0" }}>{user.bio}</p>}
                    <p className="ven-meta" style={{ margin: "12px 0 0" }}>
                        注册于 {formatDateTime(user.createdAt)}
                    </p>
                    <div
                        style={{
                            display: "flex",
                            gap: 32,
                            marginTop: 20,
                            paddingTop: 16,
                            borderTop: `1px solid ${v.border}`,
                        }}
                    >
                        <span>
                            <strong style={{ fontSize: 20, fontWeight: 650 }}>{state.stats.posts}</strong>{" "}
                            <span className="ven-meta">文章</span>
                        </span>
                        <span>
                            <strong style={{ fontSize: 20, fontWeight: 650 }}>{state.stats.comments}</strong>{" "}
                            <span className="ven-meta">评论</span>
                        </span>
                    </div>
                    {state.isAuthor && (
                        <a
                            href={`/author/${user.username}`}
                            className="ven-meta"
                            style={{ display: "inline-block", marginTop: 18, textDecoration: "none", color: v.text }}
                        >
                            查看作者主页 →
                        </a>
                    )}
                </div>
            </section>
        </Layout>
    );
}

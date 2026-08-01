/** 用户个人页（公开动态页）：头像/用户名/角色/简介/注册时间 + 文章·评论统计；作者附作者主页入口；本人可见收藏列表 */

import { FormEvent, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { CheckIcon } from "../../lib/icons";
import { Layout } from "../../lib/layout";
import { formatDateTime } from "../../lib/format";
import { v } from "../../lib/theme";
import { LetterAvatar } from "../../profiles/avatar";
import { PostList } from "../../posts/list";
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
                <LetterAvatar name={user.username} avatarUrl={user.avatarUrl} />
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
            {state.email !== undefined && <EmailSection initial={state.email} />}
            {state.favorites && state.favorites.length > 0 && (
                <section style={{ marginTop: 48 }}>
                    <h2 style={{ fontSize: 20, marginBottom: 16 }}>我的收藏</h2>
                    <PostList posts={state.favorites} />
                </section>
            )}
        </Layout>
    );
}

/** 绑定/修改邮箱（仅本人可见） */
function EmailSection({ initial }: { initial: string }) {
    const [email, setEmail] = useState(initial);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [saved, setSaved] = useState(false);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        setSaved(false);
        try {
            const resp = await fetch("/api/me/email", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(
                    data?.error === "email taken"
                        ? "该邮箱已被其他账号绑定"
                        : data?.error === "invalid email"
                          ? "邮箱格式不正确"
                          : "保存失败",
                );
                return;
            }
            setSaved(true);
            window.setTimeout(() => setSaved(false), 2000);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={{ padding: "18px 20px", marginTop: 32, maxWidth: 460 }}>
            <h2 style={{ fontSize: 17, margin: "0 0 6px" }}>绑定邮箱</h2>
            <p style={{ fontSize: 13, color: v.textSecondary, margin: "0 0 14px" }}>
                用于邮箱验证码登录与 @ 邮件通知（仅本人可见）。
            </p>
            <form onSubmit={onSubmit} style={{ display: "flex", gap: 10, alignItems: "center" }}>
                <input className="ven-input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" required />
                <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting} style={{ flexShrink: 0 }}>
                    {submitting ? "保存中…" : "保存"}
                </button>
            </form>
            {error && <p style={{ color: v.danger, fontSize: 13, margin: "8px 0 0" }}>{error}</p>}
            {saved && (
                <p style={{ display: "flex", alignItems: "center", gap: 5, color: v.accent, fontSize: 13, margin: "8px 0 0" }}>
                    <CheckIcon size={13} />
                    已保存
                </p>
            )}
        </section>
    );
}

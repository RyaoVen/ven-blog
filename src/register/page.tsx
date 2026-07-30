/** 注册页：表单提交 /auth/register（注册即登录），成功跳转 next 查询参数或 /posts */

import { FormEvent, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";

export default function RegisterPage({ bootstrap }: PageAppProps) {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [confirm, setConfirm] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        if (password !== confirm) {
            setError("两次输入的密码不一致");
            return;
        }
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/auth/register", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username, password }),
            });
            if (!resp.ok) {
                const data = await resp.json().catch(() => null);
                const messages: Record<string, string> = {
                    "username taken": "该用户名已被占用",
                    "username must be 2-32 characters": "用户名需 2-32 个字符",
                    "password must be at least 6 characters": "密码至少 6 位",
                };
                setError(messages[data?.error] ?? "注册失败，请重试");
                return;
            }
            navigate(bootstrap.query.next || "/posts");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <Layout>
            <div className="ven-card" style={{ maxWidth: 400, margin: "48px auto", padding: "32px 28px" }}>
                <p className="ven-meta" style={{ margin: "0 0 6px" }}>
                    SIGN UP
                </p>
                <h1 style={{ fontSize: 22, marginBottom: 4 }}>注册</h1>
                <p style={{ fontSize: 14, color: v.textSecondary, marginBottom: 24 }}>
                    注册即可评论、点赞、收藏
                </p>
                <form style={{ display: "flex", flexDirection: "column", gap: 16 }} onSubmit={onSubmit}>
                    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                        用户名
                        <input
                            className="ven-input"
                            value={username}
                            onChange={(e) => setUsername(e.target.value)}
                            autoComplete="username"
                            required
                        />
                    </label>
                    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                        密码
                        <input
                            className="ven-input"
                            type="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            autoComplete="new-password"
                            required
                        />
                    </label>
                    <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                        确认密码
                        <input
                            className="ven-input"
                            type="password"
                            value={confirm}
                            onChange={(e) => setConfirm(e.target.value)}
                            autoComplete="new-password"
                            required
                        />
                    </label>
                    {error && <p style={{ color: v.danger, fontSize: 14, margin: 0 }}>{error}</p>}
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "注册中…" : "注册"}
                    </button>
                    <p style={{ fontSize: 13, color: v.textSecondary, margin: 0 }}>
                        已有账号？<a href="/login">去登录</a>
                    </p>
                </form>
            </div>
        </Layout>
    );
}

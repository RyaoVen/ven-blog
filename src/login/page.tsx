/** 登录页：表单提交 /auth/login，成功跳转 next 查询参数或 /posts */

import { FormEvent, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";

export default function LoginPage({ bootstrap }: PageAppProps) {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/auth/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username, password }),
            });
            if (!resp.ok) {
                const data = await resp.json().catch(() => null);
                setError(data?.error === "invalid credentials" ? "用户名或密码错误" : "登录失败");
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
                <h1 style={{ fontSize: 22, marginBottom: 4 }}>登录</h1>
                <p style={{ fontSize: 14, color: v.textSecondary, marginBottom: 24 }}>登录后可评论、点赞、收藏</p>
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
                            autoComplete="current-password"
                            required
                        />
                    </label>
                    {error && <p style={{ color: v.danger, fontSize: 14, margin: 0 }}>{error}</p>}
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "登录中…" : "登录"}
                    </button>
                </form>
            </div>
        </Layout>
    );
}

/** 登录页：表单提交 /auth/login，成功跳转 next 查询参数或 /posts */

import { FormEvent, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";

const styles = {
    form: { maxWidth: 360, display: "flex", flexDirection: "column", gap: 12 },
    label: { display: "flex", flexDirection: "column", gap: 4, fontSize: 14 },
    input: {
        padding: "6px 10px",
        fontSize: 14,
        border: "1px solid #d0d7de",
        borderRadius: 6,
    },
    submit: {
        padding: "8px 0",
        fontSize: 14,
        color: "#fff",
        background: "#1f883d",
        border: "none",
        borderRadius: 6,
        cursor: "pointer",
    },
    error: { color: "#cf222e", fontSize: 14, margin: 0 },
} as const;

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
            <h1>登录</h1>
            <form style={styles.form} onSubmit={onSubmit}>
                <label style={styles.label}>
                    用户名
                    <input
                        style={styles.input}
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        autoComplete="username"
                        required
                    />
                </label>
                <label style={styles.label}>
                    密码
                    <input
                        style={styles.input}
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        autoComplete="current-password"
                        required
                    />
                </label>
                {error && <p style={styles.error}>{error}</p>}
                <button style={styles.submit} type="submit" disabled={submitting}>
                    {submitting ? "登录中…" : "登录"}
                </button>
            </form>
        </Layout>
    );
}

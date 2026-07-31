/** 认证表单（登录/注册）：/login /register 页面与导航认证弹窗共用。
 * onSuccess 缺省时按 SPA 跳转到 nextUrl（或 /posts）；弹窗场景传 reload 闭包。 */

import { FormEvent, useState } from "react";
import { navigate } from "../app/router";
import { v } from "./theme";

interface AuthFormProps {
    /** 登录/注册成功后的跳转目标（SPA 场景） */
    nextUrl?: string;
    /** 自定义成功回调（弹窗场景：reload 应用会话） */
    onSuccess?: () => void;
}

const formStyle = { display: "flex", flexDirection: "column", gap: 14 } as const;
const labelStyle = { display: "flex", flexDirection: "column", gap: 6, fontSize: 14 } as const;

function useSubmit(done: (nextUrl?: string) => void, nextUrl?: string) {
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);

    async function run(action: () => Promise<Response>) {
        setSubmitting(true);
        setError(null);
        try {
            const resp = await action();
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                return data?.error as string;
            }
            done(nextUrl);
            return null;
        } catch {
            return "网络错误，请重试";
        } finally {
            setSubmitting(false);
        }
    }

    return { error, submitting, setError, run };
}

export function LoginForm({ nextUrl, onSuccess }: AuthFormProps) {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const { error, submitting, setError, run } = useSubmit((url) => (onSuccess ? onSuccess() : navigate(url || "/posts")), nextUrl);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        const err = await run(() =>
            fetch("/auth/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username, password }),
            }),
        );
        if (err) {
            setError(err === "invalid credentials" ? "用户名或密码错误" : "登录失败");
        }
    }

    return (
        <form style={formStyle} onSubmit={onSubmit}>
            <label style={labelStyle}>
                用户名
                <input
                    className="ven-input"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    autoComplete="username"
                    required
                />
            </label>
            <label style={labelStyle}>
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
    );
}

export function RegisterForm({ nextUrl, onSuccess }: AuthFormProps) {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const [confirm, setConfirm] = useState("");
    const { error, submitting, setError, run } = useSubmit((url) => (onSuccess ? onSuccess() : navigate(url || "/posts")), nextUrl);

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        if (password !== confirm) {
            setError("两次输入的密码不一致");
            return;
        }
        const err = await run(() =>
            fetch("/auth/register", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username, password }),
            }),
        );
        if (err) {
            const messages: Record<string, string> = {
                "username taken": "该用户名已被占用",
                "username must be 2-32 characters": "用户名需 2-32 个字符",
                "password must be at least 6 characters": "密码至少 6 位",
            };
            setError(messages[err] ?? "注册失败，请重试");
        }
    }

    return (
        <form style={formStyle} onSubmit={onSubmit}>
            <label style={labelStyle}>
                用户名
                <input
                    className="ven-input"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    autoComplete="username"
                    required
                />
            </label>
            <label style={labelStyle}>
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
            <label style={labelStyle}>
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
        </form>
    );
}

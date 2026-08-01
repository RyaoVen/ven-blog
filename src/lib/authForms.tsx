/** 认证表单（登录/注册）：/login /register 页面与导航认证弹窗共用。
 * 登录支持密码与邮箱验证码两种方式（tab 切换）；注册邮箱必填（验证码登录依赖）。
 * onSuccess 缺省时按 SPA 跳转到 nextUrl（或 /posts）；弹窗场景传 reload 闭包。 */

import { FormEvent, useEffect, useState } from "react";
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
const tabBarStyle = { display: "flex", gap: 16, marginBottom: 18, borderBottom: `1px solid ${v.border}` } as const;

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

/** 登录 tab 切换按钮 */
function Tab({ active, label, onClick }: { active: boolean; label: string; onClick: () => void }) {
    return (
        <button
            type="button"
            onClick={onClick}
            style={{
                border: "none",
                background: "none",
                padding: "0 0 8px",
                cursor: "pointer",
                fontSize: 14,
                color: active ? v.accent : v.textSecondary,
                fontWeight: active ? 650 : 400,
                borderBottom: active ? `2px solid ${v.accent}` : "2px solid transparent",
            }}
        >
            {label}
        </button>
    );
}

export function LoginForm({ nextUrl, onSuccess }: AuthFormProps) {
    const [tab, setTab] = useState<"password" | "email">("password");
    const done = (url?: string) => (onSuccess ? onSuccess() : navigate(url || "/posts"));
    return (
        <div>
            <div style={tabBarStyle}>
                <Tab active={tab === "password"} label="密码登录" onClick={() => setTab("password")} />
                <Tab active={tab === "email"} label="邮箱验证码登录" onClick={() => setTab("email")} />
            </div>
            {tab === "password" ? (
                <PasswordLoginForm nextUrl={nextUrl} onSuccess={onSuccess} done={done} />
            ) : (
                <EmailCodeLoginForm nextUrl={nextUrl} done={done} />
            )}
        </div>
    );
}

function PasswordLoginForm({ nextUrl, done }: AuthFormProps & { done: (url?: string) => void }) {
    const [username, setUsername] = useState("");
    const [password, setPassword] = useState("");
    const { error, submitting, setError, run } = useSubmit(done, nextUrl);

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
                <input className="ven-input" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" required />
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

/** 邮箱验证码登录（发送倒计时 60s） */
function EmailCodeLoginForm({ nextUrl, done }: { nextUrl?: string; done: (url?: string) => void }) {
    const [email, setEmail] = useState("");
    const [code, setCode] = useState("");
    const [countdown, setCountdown] = useState(0);
    const [sendMsg, setSendMsg] = useState<string | null>(null);
    const { error, submitting, setError, run } = useSubmit(done, nextUrl);

    useEffect(() => {
        if (countdown <= 0) {
            return;
        }
        const timer = window.setTimeout(() => setCountdown((c) => c - 1), 1000);
        return () => window.clearTimeout(timer);
    }, [countdown]);

    async function sendCode() {
        setSendMsg(null);
        setError(null);
        const resp = await fetch("/auth/email/code", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ email }),
        });
        if (resp.ok) {
            setCountdown(60);
            setSendMsg("验证码已发送（10 分钟内有效）");
        } else {
            const data = await resp.json().catch(() => null);
            setError(data?.error === "invalid email" ? "邮箱格式不正确" : "发送失败，请重试");
        }
    }

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        const err = await run(() =>
            fetch("/auth/email/login", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ email, code }),
            }),
        );
        if (err) {
            setError("验证码错误或已过期");
        }
    }

    return (
        <form style={formStyle} onSubmit={onSubmit}>
            <label style={labelStyle}>
                邮箱
                <div style={{ display: "flex", gap: 8 }}>
                    <input
                        className="ven-input"
                        type="email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        placeholder="you@example.com"
                        required
                    />
                    <button className="ven-btn" type="button" style={{ flexShrink: 0 }} disabled={countdown > 0 || !email} onClick={sendCode}>
                        {countdown > 0 ? `${countdown}s` : "发送验证码"}
                    </button>
                </div>
            </label>
            <label style={labelStyle}>
                验证码
                <input
                    className="ven-input"
                    value={code}
                    onChange={(e) => setCode(e.target.value)}
                    placeholder="6 位数字"
                    maxLength={6}
                    inputMode="numeric"
                    required
                />
            </label>
            {sendMsg && <p style={{ color: v.accent, fontSize: 13, margin: 0 }}>{sendMsg}</p>}
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
    const [email, setEmail] = useState("");
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
                body: JSON.stringify({ username, password, email }),
            }),
        );
        if (err) {
            const messages: Record<string, string> = {
                "username taken": "该用户名已被占用",
                "email taken": "该邮箱已被其他账号绑定",
                "invalid email": "邮箱格式不正确",
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
                <input className="ven-input" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" required />
            </label>
            <label style={labelStyle}>
                邮箱（用于验证码登录与 @ 邮件通知）
                <input className="ven-input" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" required />
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

/** 后台-设置页：账号安全 / 作者资料 / 内容配置 / 文章分类 / 评论审核 */

import { ChangeEvent, FormEvent, ReactNode, useRef, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { CheckIcon } from "../../lib/icons";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import type { AdminSettingsState, SettingsContent } from "../settingsTypes";

const sectionStyle = { padding: "22px 24px", marginBottom: 24 } as const;

function SectionTitle({ children }: { children: string }) {
    return (
        <h2 style={{ fontSize: 17, margin: "0 0 16px" }}>
            {children}
        </h2>
    );
}

/** 通用保存成功提示（短暂显示） */
function useToast() {
    const [toast, setToast] = useState<string | null>(null);
    const show = (text: string) => {
        setToast(text);
        window.setTimeout(() => setToast(null), 2000);
    };
    const node = toast ? (
        <span style={{ display: "inline-flex", alignItems: "center", gap: 5, fontSize: 13, color: v.accent }}>
            <CheckIcon size={13} />
            {toast}
        </span>
    ) : null;
    return { show, node };
}

export default function AdminSettingsPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? {
        content: { paragraphs: [], skills: [], friends: [], quotes: [], projects: [], github: "" },
        moderation: false,
        categories: [],
        profile: { username: "", bio: "", avatarUrl: "" },
        email: { host: "", port: "", user: "", fromName: "", passwordSet: false, authorEmail: "" },
    }) as AdminSettingsState;
    return (
        <AdminLayout route={bootstrap.route}>
            <PasswordSection />
            <ProfileSection profile={state.profile} />
            <EmailSection config={state.email} />
            <ContentSection content={state.content} />
            <CategoriesSection categories={state.categories} />
            <ModerationSection initial={state.moderation} />
        </AdminLayout>
    );
}

/* ===== 邮箱配置 ===== */
function EmailSection({ config }: { config: AdminSettingsState["email"] }) {
    const [host, setHost] = useState(config.host);
    const [port, setPort] = useState(config.port);
    const [user, setUser] = useState(config.user);
    const [password, setPassword] = useState("");
    const [fromName, setFromName] = useState(config.fromName);
    const [authorEmail, setAuthorEmail] = useState(config.authorEmail);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/settings/email", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ host, port, user, password, fromName, authorEmail }),
            });
            if (!resp.ok) {
                const data = await resp.json().catch(() => null);
                setError(data?.error === "invalid email" ? "作者邮箱格式不正确" : (data?.error ?? "保存失败"));
                return;
            }
            show("邮箱配置已保存（立即可用）");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>邮箱配置</SectionTitle>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                <p className="ven-meta" style={{ margin: 0 }}>
                    网站发件邮箱（SMTP 代理，用于发送验证码与 @ 通知）
                </p>
                <div style={{ display: "grid", gridTemplateColumns: "2fr 1fr", gap: 10 }}>
                    <input className="ven-input" placeholder="SMTP 主机（如 smtp.qq.com）" value={host} onChange={(e) => setHost(e.target.value)} />
                    <input className="ven-input" placeholder="端口（465/587）" value={port} onChange={(e) => setPort(e.target.value)} />
                </div>
                <input className="ven-input" placeholder="SMTP 账号（即发件地址）" value={user} onChange={(e) => setUser(e.target.value)} />
                <input
                    className="ven-input"
                    type="password"
                    placeholder={config.passwordSet ? "密码/授权码（已设置，留空保持不变）" : "密码/授权码"}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    autoComplete="new-password"
                />
                <input className="ven-input" placeholder="发件人名称（可选，如 ven-blog）" value={fromName} onChange={(e) => setFromName(e.target.value)} />
                <p className="ven-meta" style={{ margin: "8px 0 0" }}>
                    作者个人邮箱（接收 @ 通知，也可用于验证码登录；会同步绑定到 author 账号）
                </p>
                <input className="ven-input" type="email" placeholder="you@example.com" value={authorEmail} onChange={(e) => setAuthorEmail(e.target.value)} />
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : "保存邮箱配置"}
                    </button>
                    {node}
                </div>
            </form>
        </section>
    );
}

/* ===== 账号安全 ===== */
function PasswordSection() {
    const [oldPassword, setOldPassword] = useState("");
    const [newPassword, setNewPassword] = useState("");
    const [confirm, setConfirm] = useState("");
    const [error, setError] = useState<string | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        if (newPassword !== confirm) {
            setError("两次输入的新密码不一致");
            return;
        }
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/settings/password", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ oldPassword, newPassword }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(data?.error === "old password incorrect" ? "旧密码不正确" : (data?.error ?? "修改失败"));
                return;
            }
            setOldPassword("");
            setNewPassword("");
            setConfirm("");
            show("密码已更新");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>账号安全</SectionTitle>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 12, maxWidth: 360 }}>
                <input className="ven-input" type="password" placeholder="旧密码" value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} autoComplete="current-password" required />
                <input className="ven-input" type="password" placeholder="新密码（至少 6 位）" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} autoComplete="new-password" required />
                <input className="ven-input" type="password" placeholder="确认新密码" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" required />
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "提交中…" : "修改密码"}
                    </button>
                    {node}
                </div>
            </form>
        </section>
    );
}

/* ===== 作者资料 ===== */
function ProfileSection({ profile }: { profile: AdminSettingsState["profile"] }) {
    const [bio, setBio] = useState(profile.bio);
    const [avatarUrl, setAvatarUrl] = useState(profile.avatarUrl);
    const [uploading, setUploading] = useState(false);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const fileRef = useRef<HTMLInputElement>(null);
    const { show, node } = useToast();

    async function onPickAvatar(event: ChangeEvent<HTMLInputElement>) {
        const file = event.target.files?.[0];
        event.target.value = "";
        if (!file) {
            return;
        }
        setUploading(true);
        setError(null);
        try {
            const form = new FormData();
            form.append("file", file);
            const resp = await fetch("/api/upload", { method: "POST", body: form });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(data?.error ?? "头像上传失败");
                return;
            }
            setAvatarUrl(data.url);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setUploading(false);
        }
    }

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/settings/profile", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ bio, avatarUrl }),
            });
            if (!resp.ok) {
                const data = await resp.json().catch(() => null);
                setError(data?.error ?? "保存失败");
                return;
            }
            show("资料已保存（首页/作者主页稍后自动刷新）");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>作者资料</SectionTitle>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 14 }}>
                <div style={{ display: "flex", alignItems: "center", gap: 16 }}>
                    {avatarUrl ? (
                        <img src={avatarUrl} alt="头像" style={{ width: 56, height: 56, borderRadius: 3, objectFit: "cover", border: `1px solid ${v.border}` }} />
                    ) : (
                        <span
                            style={{
                                width: 56,
                                height: 56,
                                borderRadius: 3,
                                background: v.text,
                                display: "inline-flex",
                                alignItems: "center",
                                justifyContent: "center",
                                fontSize: 22,
                                fontWeight: 700,
                                color: v.bg,
                            }}
                        >
                            {profile.username.slice(0, 1).toUpperCase()}
                        </span>
                    )}
                    <div>
                        <button className="ven-btn" type="button" disabled={uploading} onClick={() => fileRef.current?.click()}>
                            {uploading ? "上传中…" : "更换头像"}
                        </button>
                        <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/webp,image/gif" style={{ display: "none" }} onChange={onPickAvatar} />
                        <p className="ven-meta" style={{ margin: "8px 0 0" }}>
                            jpg/png/webp/gif，5MB 内
                        </p>
                    </div>
                </div>
                <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                    个人简介（bio，展示在 hero 作者卡与个人页）
                    <textarea className="ven-input" rows={3} value={bio} onChange={(e) => setBio(e.target.value)} maxLength={200} placeholder="ven_hybird 框架作者，本站站长。" />
                </label>
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : "保存资料"}
                    </button>
                    {node}
                </div>
            </form>
        </section>
    );
}

/* ===== 内容配置（结构化表单行） ===== */
function ContentSection({ content }: { content: SettingsContent }) {
    const [paragraphs, setParagraphs] = useState<string[]>(content.paragraphs);
    const [skills, setSkills] = useState(content.skills);
    const [friends, setFriends] = useState(content.friends);
    const [quotes, setQuotes] = useState(content.quotes);
    const [projects, setProjects] = useState(content.projects);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const payload: SettingsContent = {
                paragraphs: paragraphs.map((p) => p.trim()).filter(Boolean),
                skills: skills.filter((s) => s.name.trim()),
                friends: friends.filter((f) => f.name.trim()),
                quotes: quotes.filter((q) => q.text.trim()),
                projects: projects.filter((p) => p.name.trim()),
                github: content.github,
            };
            const resp = await fetch("/api/admin/settings/content", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            });
            if (!resp.ok) {
                setError("保存失败");
                return;
            }
            show("内容配置已保存（相关页面稍后自动刷新）");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>内容配置</SectionTitle>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 22 }}>
                <EditorBlock title="个人介绍段落" addLabel="添加段落" onAdd={() => setParagraphs((l) => [...l, ""])}>
                    {paragraphs.map((p, i) => (
                        <RowShell key={i} onRemove={() => setParagraphs((l) => l.filter((_, x) => x !== i))}>
                            <textarea
                                className="ven-input"
                                rows={2}
                                value={p}
                                onChange={(e) => setParagraphs((l) => l.map((x, xi) => (xi === i ? e.target.value : x)))}
                                placeholder={`第 ${i + 1} 段`}
                            />
                        </RowShell>
                    ))}
                </EditorBlock>
                <EditorBlock title="技术栈" addLabel="添加技能" onAdd={() => setSkills((l) => [...l, { name: "", level: "know" }])}>
                    {skills.map((s, i) => (
                        <RowShell key={i} onRemove={() => setSkills((l) => l.filter((_, x) => x !== i))}>
                            <input
                                className="ven-input"
                                style={{ flex: 1, minWidth: 140 }}
                                value={s.name}
                                onChange={(e) => setSkills((l) => l.map((x, xi) => (xi === i ? { ...x, name: e.target.value } : x)))}
                                placeholder="技能名"
                            />
                            <select
                                className="ven-input"
                                style={{ width: 110, flexShrink: 0 }}
                                value={s.level}
                                onChange={(e) => setSkills((l) => l.map((x, xi) => (xi === i ? { ...x, level: e.target.value } : x)))}
                            >
                                <option value="deep">深入</option>
                                <option value="solid">熟练</option>
                                <option value="know">了解</option>
                            </select>
                        </RowShell>
                    ))}
                </EditorBlock>
                <EditorBlock title="友链" addLabel="添加友链" onAdd={() => setFriends((l) => [...l, { name: "", url: "", desc: "" }])}>
                    {friends.map((f, i) => (
                        <RowShell key={i} onRemove={() => setFriends((l) => l.filter((_, x) => x !== i))}>
                            <input className="ven-input" style={{ width: 120 }} value={f.name} onChange={(e) => setFriends((l) => l.map((x, xi) => (xi === i ? { ...x, name: e.target.value } : x)))} placeholder="名称" />
                            <input className="ven-input" style={{ flex: 1, minWidth: 160 }} value={f.url} onChange={(e) => setFriends((l) => l.map((x, xi) => (xi === i ? { ...x, url: e.target.value } : x)))} placeholder="https://…" />
                            <input className="ven-input" style={{ width: 140 }} value={f.desc} onChange={(e) => setFriends((l) => l.map((x, xi) => (xi === i ? { ...x, desc: e.target.value } : x)))} placeholder="描述" />
                        </RowShell>
                    ))}
                </EditorBlock>
                <EditorBlock title="收藏的句子" addLabel="添加句子" onAdd={() => setQuotes((l) => [...l, { text: "", source: "" }])}>
                    {quotes.map((q, i) => (
                        <RowShell key={i} onRemove={() => setQuotes((l) => l.filter((_, x) => x !== i))}>
                            <input className="ven-input" style={{ flex: 1, minWidth: 200 }} value={q.text} onChange={(e) => setQuotes((l) => l.map((x, xi) => (xi === i ? { ...x, text: e.target.value } : x)))} placeholder="句子" />
                            <input className="ven-input" style={{ width: 160 }} value={q.source} onChange={(e) => setQuotes((l) => l.map((x, xi) => (xi === i ? { ...x, source: e.target.value } : x)))} placeholder="出处" />
                        </RowShell>
                    ))}
                </EditorBlock>
                <EditorBlock title="维护的项目" addLabel="添加项目" onAdd={() => setProjects((l) => [...l, { name: "", desc: "", url: "" }])}>
                    {projects.map((p, i) => (
                        <RowShell key={i} onRemove={() => setProjects((l) => l.filter((_, x) => x !== i))}>
                            <input className="ven-input" style={{ width: 140 }} value={p.name} onChange={(e) => setProjects((l) => l.map((x, xi) => (xi === i ? { ...x, name: e.target.value } : x)))} placeholder="项目名" />
                            <input className="ven-input" style={{ flex: 1, minWidth: 160 }} value={p.desc} onChange={(e) => setProjects((l) => l.map((x, xi) => (xi === i ? { ...x, desc: e.target.value } : x)))} placeholder="描述" />
                            <input className="ven-input" style={{ width: 160 }} value={p.url} onChange={(e) => setProjects((l) => l.map((x, xi) => (xi === i ? { ...x, url: e.target.value } : x)))} placeholder="https://…" />
                        </RowShell>
                    ))}
                </EditorBlock>
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : "保存配置"}
                    </button>
                    {node}
                </div>
            </form>
        </section>
    );
}

/** 编辑器块：标题 + 添加按钮 + 行列表 */
function EditorBlock({
    title,
    addLabel,
    onAdd,
    children,
}: {
    title: string;
    addLabel: string;
    onAdd: () => void;
    children: ReactNode;
}) {
    return (
        <div>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", marginBottom: 10 }}>
                <span style={{ fontSize: 14, fontWeight: 650 }}>{title}</span>
                <button type="button" className="ven-btn" style={{ padding: "3px 12px", fontSize: 12 }} onClick={onAdd}>
                    {addLabel}
                </button>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
                {children.length === 0 ? (
                    <p style={{ color: v.textMuted, fontSize: 13, margin: 0 }}>暂无条目，点击右上「{addLabel}」。</p>
                ) : (
                    children
                )}
            </div>
        </div>
    );
}

/** 表单行外壳（右侧移除按钮） */
function RowShell({ onRemove, children }: { onRemove: () => void; children: React.ReactNode }) {
    return (
        <div style={{ display: "flex", gap: 8, alignItems: "flex-start" }}>
            <div style={{ flex: 1, display: "flex", gap: 8, flexWrap: "wrap" }}>{children}</div>
            <button
                type="button"
                onClick={onRemove}
                className="ven-btn ven-btn-danger"
                style={{ padding: "3px 10px", fontSize: 12, flexShrink: 0 }}
                aria-label="移除"
            >
                ×
            </button>
        </div>
    );
}

/* ===== 文章分类（结构化表单行） ===== */
function CategoriesSection({ categories }: { categories: string[] }) {
    const [items, setItems] = useState<string[]>(categories);
    const [submitting, setSubmitting] = useState(false);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        try {
            const resp = await fetch("/api/admin/settings/categories", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ categories: items.map((x) => x.trim()).filter(Boolean) }),
            });
            if (resp.ok) {
                show("分类已保存");
            }
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>文章分类（编辑器必选其一）</SectionTitle>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                <EditorBlock title="" addLabel="添加分类" onAdd={() => setItems((l) => [...l, ""])}>
                    {items.map((c, i) => (
                        <RowShell key={i} onRemove={() => setItems((l) => l.filter((_, x) => x !== i))}>
                            <input
                                className="ven-input"
                                style={{ maxWidth: 240 }}
                                value={c}
                                onChange={(e) => setItems((l) => l.map((x, xi) => (xi === i ? e.target.value : x)))}
                                placeholder="分类名"
                            />
                        </RowShell>
                    ))}
                </EditorBlock>
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : "保存分类"}
                    </button>
                    {node}
                </div>
            </form>
        </section>
    );
}

/* ===== 评论审核 ===== */
function ModerationSection({ initial }: { initial: boolean }) {
    const [on, setOn] = useState(initial);
    const [saving, setSaving] = useState(false);
    const { show, node } = useToast();

    async function toggle(next: boolean) {
        setOn(next);
        setSaving(true);
        try {
            await fetch("/api/admin/settings/moderation", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ on: next }),
            });
            show(next ? "已开启评论审核" : "已关闭评论审核");
        } finally {
            setSaving(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>评论审核</SectionTitle>
            <label style={{ display: "flex", alignItems: "center", gap: 10, fontSize: 14, cursor: "pointer" }}>
                <input type="checkbox" checked={on} disabled={saving} onChange={(e) => toggle(e.target.checked)} style={{ width: 16, height: 16, accentColor: "#0d9488" }} />
                开启评论审核（开启后，所有新评论需经你人工审核通过才会公开显示）
            </label>
            <div style={{ marginTop: 10 }}>{node}</div>
        </section>
    );
}

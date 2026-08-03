/** 后台-设置页：账号安全 / 作者资料 / 邮箱 / AI 审核（LLM）/ 内容配置 / 文章分类 / 评论功能 / 评论审核 / API 访问密钥 */

import { ChangeEvent, FormEvent, useEffect, useRef, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { CheckIcon } from "../../lib/icons";
import { ConfirmModal, Modal } from "../../lib/modal";
import { EditorBlock, RowShell, rowShellCss } from "../editorBlocks";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import { formatDateTime } from "../../lib/format";
import type { AdminSettingsState, ApiKeyView, SettingsContent } from "../settingsTypes";

const sectionStyle = { padding: "22px 24px", marginBottom: 24 } as const;

/** 页面级窄屏样式：双列输入（2fr 1fr）≤480px 收敛单列；RowShell 固定宽输入 ≤480px 满宽 */
const settingsCss = rowShellCss + `
.ven-grid-pair { display: grid; grid-template-columns: 2fr 1fr; gap: 10; }
@media (max-width: 480px) {
    .ven-grid-pair { grid-template-columns: 1fr; }
}
`;

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
        commentsEnabled: true,
        aiModeration: false,
        categories: [],
        profile: { username: "", bio: "", avatarUrl: "" },
        email: { host: "", port: "", user: "", fromName: "", passwordSet: false, authorEmail: "" },
        llm: { baseUrl: "", model: "", keySet: false },
        siteIcon: "",
    }) as AdminSettingsState;
    return (
        <AdminLayout route={bootstrap.route}>
            <style>{settingsCss}</style>
            <SiteSection siteIcon={state.siteIcon} username={state.profile.username} />
            <PasswordSection />
            <ProfileSection profile={state.profile} />
            <EmailSection config={state.email} />
            <LLMSection config={state.llm} aiOn={state.aiModeration} />
            <ContentSection content={state.content} />
            <CategoriesSection categories={state.categories} />
            <CommentsToggleSection initial={state.commentsEnabled} />
            <ModerationSection initial={state.moderation} />
            <KeysSection />
        </AdminLayout>
    );
}

/* ===== AI 审核（LLM 配置 + 开关） ===== */
function LLMSection({ config, aiOn }: { config: AdminSettingsState["llm"]; aiOn: boolean }) {
    const [baseUrl, setBaseUrl] = useState(config.baseUrl);
    const [model, setModel] = useState(config.model);
    const [apiKey, setApiKey] = useState("");
    const [on, setOn] = useState(aiOn);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function toggle(next: boolean) {
        setOn(next);
        try {
            await fetch("/api/admin/settings/ai-moderation", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ on: next }),
            });
            show(next ? "已开启 AI 自动审核" : "已关闭 AI 自动审核");
        } catch {
            /* 静默：开关状态本地已切 */
        }
    }

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/settings/llm", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ baseUrl: baseUrl.trim(), apiKey: apiKey.trim(), model: model.trim() }),
            });
            if (!resp.ok) {
                const data = await resp.json().catch(() => null);
                setError(data?.error === "invalid base url" ? "端点 URL 格式不正确" : (data?.error ?? "保存失败"));
                return;
            }
            setApiKey("");
            show("LLM 配置已保存（下一轮审核起生效）");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>AI 自动审核（LLM）</SectionTitle>
            <label style={{ display: "flex", alignItems: "center", gap: 10, fontSize: 14, cursor: "pointer", marginBottom: 16 }}>
                <input type="checkbox" checked={on} onChange={(e) => toggle(e.target.checked)} style={{ width: 16, height: 16, accentColor: "#0d9488" }} />
                开启 AI 自动审核（新评论/留言先经 LLM 判定：明显违规自动驳回、正常自动放行、不确定转人工）
            </label>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                <div className="ven-grid-pair">
                    <input
                        className="ven-input"
                        placeholder="OpenAI 兼容端点（默认 https://api.deepseek.com/v1）"
                        value={baseUrl}
                        onChange={(e) => setBaseUrl(e.target.value)}
                    />
                    <input
                        className="ven-input"
                        placeholder="模型（默认 deepseek-chat）"
                        value={model}
                        onChange={(e) => setModel(e.target.value)}
                    />
                </div>
                <input
                    className="ven-input"
                    type="password"
                    placeholder={config.keySet ? "API Key（已设置，留空保持不变）" : "API Key（必填，审核 worker 依赖）"}
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    autoComplete="new-password"
                />
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : "保存 LLM 配置"}
                    </button>
                    {node}
                    {!config.keySet && !apiKey && (
                        <span className="ven-meta" style={{ color: v.danger }}>
                            未配置 API Key，AI 审核不会运行
                        </span>
                    )}
                </div>
            </form>
        </section>
    );
}

/* ===== 站点与用户名 ===== */
function SiteSection({ siteIcon, username }: { siteIcon: string; username: string }) {
    const [icon, setIcon] = useState(siteIcon);
    const [name, setName] = useState(username);
    const [uploading, setUploading] = useState(false);
    const [savingIcon, setSavingIcon] = useState(false);
    const [savingName, setSavingName] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const fileRef = useRef<HTMLInputElement>(null);
    const { show, node } = useToast();

    async function onPickIcon(event: ChangeEvent<HTMLInputElement>) {
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
                setError(data?.error ?? "图标上传失败");
                return;
            }
            setIcon(data.url);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setUploading(false);
        }
    }

    async function saveIcon(next: string) {
        setSavingIcon(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/settings/site", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ icon: next }),
            });
            if (!resp.ok) {
                setError("保存失败");
                return;
            }
            setIcon(next);
            show("站点图标已保存（刷新页面后生效）");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSavingIcon(false);
        }
    }

    async function saveUsername(event: FormEvent) {
        event.preventDefault();
        if (name.trim() === username) {
            return;
        }
        setSavingName(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/settings/username", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ username: name.trim() }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(
                    data?.error === "username taken"
                        ? "该用户名已被占用"
                        : data?.error === "username must be 2-32 characters"
                          ? "用户名需 2-32 个字符"
                          : "修改失败",
                );
                return;
            }
            show(`用户名已改为「${data?.username}」（作者主页地址已同步变更）`);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSavingName(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>站点与用户名</SectionTitle>
            <div style={{ display: "flex", alignItems: "center", gap: 16, marginBottom: 18 }}>
                {icon ? (
                    <img src={icon} alt="站点图标" style={{ width: 44, height: 44, borderRadius: 3, objectFit: "cover", border: `1px solid ${v.border}` }} />
                ) : (
                    <span
                        style={{
                            width: 44,
                            height: 44,
                            borderRadius: 3,
                            background: v.text,
                            display: "inline-flex",
                            alignItems: "center",
                            justifyContent: "center",
                            fontSize: 18,
                            fontWeight: 700,
                            color: v.bg,
                        }}
                    >
                        V
                    </span>
                )}
                <div>
                    <div style={{ display: "flex", gap: 8 }}>
                        <button className="ven-btn" type="button" disabled={uploading || savingIcon} onClick={() => fileRef.current?.click()}>
                            {uploading ? "上传中…" : "选择图标"}
                        </button>
                        {icon && (
                            <button className="ven-btn" type="button" disabled={savingIcon} onClick={() => saveIcon("")}>
                                移除
                            </button>
                        )}
                    </div>
                    <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/webp,image/gif" style={{ display: "none" }} onChange={onPickIcon} />
                    <p className="ven-meta" style={{ margin: "8px 0 0" }}>
                        站点图标（favicon 与导航品牌标），方形最佳
                    </p>
                </div>
                {icon !== siteIcon && (
                    <button className="ven-btn ven-btn-primary" type="button" disabled={savingIcon} onClick={() => saveIcon(icon)}>
                        {savingIcon ? "保存中…" : "保存图标"}
                    </button>
                )}
            </div>
            <form onSubmit={saveUsername} style={{ display: "flex", flexDirection: "column", gap: 10, maxWidth: 360 }}>
                <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
                    作者用户名（登录账号与作者主页地址 /author/&lt;用户名&gt; 随之变更）
                    <input className="ven-input" value={name} onChange={(e) => setName(e.target.value)} minLength={2} maxLength={32} required />
                </label>
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={savingName || name.trim() === username}>
                        {savingName ? "提交中…" : "修改用户名"}
                    </button>
                    {node}
                </div>
            </form>
            {error && <p style={{ color: v.danger, fontSize: 13, margin: "10px 0 0" }}>{error}</p>}
        </section>
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
                <div className="ven-grid-pair">
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

/* ===== 内容配置（首页句子；项目与展示柜移至 /admin/author 编辑） ===== */
function ContentSection({ content }: { content: SettingsContent }) {
    const [quotes, setQuotes] = useState(content.quotes);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/settings/content", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    quotes: quotes.filter((q) => q.text.trim()),
                }),
            });
            if (!resp.ok) {
                setError("保存失败");
                return;
            }
            show("内容配置已保存（首页稍后自动刷新）");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>内容配置（首页句子）</SectionTitle>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 22 }}>
                <EditorBlock title="收藏的句子" addLabel="添加句子" onAdd={() => setQuotes((l) => [...l, { text: "", source: "" }])}>
                    {quotes.map((q, i) => (
                        <RowShell key={i} onRemove={() => setQuotes((l) => l.filter((_, x) => x !== i))}>
                            <input className="ven-input" style={{ flex: 1, minWidth: 200 }} value={q.text} onChange={(e) => setQuotes((l) => l.map((x, xi) => (xi === i ? { ...x, text: e.target.value } : x)))} placeholder="句子" />
                            <input className="ven-input ven-fw" style={{ width: 160 }} value={q.source} onChange={(e) => setQuotes((l) => l.map((x, xi) => (xi === i ? { ...x, source: e.target.value } : x)))} placeholder="出处" />
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

/* ===== 文章分类（管理器：新增/改名/删除迁移） ===== */
function CategoriesSection({ categories }: { categories: string[] }) {
    const [items, setItems] = useState<string[]>(categories);
    const [newName, setNewName] = useState("");
    const [editing, setEditing] = useState<{ old: string; next: string } | null>(null);
    const [deleting, setDeleting] = useState<{ name: string; count: number } | null>(null);
    const [migrateTo, setMigrateTo] = useState("");
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function add() {
        const name = newName.trim();
        if (!name) {
            return;
        }
        setError(null);
        const resp = await fetch("/api/admin/categories", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name }),
        });
        if (resp.ok) {
            setItems((l) => [...l, name]);
            setNewName("");
            show(`已添加「${name}」`);
        } else {
            const data = await resp.json().catch(() => null);
            setError(data?.error === "category exists" ? "分类已存在" : "添加失败");
        }
    }

    async function rename() {
        if (!editing) {
            return;
        }
        const next = editing.next.trim();
        if (!next || next === editing.old) {
            setEditing(null);
            return;
        }
        setError(null);
        const resp = await fetch(`/api/admin/categories/${encodeURIComponent(editing.old)}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ name: next }),
        });
        if (resp.ok) {
            setItems((l) => l.map((x) => (x === editing.old ? next : x)));
            show(`已改名为「${next}」（该分类文章已同步迁移）`);
        } else {
            const data = await resp.json().catch(() => null);
            setError(data?.error === "category exists" ? "目标名已存在" : (data?.error ?? "改名失败"));
        }
        setEditing(null);
    }

    async function remove(name: string) {
        setError(null);
        const resp = await fetch(`/api/admin/categories/${encodeURIComponent(name)}`, { method: "DELETE" });
        if (resp.status === 409) {
            const data = await resp.json().catch(() => null);
            setDeleting({ name, count: data?.count ?? 0 });
            setMigrateTo(items.find((x) => x !== name) ?? "");
            return;
        }
        if (resp.ok) {
            setItems((l) => l.filter((x) => x !== name));
            show(`已删除「${name}」`);
        }
    }

    async function confirmMigrate() {
        if (!deleting) {
            return;
        }
        const resp = await fetch(
            `/api/admin/categories/${encodeURIComponent(deleting.name)}?migrateTo=${encodeURIComponent(migrateTo)}`,
            { method: "DELETE" },
        );
        if (resp.ok) {
            const data = await resp.json().catch(() => null);
            setItems((l) => l.filter((x) => x !== deleting.name));
            show(`已迁移 ${data?.migrated ?? 0} 篇到「${migrateTo}」并删除「${deleting.name}」`);
            setDeleting(null);
        } else {
            const data = await resp.json().catch(() => null);
            setError(data?.error ?? "迁移删除失败");
            setDeleting(null);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>文章分类</SectionTitle>
            <div style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 16 }}>
                <input
                    className="ven-input"
                    style={{ maxWidth: 240, flex: 1 }}
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && (e.preventDefault(), add())}
                    placeholder="新分类名（回车添加）"
                />
                <button className="ven-btn ven-btn-primary" type="button" onClick={add}>
                    添加
                </button>
            </div>
            {items.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14 }}>暂无分类。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {items.map((c) => (
                        <li key={c} style={{ display: "flex", gap: 8, alignItems: "center", flexWrap: "wrap", padding: "8px 0", borderBottom: `1px solid ${v.border}` }}>
                            {editing?.old === c ? (
                                <>
                                    <input
                                        className="ven-input"
                                        style={{ maxWidth: 240, flex: 1 }}
                                        value={editing.next}
                                        onChange={(e) => setEditing({ old: c, next: e.target.value })}
                                        onKeyDown={(e) => {
                                            if (e.key === "Enter") {
                                                e.preventDefault();
                                                rename();
                                            }
                                            if (e.key === "Escape") {
                                                setEditing(null);
                                            }
                                        }}
                                        autoFocus
                                    />
                                    <button className="ven-btn ven-btn-primary" type="button" style={{ padding: "3px 12px", fontSize: 12 }} onClick={rename}>
                                        保存
                                    </button>
                                    <button className="ven-btn" type="button" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => setEditing(null)}>
                                        取消
                                    </button>
                                </>
                            ) : (
                                <>
                                    <span style={{ fontSize: 14, fontWeight: 550 }}>{c}</span>
                                    <span style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
                                        <button className="ven-btn" type="button" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => setEditing({ old: c, next: c })}>
                                            改名
                                        </button>
                                        <button className="ven-btn ven-btn-danger" type="button" style={{ padding: "3px 12px", fontSize: 12 }} onClick={() => remove(c)}>
                                            删除
                                        </button>
                                    </span>
                                </>
                            )}
                        </li>
                    ))}
                </ul>
            )}
            {error && <p style={{ color: v.danger, fontSize: 13, margin: "10px 0 0" }}>{error}</p>}
            <div style={{ marginTop: 10 }}>{node}</div>
            <Modal open={deleting !== null} onClose={() => setDeleting(null)} width={420}>
                <h3 style={{ margin: "0 0 10px", fontSize: 16 }}>删除分类「{deleting?.name}」</h3>
                <p style={{ fontSize: 14, color: v.textSecondary, margin: "0 0 14px" }}>
                    该分类下还有 <strong>{deleting?.count}</strong> 篇文章。删除前请选择一个目标分类，文章将一键迁移过去。
                </p>
                <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14, marginBottom: 16 }}>
                    迁移到
                    <select className="ven-input" value={migrateTo} onChange={(e) => setMigrateTo(e.target.value)}>
                        {items.filter((x) => x !== deleting?.name).map((x) => (
                            <option key={x} value={x}>
                                {x}
                            </option>
                        ))}
                    </select>
                </label>
                <div style={{ display: "flex", justifyContent: "flex-end", gap: 10 }}>
                    <button className="ven-btn" type="button" onClick={() => setDeleting(null)}>
                        取消
                    </button>
                    <button className="ven-btn ven-btn-primary" type="button" onClick={confirmMigrate} disabled={!migrateTo}>
                        迁移并删除
                    </button>
                </div>
            </Modal>
        </section>
    );
}

/* ===== 评论功能（总开关：一键关闭全站评论区） ===== */
function CommentsToggleSection({ initial }: { initial: boolean }) {
    const [on, setOn] = useState(initial);
    const [saving, setSaving] = useState(false);
    const { show, node } = useToast();

    async function toggle(next: boolean) {
        setOn(next);
        setSaving(true);
        try {
            const resp = await fetch("/api/admin/settings/comments-enabled", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ on: next }),
            });
            if (!resp.ok) {
                setOn(!next); // 保存失败回滚开关
                return;
            }
            show(next ? "评论功能已开启（文章页/动态页评论区恢复）" : "评论功能已关闭（全站评论区隐藏，新评论被拒绝）");
        } catch {
            setOn(!next);
        } finally {
            setSaving(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>评论功能</SectionTitle>
            <label style={{ display: "flex", alignItems: "center", gap: 10, fontSize: 14, cursor: "pointer" }}>
                <input type="checkbox" checked={on} disabled={saving} onChange={(e) => toggle(e.target.checked)} style={{ width: 16, height: 16, accentColor: "#0d9488" }} />
                开启评论功能（关闭后全站评论区隐藏、新评论提交被拒绝；后台审核队列与历史评论保留，重新开启即恢复）
            </label>
            <div style={{ marginTop: 10 }}>{node}</div>
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
            show(next ? "已开启评论与留言审核" : "已关闭评论与留言审核");
        } finally {
            setSaving(false);
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>评论与留言审核</SectionTitle>
            <label style={{ display: "flex", alignItems: "center", gap: 10, fontSize: 14, cursor: "pointer" }}>
                <input type="checkbox" checked={on} disabled={saving} onChange={(e) => toggle(e.target.checked)} style={{ width: 16, height: 16, accentColor: "#0d9488" }} />
                开启评论与留言审核（开启后，新评论与新留言先进入待审队列——配置 AI 审核则由 LLM 先判，不确定与失败的转人工）
            </label>
            <div style={{ marginTop: 10 }}>{node}</div>
        </section>
    );
}

/* ===== API 访问密钥 =====
 * 数据客户端现取（不进 SSR initialState）：密钥动态（last_used_at 随鉴权变化、可随时吊销）且含高敏操作。
 * 明文仅创建弹窗展示一次，关窗即丢弃；服务端只存哈希。 */
const mono = { fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace" } as const;

function KeysSection() {
    const [keys, setKeys] = useState<ApiKeyView[]>([]);
    const [name, setName] = useState("");
    const [creating, setCreating] = useState(false);
    const [created, setCreated] = useState<{ raw: string; view: ApiKeyView } | null>(null);
    const [copied, setCopied] = useState(false);
    const [revoking, setRevoking] = useState<ApiKeyView | null>(null);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function load() {
        try {
            const resp = await fetch("/api/admin/keys");
            const data = await resp.json().catch(() => null);
            if (resp.ok) {
                setKeys(data?.keys ?? []);
            }
        } catch {
            // 拉取失败保留旧数据（列表非关键路径，不打断页面）
        }
    }

    useEffect(() => {
        void load();
    }, []);

    async function create(event: FormEvent) {
        event.preventDefault();
        if (!name.trim()) {
            return;
        }
        setCreating(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/keys", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ name: name.trim() }),
            });
            const data = await resp.json().catch(() => null);
            if (!resp.ok) {
                setError(
                    data?.error === "name is required"
                        ? "请填写用途备注"
                        : data?.error === "name too long (max 64)"
                          ? "备注最长 64 字符"
                          : (data?.error ?? "生成失败"),
                );
                return;
            }
            setCreated({ raw: data.key, view: data.view });
            setName("");
            setCopied(false);
        } catch {
            setError("网络错误，请重试");
        } finally {
            setCreating(false);
        }
    }

    async function copyRaw() {
        if (!created) {
            return;
        }
        try {
            await navigator.clipboard.writeText(created.raw);
            setCopied(true);
        } catch {
            // 剪贴板不可用时忽略（仍可手动选中复制）
        }
    }

    /** 关窗即丢弃明文，重新拉列表 */
    function closeCreated() {
        setCreated(null);
        void load();
    }

    async function revoke() {
        if (!revoking) {
            return;
        }
        const target = revoking;
        setRevoking(null);
        setError(null);
        try {
            const resp = await fetch(`/api/admin/keys/${encodeURIComponent(target.id)}`, { method: "DELETE" });
            if (resp.ok) {
                show(`已吊销「${target.name}」（即时生效）`);
            } else {
                const data = await resp.json().catch(() => null);
                setError(data?.error === "api key not found" ? "密钥不存在或已吊销" : (data?.error ?? "吊销失败"));
            }
            void load();
        } catch {
            setError("网络错误，请重试");
        }
    }

    return (
        <section className="ven-card" style={sectionStyle}>
            <SectionTitle>API 访问密钥</SectionTitle>
            <p className="ven-meta" style={{ margin: "0 0 14px" }}>
                程序化调用（agent / 脚本）使用的凭据，请求头携带 <code style={mono}>Authorization: Bearer ven_xxx</code>。
                服务端只保存哈希，明文仅在生成时展示一次；吊销立即生效。
            </p>
            <form onSubmit={create} style={{ display: "flex", gap: 8, flexWrap: "wrap", marginBottom: 16 }}>
                <input
                    className="ven-input"
                    style={{ maxWidth: 240, flex: 1 }}
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="用途备注（如 zcode-agent）"
                    maxLength={64}
                />
                <button className="ven-btn ven-btn-primary" type="submit" disabled={creating}>
                    {creating ? "生成中…" : "生成密钥"}
                </button>
            </form>
            {keys.length === 0 ? (
                <p style={{ color: v.textMuted, fontSize: 14, margin: 0 }}>暂无密钥。</p>
            ) : (
                <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
                    {keys.map((k) => (
                        <li
                            key={k.id}
                            style={{
                                display: "flex",
                                gap: 12,
                                alignItems: "center",
                                padding: "10px 0",
                                borderBottom: `1px solid ${v.border}`,
                                opacity: k.revokedAt ? 0.55 : 1,
                            }}
                        >
                            <code style={{ ...mono, fontSize: 13 }}>{k.prefix}…</code>
                            <span style={{ fontSize: 14, fontWeight: 550 }}>{k.name}</span>
                            <span className="ven-meta" style={{ fontSize: 12 }}>
                                {formatDateTime(k.createdAt)}
                            </span>
                            <span className="ven-meta" style={{ fontSize: 12 }}>
                                {k.lastUsedAt ? `最后使用 ${formatDateTime(k.lastUsedAt)}` : "从未使用"}
                            </span>
                            {k.revokedAt ? (
                                <span className="ven-meta" style={{ fontSize: 12, marginLeft: "auto" }}>
                                    已吊销
                                </span>
                            ) : (
                                <button
                                    className="ven-btn ven-btn-danger"
                                    type="button"
                                    style={{ padding: "3px 12px", fontSize: 12, marginLeft: "auto" }}
                                    onClick={() => setRevoking(k)}
                                >
                                    吊销
                                </button>
                            )}
                        </li>
                    ))}
                </ul>
            )}
            {error && <p style={{ color: v.danger, fontSize: 13, margin: "10px 0 0" }}>{error}</p>}
            <div style={{ marginTop: 10 }}>{node}</div>

            {/* 新建成功：明文仅此一次展示，关窗即丢弃 */}
            <Modal open={created !== null} onClose={closeCreated} width={520}>
                <h3 style={{ margin: "0 0 8px", fontSize: 16 }}>密钥已生成（{created?.view.name}）</h3>
                <p style={{ fontSize: 14, color: v.danger, fontWeight: 550, margin: "0 0 12px" }}>
                    密钥只显示这一次，关闭后将无法再次查看；请立即复制并妥善保存。
                </p>
                <div
                    style={{
                        ...mono,
                        padding: "12px 14px",
                        background: v.bg,
                        border: `1px solid ${v.border}`,
                        borderRadius: 6,
                        fontSize: 13,
                        overflowX: "auto",
                        whiteSpace: "nowrap",
                        marginBottom: 16,
                    }}
                >
                    {created?.raw}
                </div>
                <div style={{ display: "flex", justifyContent: "flex-end", gap: 10 }}>
                    <button className="ven-btn" type="button" onClick={copyRaw}>
                        {copied ? "已复制" : "复制"}
                    </button>
                    <button className="ven-btn ven-btn-primary" type="button" onClick={closeCreated}>
                        我已复制
                    </button>
                </div>
            </Modal>

            {/* 吊销确认（即时生效） */}
            <ConfirmModal
                open={revoking !== null}
                title={`吊销密钥「${revoking?.name ?? ""}」？`}
                message="吊销后立即生效，使用该密钥的请求将马上被拒绝；此操作不可恢复。"
                confirmText="吊销"
                danger
                onCancel={() => setRevoking(null)}
                onConfirm={revoke}
            />
        </section>
    );
}

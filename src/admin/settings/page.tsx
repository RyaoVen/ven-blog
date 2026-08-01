/** 后台-设置页：账号安全 / 作者资料 / 内容配置 / 文章分类 / 评论审核 */

import { ChangeEvent, FormEvent, useRef, useState } from "react";
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
    }) as AdminSettingsState;
    return (
        <AdminLayout route={bootstrap.route}>
            <PasswordSection />
            <ProfileSection profile={state.profile} />
            <ContentSection content={state.content} />
            <CategoriesSection categories={state.categories} />
            <ModerationSection initial={state.moderation} />
        </AdminLayout>
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

/* ===== 内容配置 ===== */
function ContentSection({ content }: { content: SettingsContent }) {
    const [paragraphs, setParagraphs] = useState(content.paragraphs.join("\n"));
    const [skills, setSkills] = useState(content.skills.map((s) => `${s.name}|${s.level}`).join("\n"));
    const [friends, setFriends] = useState(content.friends.map((f) => `${f.name}|${f.url}|${f.desc}`).join("\n"));
    const [quotes, setQuotes] = useState(content.quotes.map((q) => `${q.text}|${q.source}`).join("\n"));
    const [projects, setProjects] = useState(content.projects.map((p) => `${p.name}|${p.desc}|${p.url}`).join("\n"));
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const payload: SettingsContent = {
                paragraphs: lines(paragraphs),
                skills: lines(skills).map((line) => {
                    const [name = "", level = "know"] = line.split("|");
                    return { name: name.trim(), level: level.trim() };
                }),
                friends: lines(friends).map((line) => {
                    const [name = "", url = "", desc = ""] = line.split("|");
                    return { name: name.trim(), url: url.trim(), desc: desc.trim() };
                }),
                quotes: lines(quotes).map((line) => {
                    const [text = "", source = ""] = line.split("|");
                    return { text: text.trim(), source: source.trim() };
                }),
                projects: lines(projects).map((line) => {
                    const [name = "", desc = "", url = ""] = line.split("|");
                    return { name: name.trim(), desc: desc.trim(), url: url.trim() };
                }),
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
            <SectionTitle>内容配置（行格式，每行一条）</SectionTitle>
            <form onSubmit={onSubmit} style={{ display: "flex", flexDirection: "column", gap: 14 }}>
                <Field label="个人介绍段落（作者主页，每行一段）" value={paragraphs} onChange={setParagraphs} rows={5} />
                <Field label="技术栈（name|level，level ∈ deep/solid/know）" value={skills} onChange={setSkills} rows={4} />
                <Field label="友链（name|url|desc）" value={friends} onChange={setFriends} rows={4} />
                <Field label="收藏的句子（text|source）" value={quotes} onChange={setQuotes} rows={3} />
                <Field label="维护的项目（name|desc|url）" value={projects} onChange={setProjects} rows={3} />
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

function Field({ label, value, onChange, rows }: { label: string; value: string; onChange: (v: string) => void; rows: number }) {
    return (
        <label style={{ display: "flex", flexDirection: "column", gap: 6, fontSize: 14 }}>
            {label}
            <textarea className="ven-input" rows={rows} value={value} onChange={(e) => onChange(e.target.value)} style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace", fontSize: 13 }} />
        </label>
    );
}

function lines(raw: string): string[] {
    return raw
        .split("\n")
        .map((l) => l.trim())
        .filter(Boolean);
}

/* ===== 文章分类 ===== */
function CategoriesSection({ categories }: { categories: string[] }) {
    const [raw, setRaw] = useState(categories.join("\n"));
    const [submitting, setSubmitting] = useState(false);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        try {
            const resp = await fetch("/api/admin/settings/categories", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ categories: lines(raw) }),
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
                <Field label="每行一个分类名" value={raw} onChange={setRaw} rows={3} />
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

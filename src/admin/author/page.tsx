/** 后台-个人主页编辑：介绍段落 / 技术栈 / 友链 */

import { FormEvent, useState } from "react";
import type { PageAppProps } from "../../app/pageApp";
import { CheckIcon } from "../../lib/icons";
import { v } from "../../lib/theme";
import { AdminLayout } from "../adminLayout";
import { EditorBlock, RowShell } from "../editorBlocks";

interface FriendItem {
    name: string;
    url: string;
    desc: string;
}

interface SkillItem {
    name: string;
    level: string;
}

interface AuthorContentState {
    paragraphs: string[];
    skills: SkillItem[];
    friends: FriendItem[];
}

function useToast() {
    const [toast, setToast] = useState<string | null>(null);
    const show = (text: string) => {
        setToast(text);
        window.setTimeout(() => setToast(null), 2000);
    };
    const node = toast ? (
        <span style={{ display: "inline-flex", alignItems: "center", gap: 5, fontSize: 13, color: v.accent }}>
            <CheckIcon size={13} />
            {text}
        </span>
    ) : null;
    return { show, node };
}

export default function AdminAuthorPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { paragraphs: [], skills: [], friends: [] }) as AuthorContentState;
    const [paragraphs, setParagraphs] = useState<string[]>(state.paragraphs);
    const [skills, setSkills] = useState<SkillItem[]>(state.skills);
    const [friends, setFriends] = useState<FriendItem[]>(state.friends);
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    async function onSubmit(event: FormEvent) {
        event.preventDefault();
        setSubmitting(true);
        setError(null);
        try {
            const resp = await fetch("/api/admin/author/content", {
                method: "PUT",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({
                    paragraphs: paragraphs.map((p) => p.trim()).filter(Boolean),
                    skills: skills.filter((s) => s.name.trim()),
                    friends: friends.filter((f) => f.name.trim()),
                }),
            });
            if (!resp.ok) {
                setError("保存失败");
                return;
            }
            show("个人主页已保存（/author 稍后自动刷新）");
        } catch {
            setError("网络错误，请重试");
        } finally {
            setSubmitting(false);
        }
    }

    return (
        <AdminLayout route={bootstrap.route}>
            <form onSubmit={onSubmit} className="ven-card" style={{ padding: "22px 24px", display: "flex", flexDirection: "column", gap: 22 }}>
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
                {error && <p style={{ color: v.danger, fontSize: 13, margin: 0 }}>{error}</p>}
                <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
                    <button className="ven-btn ven-btn-primary" type="submit" disabled={submitting}>
                        {submitting ? "保存中…" : "保存个人主页"}
                    </button>
                    {node}
                </div>
            </form>
        </AdminLayout>
    );
}

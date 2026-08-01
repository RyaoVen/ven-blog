/** 后台-个人主页编辑：介绍段落 / 技术栈 / 展示柜 / 友链 */

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

interface ProjectItem {
    name: string;
    desc: string;
    url: string;
}

interface PostOption {
    id: string;
    title: string;
}

interface AuthorContentState {
    paragraphs: string[];
    skills: SkillItem[];
    friends: FriendItem[];
    projects: ProjectItem[];
    showcasePosts: number[];
    allPosts: PostOption[];
}

const SHOWCASE_MAX = 4;

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
    const state = (bootstrap.initialState ?? {
        paragraphs: [],
        skills: [],
        friends: [],
        projects: [],
        showcasePosts: [],
        allPosts: [],
    }) as AuthorContentState;
    const [paragraphs, setParagraphs] = useState<string[]>(state.paragraphs);
    const [skills, setSkills] = useState<SkillItem[]>(state.skills);
    const [friends, setFriends] = useState<FriendItem[]>(state.friends);
    const [projects, setProjects] = useState<ProjectItem[]>(state.projects);
    const [picks, setPicks] = useState<string[]>((state.showcasePosts ?? []).map(String));
    const [submitting, setSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const { show, node } = useToast();

    function togglePick(id: string) {
        setPicks((l) => {
            if (l.includes(id)) {
                return l.filter((x) => x !== id);
            }
            if (l.length >= SHOWCASE_MAX) {
                return l;
            }
            return [...l, id];
        });
    }

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
                    projects: projects.filter((p) => p.name.trim()),
                    showcasePosts: picks.map((x) => Number(x)),
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
                <EditorBlock
                    title={`展示柜文章（选取顺序即展示顺序，最多 ${SHOWCASE_MAX} 篇；不选则展示最新文章）`}
                    addLabel={picks.length > 0 ? `清空选择（${picks.length}/${SHOWCASE_MAX}）` : undefined}
                    onAdd={picks.length > 0 ? () => setPicks([]) : undefined}
                >
                    {state.allPosts.length === 0 ? (
                        <p style={{ color: v.textMuted, fontSize: 13, margin: 0 }}>还没有文章。</p>
                    ) : (
                        <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "flex", flexDirection: "column", gap: 6 }}>
                            {state.allPosts.map((p) => {
                                const order = picks.indexOf(p.id);
                                const selected = order !== -1;
                                const full = !selected && picks.length >= SHOWCASE_MAX;
                                return (
                                    <li key={p.id}>
                                        <button
                                            type="button"
                                            onClick={() => togglePick(p.id)}
                                            disabled={full}
                                            style={{
                                                display: "flex",
                                                alignItems: "center",
                                                gap: 10,
                                                width: "100%",
                                                textAlign: "left",
                                                padding: "8px 12px",
                                                fontSize: 14,
                                                cursor: full ? "not-allowed" : "pointer",
                                                borderRadius: 3,
                                                border: `1px solid ${selected ? v.accent : v.border}`,
                                                background: selected ? "var(--bg-inset)" : "transparent",
                                                color: full ? v.textMuted : v.text,
                                                opacity: full ? 0.55 : 1,
                                            }}
                                        >
                                            <span
                                                style={{
                                                    width: 20,
                                                    height: 20,
                                                    flexShrink: 0,
                                                    borderRadius: 3,
                                                    border: `1px solid ${selected ? v.accent : v.borderStrong}`,
                                                    background: selected ? v.accent : "transparent",
                                                    color: "#fff",
                                                    fontSize: 11,
                                                    fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace",
                                                    display: "inline-flex",
                                                    alignItems: "center",
                                                    justifyContent: "center",
                                                }}
                                            >
                                                {selected ? order + 1 : ""}
                                            </span>
                                            {p.title}
                                        </button>
                                    </li>
                                );
                            })}
                        </ul>
                    )}
                </EditorBlock>
                <EditorBlock title="展示柜项目（同时展示在首页仪表盘）" addLabel="添加项目" onAdd={() => setProjects((l) => [...l, { name: "", desc: "", url: "" }])}>
                    {projects.map((p, i) => (
                        <RowShell key={i} onRemove={() => setProjects((l) => l.filter((_, x) => x !== i))}>
                            <input className="ven-input" style={{ width: 140 }} value={p.name} onChange={(e) => setProjects((l) => l.map((x, xi) => (xi === i ? { ...x, name: e.target.value } : x)))} placeholder="项目名" />
                            <input className="ven-input" style={{ flex: 1, minWidth: 160 }} value={p.desc} onChange={(e) => setProjects((l) => l.map((x, xi) => (xi === i ? { ...x, desc: e.target.value } : x)))} placeholder="描述" />
                            <input className="ven-input" style={{ width: 160 }} value={p.url} onChange={(e) => setProjects((l) => l.map((x, xi) => (xi === i ? { ...x, url: e.target.value } : x)))} placeholder="https://…" />
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

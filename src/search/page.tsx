/** 搜索页：受控搜索框提交导航 /search?q=…，结果复用文章卡片列表渲染 */

import { FormEvent, useEffect, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";
import { PostList } from "../posts/list";
import type { SearchState } from "./types";

export default function SearchPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { q: "", results: [], scope: "", pluginResults: [] }) as SearchState;
    const [kw, setKw] = useState(state.q);
    const scope = state.scope || "all";

    // 导航（含前进/后退）后 initialState 变化而组件不卸载，同步输入框为当前生效关键词
    useEffect(() => setKw(state.q), [state.q]);

    function onSubmit(event: FormEvent) {
        event.preventDefault();
        navigate(`/search?q=${encodeURIComponent(kw.trim())}&scope=${encodeURIComponent(scope)}`);
    }

    function switchScope(next: string) {
        navigate(`/search?q=${encodeURIComponent(kw.trim())}&scope=${next}`);
    }

    return (
        <Layout>
            <header style={{ marginBottom: 24 }}>
                <h1 style={{ fontSize: 28 }}>搜索</h1>
            </header>
            <form onSubmit={onSubmit} style={{ display: "flex", gap: 12, marginBottom: 24 }}>
                <input
                    className="ven-input"
                    style={{ flex: 1 }}
                    value={kw}
                    onChange={(e) => setKw(e.target.value)}
                    placeholder="输入关键词，按标题或正文检索"
                />
                <button className="ven-btn ven-btn-primary" type="submit">
                    搜索
                </button>
            </form>
            <div style={{ display: "flex", gap: 8, marginBottom: 20 }}>
                {([
                    ["all", "全站"],
                    ["blog", "博客"],
                    ["docs", "文档"],
                ] as const).map(([value, label]) => (
                    <button
                        key={value}
                        type="button"
                        className="ven-meta"
                        onClick={() => switchScope(value)}
                        style={{
                            padding: "4px 14px",
                            borderRadius: 999,
                            border: `1px solid ${v.border}`,
                            background: scope === value ? v.textPrimary : "transparent",
                            color: scope === value ? v.bg : v.textSecondary,
                            cursor: "pointer",
                        }}
                    >
                        {label}
                    </button>
                ))}
            </div>
            {state.q === "" ? (
                <p style={{ color: v.textSecondary }}>输入关键词，检索文章的标题与正文。</p>
            ) : state.results.length === 0 ? (
                <p style={{ color: v.textSecondary }}>没有匹配「{state.q}」的结果。</p>
            ) : (
                <>
                    <p style={{ color: v.textSecondary, margin: "0 0 16px" }}>
                        「{state.q}」共 {state.results.length + (state.pluginResults?.reduce((n, g) => n + (g.hits?.length ?? 0), 0) ?? 0)} 条结果
                    </p>
                    {state.results.length > 0 && <PostList posts={state.results} />}
                    {(state.pluginResults ?? []).map((group) => (
                        <section key={group.provider} style={{ marginTop: state.results.length > 0 ? 28 : 0 }}>
                            <h2 style={{ fontSize: 16, marginBottom: 12 }}>
                                文档 <span style={{ color: v.textSecondary, fontSize: 13 }}>({group.hits.length})</span>
                            </h2>
                            {group.hits.length === 0 ? (
                                <p style={{ color: v.textSecondary }}>没有匹配的文档。</p>
                            ) : (
                                <ul style={{ listStyle: "none", margin: 0, padding: 0, display: "grid", gap: 10 }}>
                                    {group.hits.map((hit) => (
                                        <li key={hit.url}>
                                            <a
                                                href={hit.url}
                                                onClick={(e) => {
                                                    e.preventDefault();
                                                    navigate(hit.url);
                                                }}
                                                style={{ color: v.textPrimary, textDecoration: "none", fontWeight: 500 }}
                                            >
                                                {hit.title}
                                            </a>
                                            {hit.summary && (
                                                <p style={{ color: v.textSecondary, fontSize: 13, margin: "4px 0 0" }}>{hit.summary}</p>
                                            )}
                                        </li>
                                    ))}
                                </ul>
                            )}
                        </section>
                    ))}
                </>
            )}
        </Layout>
    );
}

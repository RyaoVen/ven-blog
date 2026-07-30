/** 搜索页：受控搜索框提交导航 /search?q=…，结果复用文章卡片列表渲染 */

import { FormEvent, useEffect, useState } from "react";
import type { PageAppProps } from "../app/pageApp";
import { navigate } from "../app/router";
import { Layout } from "../lib/layout";
import { v } from "../lib/theme";
import { PostList } from "../posts/list";
import type { SearchState } from "./types";

export default function SearchPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { q: "", results: [] }) as SearchState;
    const [kw, setKw] = useState(state.q);

    // 导航（含前进/后退）后 initialState 变化而组件不卸载，同步输入框为当前生效关键词
    useEffect(() => setKw(state.q), [state.q]);

    function onSubmit(event: FormEvent) {
        event.preventDefault();
        navigate(`/search?q=${encodeURIComponent(kw.trim())}`);
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
            {state.q === "" ? (
                <p style={{ color: v.textSecondary }}>输入关键词，检索文章的标题与正文。</p>
            ) : state.results.length === 0 ? (
                <p style={{ color: v.textSecondary }}>没有匹配「{state.q}」的结果。</p>
            ) : (
                <>
                    <p style={{ color: v.textSecondary, margin: "0 0 16px" }}>
                        「{state.q}」共 {state.results.length} 条结果
                    </p>
                    <PostList posts={state.results} />
                </>
            )}
        </Layout>
    );
}

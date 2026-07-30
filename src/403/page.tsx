/** 403 页：已登录但角色不足时由框架原地渲染（URL 不变） */

import { Layout } from "../lib/layout";
import { v } from "../lib/theme";

export default function ForbiddenPage() {
    return (
        <Layout>
            <div style={{ textAlign: "center", padding: "80px 0" }}>
                <h1 style={{ fontSize: 56, marginBottom: 12, fontFamily: "ui-monospace, SFMono-Regular, Menlo, Consolas, monospace" }}>
                    403
                </h1>
                <p style={{ color: v.textSecondary, marginBottom: 24 }}>当前账号没有访问该页面的权限。</p>
                <a href="/posts" className="ven-btn">
                    回到文章列表
                </a>
            </div>
        </Layout>
    );
}

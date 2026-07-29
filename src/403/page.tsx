/** 403 页：已登录但角色不足时由框架原地渲染（URL 不变） */

import { Layout } from "../lib/layout";

export default function ForbiddenPage() {
    return (
        <Layout>
            <h1>403 无权访问</h1>
            <p>当前账号没有访问该页面的权限。</p>
        </Layout>
    );
}

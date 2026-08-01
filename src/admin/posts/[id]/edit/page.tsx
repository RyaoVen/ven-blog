/** 后台-编辑文章（表单回填） */

import type { PageAppProps } from "../../../../app/pageApp";
import { v } from "../../../../lib/theme";
import { AdminLayout } from "../../../adminLayout";
import { PostEditor } from "../../../editor";
import type { PostState } from "../../../posts/types";

export default function AdminPostEditPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { post: null, categories: [] }) as PostState;
    return (
        <AdminLayout route={bootstrap.route}>
            <h2 style={{ fontSize: 18, marginBottom: 16 }}>编辑文章</h2>
            {state.post ? (
                <PostEditor post={state.post} categories={state.categories} />
            ) : (
                <p style={{ color: v.textSecondary }}>文章不存在。</p>
            )}
        </AdminLayout>
    );
}

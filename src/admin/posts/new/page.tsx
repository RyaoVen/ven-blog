/** 后台-新建文章（编辑器空态） */

import type { PageAppProps } from "../../../app/pageApp";
import { AdminLayout } from "../../adminLayout";
import { PostEditor } from "../../editor";

export default function AdminPostNewPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { post: null, categories: [] }) as PostState;
    return (
        <AdminLayout route={bootstrap.route}>
            <h2 style={{ fontSize: 18, marginBottom: 16 }}>新建文章</h2>
            <PostEditor post={null} categories={state.categories} />
        </AdminLayout>
    );
}

/** 后台-新建文档 */

import type { PageAppProps } from "../../../app/pageApp";
import { AdminLayout } from "../../adminLayout";
import { DocEditorForm } from "../DocEditorForm";

export default function AdminDocNewPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { docs: [] }) as { docs?: unknown[] };
    void state;
    return (
        <AdminLayout route={bootstrap.route}>
            <h2 style={{ fontSize: 18, marginBottom: 16 }}>新建文档</h2>
            <DocEditorForm mode="create" />
        </AdminLayout>
    );
}

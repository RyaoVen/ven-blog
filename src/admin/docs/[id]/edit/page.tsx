/** 后台-编辑文档（initialState.doc 由 Go 端按 :id 灌入） */

import type { PageAppProps } from "../../../../app/pageApp";
import { AdminLayout } from "../../../adminLayout";
import { DocEditorForm } from "../../DocEditorForm";
import type { DocView } from "../../../docs/types";

export default function AdminDocEditPage({ bootstrap }: PageAppProps) {
    const state = (bootstrap.initialState ?? { doc: null }) as { doc: DocView | null };
    return (
        <AdminLayout route={bootstrap.route}>
            <h2 style={{ fontSize: 18, marginBottom: 16 }}>编辑文档</h2>
            {state.doc ? (
                <DocEditorForm mode="edit" initial={state.doc} />
            ) : (
                <p>文档不存在。</p>
            )}
        </AdminLayout>
    );
}

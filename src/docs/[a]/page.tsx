/** /docs/:a 固定深度薄壳（过渡方案：上游需求 7 合入后切 [...slug]） */

import type { PageAppProps } from "../../app/pageApp";
import { DocDetailPage } from "../DocView";

export default function Page(props: PageAppProps) {
    return <DocDetailPage {...props} />;
}

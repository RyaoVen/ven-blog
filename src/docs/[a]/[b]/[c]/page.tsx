/** /docs/:a/:b/:c 固定深度薄壳 */

import type { PageAppProps } from "../../../../app/pageApp";
import { DocDetailPage } from "../../../DocView";

export default function Page(props: PageAppProps) {
    return <DocDetailPage {...props} />;
}

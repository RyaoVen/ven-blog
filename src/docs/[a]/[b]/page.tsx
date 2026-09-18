/** /docs/:a/:b —— 章节/小节阅读（可收起侧栏） */

import type { PageAppProps } from "../../../app/pageApp";
import { ChapterReaderPage } from "../../ChapterReader";

export default function Page(props: PageAppProps) {
    return <ChapterReaderPage {...props} />;
}

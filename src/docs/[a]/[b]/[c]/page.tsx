/** /docs/:a/:b/:c —— 深层小节（同章节阅读页兜底） */

import type { PageAppProps } from "../../../../app/pageApp";
import { ChapterReaderPage } from "../../../ChapterReader";

export default function Page(props: PageAppProps) {
    return <ChapterReaderPage {...props} />;
}

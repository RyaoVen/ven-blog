/** /docs/:a —— 顶层节点按语义分派：section=书籍详情（介绍+目录）；doc=单篇章节阅读 */

import type { PageAppProps } from "../../app/pageApp";
import { BookIntroPage } from "../BookIntro";
import { ChapterReaderPage } from "../ChapterReader";

export default function Page(props: PageAppProps) {
    const mode = ((props.bootstrap.initialState as { mode?: string } | null)?.mode) ?? "chapter";
    return mode === "book" ? <BookIntroPage {...props} /> : <ChapterReaderPage {...props} />;
}

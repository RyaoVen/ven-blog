/** /docs 书架页（docs 2.0） */

import type { PageAppProps } from "../app/pageApp";
import DocsBookshelfPage from "./Bookshelf";

export default function Page(props: PageAppProps) {
    return <DocsBookshelfPage {...props} />;
}

// 首页静态内容：作者维护的项目与收藏的句子（作者自策展内容，改这里后重启生效）。
package interfaces

// homeProject 作者维护的项目。
type homeProject struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
	URL  string `json:"url"`
}

// homeQuote 作者收藏的句子。
type homeQuote struct {
	Text   string `json:"text"`
	Source string `json:"source"`
}

// authorGitHub 作者 GitHub 主页（hero 卡片按钮）。
const authorGitHub = "https://github.com/RyaoVen"

var homeProjects = []homeProject{
	{
		Name: "ven_hybird",
		Desc: "Go 网关 + Node SSR 的混合渲染框架：SSR 直出、SPA 接管、ISR 物化、SSE 实时推送",
		URL:  "https://github.com/RyaoVen/ven_hybird",
	},
	{
		Name: "ven-blog",
		Desc: "基于 ven_hybird 的个人博客（本站）：DDD 分层、MySQL 持久化、Markdown 全内容块",
		URL:  "https://github.com/RyaoVen/ven-blog",
	},
}

var homeQuotes = []homeQuote{
	{Text: "简单是可靠的先决条件。", Source: "Edsger W. Dijkstra"},
	{Text: "接受秒级一致性窗口，换来失效路径的极简与永不阻塞业务。", Source: "VenHybird 设计哲学"},
	{Text: "先让它能工作，再让它好看，最后让它快。", Source: "Kent Beck（大意）"},
}

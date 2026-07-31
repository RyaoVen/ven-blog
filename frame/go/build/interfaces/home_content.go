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

// ===== 作者主页静态内容（作者自策展，改后重启生效） =====

// authorIntroParagraphs 个人介绍段落（技术方向/性格/为什么做博客与框架/计划）。
var authorIntroParagraphs = []string{
	"我是 RyaoVen，一个喜欢折腾渲染链路与后端工程的技术人。日常关注 Go、Node.js、Web 渲染（SSR/SPA/ISR）与开发者工具，对「双栈缝合」类的基础设施工具有执念。",
	"性格上偏工程师思维：喜欢把复杂系统拆成小而确定的部件，信奉「先让它能工作，再让它好看」。写代码追求克制——能一句话声明的绝不写十行命令。",
	"这个博客（ven-blog）是我的自留地：记录技术、沉淀想法，也是 VenHybird 框架的实景试验场。VenHybird 是我自研的 Go 网关 + Node SSR 混合渲染框架——SSR 直出、SPA 接管、ISR 物化、数据变更经事件总线失效并 SSE 推送到在线浏览器，本站全部页面都跑在它上面。",
	"接下来的计划：继续打磨框架的集群能力与开发体验，博客这边逐步上全文检索、更多互动形态和更丰富的可视化。也欢迎通过留言板与我交流。",
}

// authorSkill 一项技术栈标签（level 决定配色：deep 深入 / solid 熟练 / know 了解）。
type authorSkill struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

var authorSkills = []authorSkill{
	{Name: "Go", Level: "deep"},
	{Name: "Node.js", Level: "deep"},
	{Name: "TypeScript", Level: "solid"},
	{Name: "React", Level: "solid"},
	{Name: "MySQL", Level: "solid"},
	{Name: "Fiber", Level: "solid"},
	{Name: "SSR/ISR 渲染", Level: "deep"},
	{Name: "Docker", Level: "know"},
	{Name: "Rust", Level: "know"},
}

// friendLink 友链卡片。
type friendLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Desc string `json:"desc"`
}

var friendLinks = []friendLink{
	{Name: "VenHybird", URL: "https://github.com/RyaoVen/ven_hybird", Desc: "自研混合渲染框架"},
	{Name: "ven-blog", URL: "https://github.com/RyaoVen/ven-blog", Desc: "本站源码"},
	{Name: "GitHub", URL: "https://github.com/RyaoVen", Desc: "更多项目"},
	{Name: "虚位以待", URL: "#", Desc: "欢迎交换友链"},
	{Name: "虚位以待", URL: "#", Desc: "欢迎交换友链"},
}

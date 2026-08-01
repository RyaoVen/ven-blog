// 站点内容配置的内置默认值（库里无键时回退；设置页编辑后落库覆盖）。
package settingsapp

const defaultGitHub = "https://github.com/RyaoVen"

var defaultCategories = []string{"随笔", "技术", "生活"}

var defaultParagraphs = []string{
	"我是 RyaoVen，一个喜欢折腾渲染链路与后端工程的技术人。日常关注 Go、Node.js、Web 渲染（SSR/SPA/ISR）与开发者工具，对「双栈缝合」类的基础设施工具有执念。",
	"性格上偏工程师思维：喜欢把复杂系统拆成小而确定的部件，信奉「先让它能工作，再让它好看」。写代码追求克制——能一句话声明的绝不写十行命令。",
	"这个博客（ven-blog）是我的自留地：记录技术、沉淀想法，也是 VenHybird 框架的实景试验场。VenHybird 是我自研的 Go 网关 + Node SSR 混合渲染框架——SSR 直出、SPA 接管、ISR 物化、数据变更经事件总线失效并 SSE 推送到在线浏览器，本站全部页面都跑在它上面。",
	"接下来的计划：继续打磨框架的集群能力与开发体验，博客这边逐步上全文检索、更多互动形态和更丰富的可视化。也欢迎通过留言板与我交流。",
}

var defaultSkills = []Skill{
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

var defaultFriends = []FriendLink{
	{Name: "VenHybird", URL: "https://github.com/RyaoVen/ven_hybird", Desc: "自研混合渲染框架"},
	{Name: "ven-blog", URL: "https://github.com/RyaoVen/ven-blog", Desc: "本站源码"},
	{Name: "GitHub", URL: "https://github.com/RyaoVen", Desc: "更多项目"},
	{Name: "虚位以待", URL: "#", Desc: "欢迎交换友链"},
	{Name: "虚位以待", URL: "#", Desc: "欢迎交换友链"},
}

var defaultQuotes = []Quote{
	{Text: "简单是可靠的先决条件。", Source: "Edsger W. Dijkstra"},
	{Text: "接受秒级一致性窗口，换来失效路径的极简与永不阻塞业务。", Source: "VenHybird 设计哲学"},
	{Text: "先让它能工作，再让它好看，最后让它快。", Source: "Kent Beck（大意）"},
}

var defaultProjects = []Project{
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

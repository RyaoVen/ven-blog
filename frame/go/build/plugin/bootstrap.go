package plugin

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

// EnabledKey 返回插件启停 settings 键（plugin.<name>.enabled）。
func EnabledKey(name string) string { return "plugin." + name + ".enabled" }

// Bootstrap 插件注册内核：构造 → Meta 校验 → 依赖拓扑排序 → 启停判定 → 逐个 Register → 启动 Startable。
// factories 为插件入口清单（每插件一行 New，顺序即注册序，拓扑稳定）；
// builtinPrefixes 为宿主内置路由前缀表（防插件劫持内置路由；可 nil）。
// 返回已启用插件中实现 Stoppable 者（按注册序），宿主关停时应逆序调用 Stop。
// 任一环节失败即整体失败（fail-fast），不回滚已注册插件——启动中止即最终态。
func Bootstrap(factories []func() Plugin, rt *Runtime, builtinPrefixes []string) ([]Stoppable, error) {
	logger := rt.Logger
	if logger == nil {
		logger = log.Default()
	}

	plugins := make([]Plugin, 0, len(factories))
	for _, factory := range factories {
		// New() 为纯构造（契约要求无 IO/无副作用），构造本身不失败；
		// 失败模式集中在 Register/Start 阶段。
		plugins = append(plugins, factory())
	}

	metas := make([]Meta, len(plugins))
	for i, p := range plugins {
		metas[i] = p.Meta()
	}
	if err := validateMetas(metas, builtinPrefixes); err != nil {
		return nil, err
	}

	order, err := topoSort(metas)
	if err != nil {
		return nil, err
	}

	stopOrder := make([]Stoppable, 0, len(plugins))
	for _, i := range order {
		meta := metas[i]
		enabled, err := resolveEnabled(rt, meta)
		if err != nil {
			return nil, err
		}
		if !enabled {
			logger.Printf("plugin: %s 已通过设置关闭，跳过注册", meta.Name)
			continue
		}
		if err := plugins[i].Register(rt); err != nil {
			return nil, fmt.Errorf("plugin: %s 注册失败: %w", meta.Name, err)
		}
		if s, ok := plugins[i].(Startable); ok {
			if err := s.Start(); err != nil {
				return nil, fmt.Errorf("plugin: %s 启动失败: %w", meta.Name, err)
			}
		}
		if s, ok := plugins[i].(Stoppable); ok {
			stopOrder = append(stopOrder, s)
		}
		logger.Printf("plugin: %s v%s 已注册", meta.Name, meta.Version)
	}
	return stopOrder, nil
}

// Shutdown 逆序关停插件（宿主收到停机信号、SSE drain 之后调用）。
// 单个 Stop 失败只记日志，继续关停其余插件（关停路径不让一个卡死全部）。
func Shutdown(stopOrder []Stoppable, logger *log.Logger) {
	if logger == nil {
		logger = log.Default()
	}
	for i := len(stopOrder) - 1; i >= 0; i-- {
		if err := stopOrder[i].Stop(); err != nil {
			logger.Printf("plugin: 插件关停失败: %v", err)
		}
	}
}

// validateMetas 校验元数据：名称合法唯一、前缀所有权无冲突（插件间 + 内置路由）。
func validateMetas(metas []Meta, builtinPrefixes []string) error {
	seen := make(map[string]int, len(metas))
	owned := make(map[string]string, len(metas)) // 前缀 → 插件名
	claim := func(prefix, name string) error {
		prefix, ok := normalizePrefix(prefix)
		if !ok {
			return fmt.Errorf("plugin: %s PagePrefix 非法: %q", name, prefix)
		}
		if _, dup := owned[prefix]; dup {
			return fmt.Errorf("plugin: %s PagePrefix %s 与其他插件冲突", name, prefix)
		}
		owned[prefix] = name
		return nil
	}
	for i := range metas {
		m := metas[i]
		if m.Name == "" {
			return fmt.Errorf("plugin: 第 %d 个插件 Meta.Name 为空", i)
		}
		if !isKebab(m.Name) {
			return fmt.Errorf("plugin: %s 名称非法（须 kebab-case）", m.Name)
		}
		if prev, dup := seen[m.Name]; dup {
			return fmt.Errorf("plugin: 插件重名 %s（清单位置 %d 与 %d）", m.Name, prev, i)
		}
		seen[m.Name] = i
		for _, prefix := range m.PagePrefix {
			if err := claim(prefix, m.Name); err != nil {
				return err
			}
		}
		for _, dep := range m.Depends {
			if dep == m.Name {
				return fmt.Errorf("plugin: %s 依赖自身", m.Name)
			}
		}
	}
	// 段边界前缀冲突：/a 与 /a/b 冲突（子路径被夺权），/a 与 /ab 不冲突。
	prefixes := make([]string, 0, len(owned))
	for prefix := range owned {
		prefixes = append(prefixes, prefix)
	}
	sort.Strings(prefixes)
	for i := 1; i < len(prefixes); i++ {
		if isPrefixOf(prefixes[i-1], prefixes[i]) {
			return fmt.Errorf("plugin: PagePrefix 冲突：%s（%s）覆盖 %s（%s）",
				prefixes[i], owned[prefixes[i]], prefixes[i-1], owned[prefixes[i-1]])
		}
	}
	for _, builtin := range builtinPrefixes {
		b, ok := normalizePrefix(builtin)
		if !ok {
			continue
		}
		for _, prefix := range prefixes {
			if isPrefixOf(prefix, b) || isPrefixOf(b, prefix) {
				return fmt.Errorf("plugin: %s PagePrefix %s 与内置路由 %s 冲突", owned[prefix], prefix, b)
			}
		}
	}
	return nil
}

// topoSort 依赖拓扑排序：Kahn 算法，入度同层按清单序稳定输出；未知依赖/成环报错。
func topoSort(metas []Meta) ([]int, error) {
	index := make(map[string]int, len(metas))
	for i, m := range metas {
		index[m.Name] = i
	}
	indegree := make([]int, len(metas))
	successors := make([][]int, len(metas))
	for i, m := range metas {
		for _, dep := range m.Depends {
			j, ok := index[dep]
			if !ok {
				return nil, fmt.Errorf("plugin: %s 依赖的 %s 不在清单中", m.Name, dep)
			}
			indegree[i]++
			successors[j] = append(successors[j], i)
		}
	}
	// 就绪队列按清单序弹出 → 注册序 = 清单顺序的拓扑投影（稳定）。
	ready := make([]int, 0, len(metas))
	for i := range metas {
		if indegree[i] == 0 {
			ready = append(ready, i)
		}
	}
	order := make([]int, 0, len(metas))
	for len(ready) > 0 {
		i := ready[0]
		ready = ready[1:]
		order = append(order, i)
		for _, succ := range successors[i] {
			indegree[succ]--
			if indegree[succ] == 0 {
				ready = append(ready, succ)
			}
		}
		sort.Ints(ready)
	}
	if len(order) != len(metas) {
		return nil, fmt.Errorf("plugin: 依赖关系成环，涉及 %d 个插件", len(metas)-len(order))
	}
	return order, nil
}

// resolveEnabled 判定启停：显式 on/off 生效；未配置或读取出错回退 DefaultOn（读错仅记日志）。
func resolveEnabled(rt *Runtime, meta Meta) (bool, error) {
	if rt.Settings == nil {
		return meta.DefaultOn, nil
	}
	raw, err := rt.Settings.Get(EnabledKey(meta.Name))
	if err != nil {
		log.Printf("plugin: 读取 %s 失败（%v），回退默认 %v", EnabledKey(meta.Name), err, meta.DefaultOn)
		return meta.DefaultOn, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "1", "true":
		return true, nil
	case "off", "0", "false":
		return false, nil
	default:
		return meta.DefaultOn, nil
	}
}

// normalizePrefix 规范路由前缀：以 / 开头、去尾斜杠；每段须 kebab；"/" 与空串非法
// （插件不得声明根所有权）。允许多段（如 "/my/docs"）。
func normalizePrefix(prefix string) (string, bool) {
	if prefix == "" || !strings.HasPrefix(prefix, "/") {
		return prefix, false
	}
	trimmed := strings.TrimRight(prefix, "/")
	if trimmed == "" {
		return trimmed, false
	}
	for _, segment := range strings.Split(strings.TrimPrefix(trimmed, "/"), "/") {
		if !isKebab(segment) {
			return trimmed, false
		}
	}
	return trimmed, true
}

// isPrefixOf 判断 shorter 是否为 longer 的段边界前缀（/a 是 /a/b 的前缀；/a 不是 /ab 的）。
func isPrefixOf(shorter, longer string) bool {
	if shorter == longer {
		return true
	}
	return strings.HasPrefix(longer, shorter+"/")
}

// isKebab 判断是否 kebab-case（小写字母/数字/中缀连字符，禁首尾连字符与连续连字符）。
func isKebab(s string) bool {
	if s == "" {
		return false
	}
	prevHyphen := true // 首字符前视作连字符，天然禁首连字符
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			prevHyphen = false
		case r == '-':
			if prevHyphen {
				return false
			}
			prevHyphen = true
		default:
			return false
		}
	}
	return !prevHyphen
}

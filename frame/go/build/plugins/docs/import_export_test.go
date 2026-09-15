package docs

import (
	"strings"
	"testing"
)

func TestParseFrontmatter_Full(t *testing.T) {
	fm, body, err := parseFrontmatter("---\ntitle: Attention\nsummary: 注意力\ntags: ml, notes\norder: 3\nstatus: draft\n---\n\n# 正文\n内容")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fm.Title != "Attention" || fm.Summary != "注意力" {
		t.Fatalf("meta 不对：%+v", fm)
	}
	if len(fm.Tags) != 2 || fm.Tags[0] != "ml" {
		t.Fatalf("tags 不对：%+v", fm.Tags)
	}
	if fm.Order == nil || *fm.Order != 3 {
		t.Fatalf("order 不对：%+v", fm.Order)
	}
	if fm.Status != StatusDraft {
		t.Fatalf("status 不对：%s", fm.Status)
	}
	if !strings.HasPrefix(body, "# 正文") {
		t.Fatalf("body 不对：%q", body)
	}
}

func TestParseFrontmatter_Absent(t *testing.T) {
	fm, body, err := parseFrontmatter("# 纯正文")
	if err != nil || fm.Title != "" || body != "# 纯正文" {
		t.Fatalf("无 frontmatter 应原文返回：%+v %q %v", fm, body, err)
	}
}

func TestImportExportRoundTrip(t *testing.T) {
	svc := newTestService()
	content := "---\ntitle: Attention 机制\nsummary: 注意力笔记\ntags: ml, notes\norder: 2\n---\n\n# Attention\n\nTransformer 的核心。"
	res := svc.Import([]ImportFile{
		{Path: "notes/ai/attention", Content: content},
		{Path: "notes/ai/rlhf", Content: "# RLHF\n\n无 frontmatter 正文。"},
	})
	if res.Created != 2 || len(res.Failed) != 0 {
		t.Fatalf("导入应新建 2 个：%+v", res)
	}
	// 隐式 section notes/ai 存在。
	if _, err := svc.Get("notes/ai"); err != nil {
		t.Fatalf("隐式父缺失: %v", err)
	}
	// upsert 更新。
	res2 := svc.Import([]ImportFile{{Path: "notes/ai/attention", Content: content + "\n追加。"}})
	if res2.Updated != 1 || res2.Created != 0 {
		t.Fatalf("二次导入应为更新：%+v", res2)
	}
	// export 往返。
	files, err := svc.Export("notes")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(files) != 4 { // notes、notes/ai、attention、rlhf
		t.Fatalf("导出应 4 文件（含隐式 section），得 %d", len(files))
	}
	var attention ExportFile
	for _, f := range files {
		if f.Path == "notes/ai/attention" {
			attention = f
		}
	}
	if !strings.Contains(attention.Content, "title: Attention 机制") || !strings.Contains(attention.Content, "tags: ml, notes") || !strings.Contains(attention.Content, "Transformer 的核心。") {
		t.Fatalf("导出应还原 frontmatter+正文：%s", attention.Content)
	}
	// 再导入导出文件 = 往返一致。
	reimported := make([]ImportFile, 0, len(files))
	for _, f := range files {
		reimported = append(reimported, ImportFile{Path: f.Path, Content: f.Content})
	}
	res3 := svc.Import(reimported)
	if res3.Failed != nil || res3.Updated+res3.Created != len(files) {
		t.Fatalf("导出文件应可全量再导入：%+v", res3)
	}
}

func TestImport_FailureCollected(t *testing.T) {
	svc := newTestService()
	res := svc.Import([]ImportFile{
		{Path: "", Content: "# x"},           // 空 path
		{Path: "Bad_Case/x", Content: "# x"}, // 非法 slug
		{Path: "ok/doc", Content: "# OK"},    // 成功
	})
	if len(res.Failed) != 2 {
		t.Fatalf("应收集 2 个失败：%+v", res.Failed)
	}
	if res.Created != 1 {
		t.Fatalf("应成功 1 个：%+v", res)
	}
}

func TestExport_SubtreeFilter(t *testing.T) {
	svc := newTestService()
	svc.Create(CreateInput{Title: "A", Path: "ai/a"})
	svc.Create(CreateInput{Title: "Z", Path: "zz/z"})
	files, err := svc.Export("ai")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(files) != 2 { // ai 自身 + ai/a
		t.Fatalf("子树导出应 2 文件，得 %d", len(files))
	}
	for _, f := range files {
		if !strings.HasPrefix(f.Path, "ai") {
			t.Fatalf("不应含子树外文件：%s", f.Path)
		}
	}
}

func TestBuildFrontmatter_QuotingFree(t *testing.T) {
	doc := &Doc{Title: "T", Summary: "S", Tags: []string{"a", "b"}, SortOrder: 1, Status: StatusPublished, Content: "C"}
	out := buildFrontmatter(doc, true)
	if !strings.HasPrefix(out, "---\ntitle: T\n") || !strings.HasSuffix(out, "C") {
		t.Fatalf("frontmatter 结构不对：%s", out)
	}
}

package core

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

// Tail 返回文件最后 n 行（默认 100）；文件不存在时返回友好错误。
// 从文件末尾分块读取，适合大日志。
func Tail(path string, n int) ([]string, error) {
	if n <= 0 {
		n = 100
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("日志不存在：%s（未启动过或已清理）", path)
		}
		return nil, err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := st.Size()
	if size == 0 {
		return []string{}, nil
	}

	const chunkSize = 64 * 1024
	var buf []byte
	newlines := 0
	pos := size
	for pos > 0 && newlines <= n {
		read := int64(chunkSize)
		if read > pos {
			read = pos
		}
		pos -= read
		tmp := make([]byte, read)
		if _, err := f.ReadAt(tmp, pos); err != nil {
			return nil, err
		}
		buf = append(tmp, buf...)
		newlines = bytes.Count(buf, []byte{'\n'})
	}

	lines := strings.Split(string(buf), "\n")
	// 文件以 \n 结尾时去掉 Split 产生的末尾空段
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.TrimSuffix(l, "\r"))
	}
	return out, nil
}

// PrintLogs 打印 node/go 两个日志文件的 tail（子命令 logs 用；缺失的日志打印提示不报错）。
func PrintLogs(root string, n int, w io.Writer) error {
	for _, it := range []struct {
		name string
		path string
	}{{"node", LogPath(root, ProcNode)}, {"go", LogPath(root, ProcGo)}} {
		fmt.Fprintf(w, "===== logs/%s.log =====\n", it.name)
		lines, err := Tail(it.path, n)
		if err != nil {
			fmt.Fprintln(w, err)
			continue
		}
		for _, l := range lines {
			fmt.Fprintln(w, l)
		}
	}
	return nil
}

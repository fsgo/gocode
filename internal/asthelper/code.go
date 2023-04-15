// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/9

package asthelper

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func NodeLineNo(pass *analysis.Pass, node ast.Node) string {
	p := node.Pos()
	pos := pass.Fset.Position(p)
	rn, _ := filepath.Rel(wd, pos.Filename)
	return fmt.Sprintf("%s:%d", rn, pos.Line)
}

func NodeCode(pass *analysis.Pass, node ast.Node, line int) string {
	bf := &bytes.Buffer{}
	format.Node(bf, pass.Fset, node)
	lines := strings.SplitN(bf.String(), "\n", line+1)
	if len(lines) > line {
		lines = lines[:line]
	}
	lines = strings.Split(strings.TrimSpace(strings.Join(lines, "\n")), "\n")
	pos := pass.Fset.Position(node.Pos())
	for i := 0; i < len(lines); i++ {
		lines[i] = fmt.Sprintf("%-5d %s", i+pos.Line, lines[i])
	}
	return strings.Join(lines, "\n")
}

func NodeCount(n ast.Node, fn func(c ast.Node) bool) int {
	var num int
	ast.Inspect(n, func(node ast.Node) bool {
		if fn(node) {
			num++
		}
		return true
	})
	return num
}

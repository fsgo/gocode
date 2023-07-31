// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/7/28

package asthelper

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"runtime/debug"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/tools/go/analysis"
)

func RecoverFatal(pass *analysis.Pass, node ast.Node, re any) {
	lineX := strings.Repeat("-", 120) + "\n"
	stack := debug.Stack()
	ts := token.NewFileSet()
	var msg string
	msg += color.RedString("panic when doNode: %v at\n", re)
	msg += lineX
	msg += "file: " + NodeLineNo(pass, node) + "\n"
	msg += lineX
	msg += NodeCode(pass, node, 100) + "\n"
	msg += lineX
	msg += "AstNode:\n"

	w := &bytes.Buffer{}
	_ = ast.Fprint(w, ts, node, ast.NotNilFilter)
	msg += w.String() + "\n"
	msg += lineX
	msg += color.RedString(fmt.Sprint(re)) + "\n"
	msg += string(stack)
	log.Fatalln(msg)
}

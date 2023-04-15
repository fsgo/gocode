// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/9

package asthelper

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/fsgo/gocode/zpass"
)

var wd string

func init() {
	c, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	wd = c
}

func FindAstFileByObject(pass *analysis.Pass, ov types.Object) (f *ast.File, err error) {
	p := ov.Pos()
	tokenFile := pass.Fset.File(p)
	for _, astFile := range pass.Files {
		tokenFile2 := pass.Fset.File(astFile.Pos())
		if tokenFile.Name() == tokenFile2.Name() {
			return astFile, nil
		}
	}
	mod := parser.Mode(0) | parser.ParseComments
	f, err = parser.ParseFile(pass.Fset, tokenFile.Name(), nil, mod)
	if zpass.IsDebugVerbose() {
		log.Println("parser.ParseFile:", tokenFile.Name(), ov.Pkg().Path(), err)
	}
	if err != nil {
		return nil, fmt.Errorf("parseFile %s failed: %v", tokenFile.Name(), err)
	}
	pass.Files = append(pass.Files, f)

	// conf := packages.Config{
	// 	Mode:  packages.LoadSyntax,
	// 	Tests: false,
	// }
	//
	// initial, err := packages.Load(&conf, "./...")

	return f, nil
}

func RangeImports(f *ast.File, fn func(pkg string) bool) {
	for i := 0; i < len(f.Imports); i++ {
		ni := f.Imports[i]
		p1, _ := strconv.Unquote(ni.Path.Value)
		if !fn(p1) {
			return
		}
	}
}

func HasImport(f *ast.File, pkg string) bool {
	var has bool
	RangeImports(f, func(name string) bool {
		if pkg == name {
			has = true
			return false
		}
		return true
	})
	return has
}

func IsGoFile(f *token.File) bool {
	return strings.HasSuffix(f.Name(), ".go")
}

func IsGoTestFile(f *token.File) bool {
	return strings.HasSuffix(f.Name(), "_test.go")
}

func RelName(name string) string {
	rn, err := filepath.Rel(wd, name)
	if err != nil {
		return name
	}
	return rn
}

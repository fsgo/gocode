// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/9

package gorecover

import (
	"flag"
	"go/ast"
	"go/types"
	"log"
	"sync"

	"github.com/fatih/color"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/fsgo/gocode/internal/asthelper"
)

const Doc = `find goroutines not recovered
with flag "-debug v" for verbose
`

var Analyzer = &analysis.Analyzer{
	Name: "gorecover",
	Doc:  Doc,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
	},
	Run: run,
}

var debug bool

func run(pass *analysis.Pass) (any, error) {
	log.SetPrefix("")
	if ft := flag.Lookup("debug"); ft != nil {
		debug = ft.Value.String() != ""
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.GoStmt)(nil),
	}

	var ignore bool
	inspect.Preorder(nodeFilter, func(node ast.Node) {
		if ignore {
			return
		}

		if nf, ok := node.(*ast.File); ok {
			ignore = checkIgnore(pass, nf)
			if ignore {
				return
			}
		}
		gs, ok := node.(*ast.GoStmt)
		if !ok {
			return
		}
		check(pass, gs)
	})
	return nil, nil
}

func countGoStmt(f *ast.File) int {
	return asthelper.NodeCount(f, func(c ast.Node) bool {
		_, ok := c.(*ast.GoStmt)
		return ok
	})
}

var checked sync.Map

func checkIgnore(pass *analysis.Pass, nf *ast.File) bool {
	tokenFile := pass.Fset.File(nf.Pos())

	if !asthelper.IsGoFile(tokenFile) {
		return true
	}

	if asthelper.IsGoTestFile(tokenFile) {
		return true
	}

	rn := asthelper.RelName(tokenFile.Name())

	if asthelper.HasImport(nf, "testing") {
		if debug {
			log.Println(`ignored: has import "testing":`, rn, ", has GoStmt:", countGoStmt(nf))
		}
		return true
	}

	if _, ok := checked.Load(tokenFile.Name()); ok {
		return true
	}
	checked.Store(tokenFile.Name(), true)

	return false
}

var successID int
var failID int

func check(pass *analysis.Pass, gs *ast.GoStmt) (ok bool) {
	code1 := asthelper.NodeCode(pass, gs, 10)
	defer func() {
		if !ok {
			return
		}
		successID++
		str1 := color.CyanString("[%d] GoStmt recovered >> %s\n", successID, asthelper.NodeLineNo(pass, gs))
		str2 := color.GreenString("\nrecover() at %s\n", asthelper.NodeLineNo(pass, recoverAt))
		code2 := asthelper.NodeCode(pass, recoverAt, 1)
		if debug {
			log.Println(str1 + code1 + str2 + code2)
		}
	}()
	switch vt0 := gs.Call.Fun.(type) {
	case *ast.FuncLit:
		// go func(){}
		if hasRecover(vt0.Body) {
			return true
		}
	case *ast.Ident:
		// go goFuncWithoutRecover()
		fd, ok := vt0.Obj.Decl.(*ast.FuncDecl) // fd 是 goFuncWithoutRecover 定义
		if !ok {
			return true
		}
		if hasRecover(fd.Body) {
			return true
		}
	case *ast.SelectorExpr:
		// go abc.fn1(user.fn2)
		ov := pass.TypesInfo.ObjectOf(vt0.Sel)
		astFile := findAstFileByObject(pass, ov)
		if astFile == nil {
			pass.Reportf(gs.Pos(), "cannot find *ast.File")
			return false
		}

		funcNode := findFuncDeclNode(astFile, vt0.Sel.Name)
		if funcNode == nil {
			pass.Reportf(gs.Pos(), "cannot find *ast.FuncDecl")
			return false
		}
		if hasRecover(funcNode.Body) {
			return true
		}
	default:
		pass.Reportf(gs.Pos(), "unsupported type: %T", gs.Call.Fun)
	}
	failID++
	pass.Reportf(gs.Pos(), "[%d] goroutine not recovered, func type is %T \n%s", failID, gs.Call.Fun, code1)
	return false
}

func findFuncDeclNode(f *ast.File, name string) *ast.FuncDecl {
	for _, d := range f.Decls {
		if funcDecl, ok := d.(*ast.FuncDecl); ok && funcDecl.Name.Name == name {
			return funcDecl
		}
	}
	return nil
}

func findAstFileByObject(pass *analysis.Pass, ov types.Object) *ast.File {
	f, err := asthelper.FindAstFileByObject(pass, ov)
	if err == nil {
		return f
	}
	pass.Reportf(ov.Pos(), err.Error())
	return nil
}

var recoverAt ast.Node

func hasRecover(bs *ast.BlockStmt) bool {
	for _, blockStmt := range bs.List {
		deferStmt, ok := blockStmt.(*ast.DeferStmt) // 是否包含defer 语句
		if !ok {
			continue
		}
		switch vt0 := deferStmt.Call.Fun.(type) {
		case *ast.FuncLit:
			// 判断是否有 defer func(){ }()
			for i := range vt0.Body.List {
				stmt := vt0.Body.List[i]
				if isStmtRecovered(stmt) {
					recoverAt = stmt
					return true
				}
			}
		}
	}
	return false
}

func isStmtRecovered(stmt ast.Stmt) bool {
	switch vt1 := stmt.(type) {
	case *ast.ExprStmt:
		// recover()
		if isRecoverExpr(vt1.X) {
			return true
		}
	case *ast.IfStmt:
		// if r:=recover();r!=nil{}
		as, ok := vt1.Init.(*ast.AssignStmt)
		if !ok {
			return false
		}
		if isRecoverExpr(as.Rhs[0]) {
			return true
		}
	case *ast.AssignStmt:
		// r=:recover
		if isRecoverExpr(vt1.Rhs[0]) {
			return true
		}
	}
	return false
}

func isRecoverExpr(expr ast.Expr) bool {
	ac, ok := expr.(*ast.CallExpr) // r:=recover()
	if !ok {
		return false
	}
	id, ok := ac.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	if id.Name == "recover" {
		return true
	}
	return false
}

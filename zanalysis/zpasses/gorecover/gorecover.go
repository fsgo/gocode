// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/9

package gorecover

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"log"
	"runtime"
	"sync"

	"github.com/fatih/color"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/fsgo/gocode/internal/asthelper"
	"github.com/fsgo/gocode/zpass"
)

const Doc = `find goroutines not recovered
with flag "-debug v" for verbose
`

var container = &zpass.Container{}

var Analyzer = &analysis.Analyzer{
	Name: "zpass_go_recover",
	Doc:  Doc,
	Requires: []*analysis.Analyzer{
		inspect.Analyzer,
		zpass.NewInitAnalyzer(container),
	},
	Run: run,
}

func run(pass *analysis.Pass) (any, error) {
	if zpass.IsTestPkg(pass.Pkg.Path()) {
		return nil, nil
	}

	if zpass.IsTrace() {
		log.Printf("[%s] start check pkg: %s: %s\n", pass.Analyzer.Name, pass.Pkg.Name(), pass.Pkg.Path())
	}
	container.SetCurrentPass(pass)

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
		if zpass.IsDebugVerbose() {
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
	// 默认就是自己
	// 为了兼容 go panic() 等不需要 recover 的场景
	recoverAt = gs

	code1 := asthelper.NodeCode(pass, gs, 10)
	defer func() {
		if re := recover(); re != nil {
			bf := make([]byte, 4096)
			n := runtime.Stack(bf, false)
			pass.Reportf(gs.Pos(), "panic: %v, please report a bug,\nGoStmt Code:\n%s\nStack:\n%s", re, code1, bf[:n])
		}
	}()

	defer func() {
		if !ok {
			return
		}
		successID++

		var skipped string
		if recoverAt == gs {
			skipped = "(skipped or don't need recover)"
		}

		str1 := color.CyanString("[%d] GoStmt recovered >> %s\n", successID, asthelper.NodeLineNo(pass, gs))
		str2 := color.GreenString("\nrecover() at %s %s\n", asthelper.NodeLineNo(pass, recoverAt), skipped)
		code2 := asthelper.NodeCode(pass, recoverAt, 2)
		if zpass.IsDebugVerbose() {
			log.Println(str1 + code1 + str2 + code2)
		}
	}()

	checkIdent := func(id *ast.Ident) bool {
		ok1, err1 := isIdentRecover(pass, id)
		if err1 != nil {
			pass.Reportf(gs.Pos(), err1.Error())
		}
		return ok1
	}

	switch vt0 := gs.Call.Fun.(type) {
	case *ast.FuncLit:
		// go func(){}
		if isBlockStmtRecovered(vt0.Body) {
			return true
		}
		if len(vt0.Body.List) == 1 {
			if ex, ok1 := vt0.Body.List[0].(*ast.ExprStmt); ok1 && isExprStmtRecovered(pass, ex) {
				return true
			}
		}

	case *ast.Ident:
		if vt0.Obj == nil {
			// go panic("hello")
			if vt0.Name == "panic" {
				return true
			}

			// go noPanic(fn1)
			if checkIdent(vt0) {
				return true
			}
			break
		}
		// go goFuncWithoutRecover()
		fd, ok := vt0.Obj.Decl.(*ast.FuncDecl) // fd 是 goFuncWithoutRecover 定义
		if !ok {
			return true
		}
		if isBlockStmtRecovered(fd.Body) {
			return true
		}
	case *ast.SelectorExpr:
		// go abc.fn1(user.fn2)
		if checkIdent(vt0.Sel) {
			return true
		}
	// case *ast.IndexExpr:
	// go task.jobs[i]()
	default:
		pass.Reportf(gs.Pos(), "unsupported type: %T", gs.Call.Fun)
	}
	failID++
	pass.Reportf(gs.Pos(), "[%d] goroutine not recovered, func type is %T \n%s", failID, gs.Call.Fun, code1)
	return false
}

func isIdentRecover(pass *analysis.Pass, id *ast.Ident) (bool, error) {
	ov := pass.TypesInfo.ObjectOf(id)
	if ov == nil {
		return false, fmt.Errorf("object is nil for:%s", id.String())
	}
	ap, astFile := findAstFileByObject(pass, ov)
	if astFile == nil {
		return false, errors.New("cannot find *ast.File")
	}

	pass = ap

	funcNode := findFuncDeclNode(astFile, id.Name)
	if funcNode == nil {
		return false, errors.New("cannot find *ast.FuncDecl")
	}
	if isBlockStmtRecovered(funcNode.Body) {
		return true, nil
	}

	// 多个层级的调用
	// func abc(fn func()){
	// 	   xba.noPanic(fn)
	// }
	if funcNode.Body != nil && len(funcNode.Body.List) == 1 {
		if ep, ok1 := funcNode.Body.List[0].(*ast.ExprStmt); ok1 && isExprStmtRecovered(pass, ep) {
			return true, nil
		}
	}
	return false, nil
}

func isExprStmtRecovered(pass *analysis.Pass, nd *ast.ExprStmt) bool {
	if ce, ok2 := nd.X.(*ast.CallExpr); ok2 {
		if ceID, ok3 := ce.Fun.(*ast.Ident); ok3 && ceID.Obj != nil {
			if o1, ok4 := ceID.Obj.Decl.(*ast.FuncDecl); ok4 {
				if isBlockStmtRecovered(o1.Body) {
					return true
				}
			}
		}

		if se, ok3 := ce.Fun.(*ast.SelectorExpr); ok3 {
			ok4, err := isIdentRecover(pass, se.Sel)
			if ok4 {
				return true
			}
			if err != nil {
				pass.Reportf(se.Pos(), err.Error())
			}
		}
	}
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

func findAstFileByObject(pass *analysis.Pass, ov types.Object) (*analysis.Pass, *ast.File) {
	ap, f, err := container.FindAstFileByObject(ov)
	if err == nil {
		return ap, f
	}
	pass.Reportf(ov.Pos(), err.Error())
	return nil, nil
}

var recoverAt ast.Node

func isBlockStmtRecovered(bs *ast.BlockStmt) bool {
	// func body empty when with go:linkname
	if bs == nil {
		return false
	}
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

// 判断下列：
// recover()
// _=recover()
// if re:=recover();re!=nil{
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

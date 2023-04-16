// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/13

package zpass

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/types"
	"log"
	"sync/atomic"

	"github.com/fsgo/fsgo/fssync"
	"golang.org/x/tools/go/analysis"
)

type Container struct {
	Tests    bool
	passList fssync.Map[string, *analysis.Pass]
	current  atomic.Pointer[analysis.Pass]
}

func (c *Container) AddPass(p *analysis.Pass) {
	if IsTestPkg(p.Pkg.Path()) && !c.Tests {
		return
	}
	pp := PkgPath(p.Pkg.Path())
	c.passList.Store(pp, p)
	if IsTrace() {
		log.Printf("[%s] AddPass %03d, pkg: %s\n", p.Analyzer.Name, c.passList.Count(), pp)
	}
}

func (c *Container) SetCurrentPass(p *analysis.Pass) {
	c.current.Store(p)
}

func (c *Container) CurrentPass() *analysis.Pass {
	return c.current.Load()
}

func (c *Container) FindPass(pkg string) *analysis.Pass {
	v, _ := c.passList.Load(pkg)
	return v
}

func (c *Container) FindAstFileByObject(ov types.Object) (ap *analysis.Pass, f *ast.File, err error) {
	curPass := c.CurrentPass()
	foundPass := c.FindPass(ov.Pkg().Path())
	p := ov.Pos()
	tokenFile := curPass.Fset.File(p)

	if IsTrace() {
		log.Printf("[FindAstFile] curPkg=%s, ov(Pkg=%s, Name=%s), tokenFile=%s, foundPass=%v\n",
			curPass.Pkg.Path(),
			ov.Pkg().Path(),
			ov.Name(),
			tokenFile.Name(),
			foundPass != nil,
		)
	}

	if foundPass != nil {
		for _, astFile := range foundPass.Files {
			tokenFile2 := foundPass.Fset.File(astFile.Pos())
			if tokenFile.Name() == tokenFile2.Name() {
				return foundPass, astFile, nil
			}
		}
		return nil, nil, fmt.Errorf("not found %s in pkg %s", tokenFile.Name(), ov.Pkg().Path())
	}

	// 目前已解析并加载所有依赖，所以下面的理论不会被执行到
	mod := parser.Mode(0) | parser.ParseComments
	f, err = parser.ParseFile(curPass.Fset, tokenFile.Name(), nil, mod)
	if IsDebugVerbose() {
		log.Println("parser.ParseFile:", tokenFile.Name(), ov.Pkg().Path(), err)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("parseFile %s failed: %v", tokenFile.Name(), err)
	}
	curPass.Files = append(curPass.Files, f)
	return curPass, f, nil
}

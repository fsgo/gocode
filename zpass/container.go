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
	passList fssync.Map[string, *analysis.Pass]
	current  atomic.Pointer[analysis.Pass]
}

func (c *Container) AddPass(p *analysis.Pass) {
	c.passList.Store(p.Pkg.Path(), p)
	if IsDebugVerbose() {
		log.Printf("[%s] AddPass %03d, pkg: %s\n", p.Analyzer.Name, c.passList.Count(), p.Pkg.Path())
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
	pass := c.FindPass(ov.Pkg().Path())
	p := ov.Pos()
	cur := c.CurrentPass()
	tokenFile := cur.Fset.File(p)
	if pass != nil {
		for _, astFile := range pass.Files {
			tokenFile2 := pass.Fset.File(astFile.Pos())
			if tokenFile.Name() == tokenFile2.Name() {
				return pass, astFile, nil
			}
		}
		return nil, nil, fmt.Errorf("not found %s in pkg %s", tokenFile.Name(), ov.Pkg().Path())
	}
	mod := parser.Mode(0) | parser.ParseComments
	f, err = parser.ParseFile(cur.Fset, tokenFile.Name(), nil, mod)
	if IsDebugVerbose() {
		log.Println("parser.ParseFile:", tokenFile.Name(), ov.Pkg().Path(), err)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("parseFile %s failed: %v", tokenFile.Name(), err)
	}
	cur.Files = append(cur.Files, f)
	return cur, f, nil
}

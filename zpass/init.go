// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/15

package zpass

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

func NewInitAnalyzer(c *Container) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: "zpass_init",
		Doc:  `add all pass to DefaultContainer`,
		Requires: []*analysis.Analyzer{
			inspect.Analyzer,
			// findcall.Analyzer,
		},
		Run: func(pass *analysis.Pass) (any, error) {
			tryParserFlags()
			c.AddPass(pass)
			return nil, nil
		},
		FactTypes: []analysis.Fact{new(foundFact)},
	}
}

type foundFact struct{}

func (*foundFact) String() string {
	return "found"
}

func (*foundFact) AFact() {}

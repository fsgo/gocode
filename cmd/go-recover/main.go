// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/9

package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/fsgo/gocode/zanalysis/zpasses/gorecover"
)

func main() {
	singlechecker.Main(gorecover.Analyzer)
}

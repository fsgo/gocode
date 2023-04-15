// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/13

package zpass

import (
	"flag"
	"strings"
	"sync"

	"github.com/fsgo/fsgo/fssync/fsatomic"
)

var debug fsatomic.String

var parserOnce sync.Once

func tryParserFlags() {
	parserOnce.Do(func() {
		if ft := flag.Lookup("debug"); ft != nil {
			debug.Store(ft.Value.String())
		}
	})
}

// IsDebug 判断是否指定 debug 类型
// Debug is a set of single-letter flags:
//
//	f	show [f]acts as they are created
//	p	disable [p]arallel execution of analyzers
//	s	do additional [s]anity checks on fact types and serialization
//	t	show [t]iming info (NB: use 'p' flag to avoid GC/scheduler noise)
//	v	show [v]erbose logging
func IsDebug(s string) bool {
	return strings.Contains(debug.Load(), s)
}

// IsDebugVerbose  show verbose logging
func IsDebugVerbose() bool {
	return IsDebug("v")
}

// IsDebugTiming show timing info
func IsDebugTiming() bool {
	return IsDebug("t")
}

// IsDebugFacts how facts as they are created
func IsDebugFacts() bool {
	return IsDebug("f")
}

// IsDebugParallel disable parallel execution of analyzers
func IsDebugParallel() bool {
	return IsDebug("p")
}

// IsDebugSanity do additional sanity checks on fact types and serialization
func IsDebugSanity() bool {
	return IsDebug("s")
}

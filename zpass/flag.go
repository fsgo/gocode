// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/13

package zpass

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fsgo/fsgo/fssync/fsatomic"

	"github.com/fsgo/gocode/internal/xflag"
)

var debug fsatomic.String

var parserOnce sync.Once

var vv = flag.Bool("vv", false, "show verbose trace logs")

func tryParserFlags() {
	parserOnce.Do(func() {
		flag.Parse()
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

// IsTrace verbose trace logs
func IsTrace() bool {
	return *vv
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

func init() {
	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		xflag.PrintDefaults(flag.CommandLine, func(g *flag.Flag, usage string) bool {
			if strings.Contains(usage, "deprecated") {
				return true
			}
			if _, ok := flagIgnoreName.Load(g.Name); ok {
				return true
			}
			return false
		})
	}
}

var flagIgnoreName sync.Map

func AddIgnoreFlagName(names ...string) {
	for _, name := range names {
		flagIgnoreName.Store(name, true)
	}
}

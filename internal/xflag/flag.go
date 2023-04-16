// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/16

package xflag

import (
	"flag"
	"fmt"
	"strings"
)

func PrintDefaults(f *flag.FlagSet, ig func(g *flag.Flag, usage string) bool) {
	f.VisitAll(func(g *flag.Flag) {
		var b strings.Builder
		fmt.Fprintf(&b, "  -%s", g.Name) // Two spaces before -; see next two comments.
		name, usage := flag.UnquoteUsage(g)
		if len(name) > 0 {
			b.WriteString(" ")
			b.WriteString(name)
		}
		// Boolean flags of one ASCII letter are so common we
		// treat them specially, putting their usage on the same line.
		if b.Len() <= 4 { // space, space, '-', 'x'.
			b.WriteString("\t")
		} else {
			// Four spaces before the tab triggers good alignment
			// for both 4- and 8-space tab stops.
			b.WriteString("\n    \t")
		}
		b.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))
		if g.DefValue != "" {
			fmt.Fprintf(&b, " (default %v)", g.DefValue)
		}
		if ig(g, b.String()) {
			return
		}
		fmt.Fprint(f.Output(), b.String(), "\n")
	})
}

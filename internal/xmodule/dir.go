// Copyright(C) 2024 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2024/4/17

package xmodule

import (
	"bytes"
	"strings"
)

func CacheDirName(fp string) string {
	var bf bytes.Buffer
	for i := 0; i < len(fp); i++ {
		c := string(fp[i])
		lc := strings.ToLower(c)
		if c != lc {
			bf.WriteString("!")
		}
		bf.WriteString(lc)
	}
	return bf.String()
}

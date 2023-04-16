// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/4/16

package zpass

import "strings"

func IsTestPkg(pkg string) bool {
	return strings.HasSuffix(pkg, ".test") || strings.HasSuffix(pkg, "_test")
}

func PkgPath(name string) string {
	after, _ := strings.CutPrefix(name, "vendor/")
	return after
}

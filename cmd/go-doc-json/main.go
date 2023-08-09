// Copyright(C) 2023 github.com/fsgo  All Rights Reserved.
// Author: hidu <duv123@gmail.com>
// Date: 2023/7/27

package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"log"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/singlechecker"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/fsgo/gocode/internal/asthelper"
	"github.com/fsgo/gocode/zpass"
)

func main() {
	zpass.AddIgnoreFlagName("fix", "trace", "json")
	singlechecker.Main(Analyzer)
}

const Doc = `go doc
with flag "-debug v" for verbose
`

var container = &zpass.Container{}

var Analyzer = &analysis.Analyzer{
	Name: "zpass_go_doc_json",
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
		(*ast.FuncDecl)(nil),
		(*ast.ValueSpec)(nil),
		(*ast.GenDecl)(nil),
		(*ast.TypeSpec)(nil),
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
		doNode(pass, node)
	})
	return nil, nil
}

func checkIgnore(pass *analysis.Pass, nf *ast.File) bool {
	tokenFile := pass.Fset.File(nf.Pos())

	if !asthelper.IsGoFile(tokenFile) {
		return true
	}

	return asthelper.IsGoTestFile(tokenFile)
}

func doNode(pass *analysis.Pass, node ast.Node) {
	defer func() {
		if re := recover(); re != nil {
			asthelper.RecoverFatal(pass, node, re)
		}
	}()
	switch vt := node.(type) {
	case *ast.File:
		doAstFile(pass, vt)
	case *ast.FuncDecl:
		doFuncDecl(pass, vt)
	case *ast.ValueSpec:
		doValueSpec(pass, vt)
	case *ast.GenDecl:
		doGenDecl(pass, vt)
	case *ast.TypeSpec:
		doTypeSpec(pass, vt)
	default:
		panic(fmt.Errorf("not support %T", node))
	}
}

func doAstFile(pass *analysis.Pass, node *ast.File) {
	if node.Doc == nil {
		return
	}
	doc := newDocLine(pass)
	doc.Type = "file"
	doc.AddUsage(node.Doc.Text())
	doc.Print()
}

func paramName(pass *analysis.Pass, msgType string, field ast.Expr) string {
	pass.TypesInfo.TypeOf(field)
	switch pt := field.(type) {
	case *ast.Ident:
		// func(id string)
		return pt.Name
	case *ast.SelectorExpr:
		// func(addr net.Addr)
		return selectorExprFullName(pt)
	case *ast.StarExpr:
		return "*" + paramName(pass, msgType, pt.X)
	case *ast.Ellipsis:
		// func(its ....XXX)
		return "... " + paramName(pass, msgType, pt.Elt)
	case *ast.ArrayType:
		// func(its ....XXX)
		return "[] " + paramName(pass, msgType, pt.Elt)
	case *ast.FuncType:
		// fn func()
		return "func" // todo 改进输出格式
	case *ast.ChanType:
		return "chan" // todo 改进输出格式
	case *ast.MapType:
		return "map" // todo 改进输出格式
	case *ast.IndexExpr:
		// abc atomic.Pointer[config]
		return receiverName(pass, field)
	case *ast.InterfaceType:
		// func do(value interface{})  --> value 的类型
		return "any"
	case *ast.StructType:
		// runtime.MemStats  的 BySize 字段
		// type MemStats struct{
		// 	BySize [61]struct {
		//		// Size is the maximum byte size of an object in this
		//   }
		// }
		return "struct"
	default:
		panic(fmt.Sprintf(msgType+": not support %T", pt))
	}
}

// 用于返回接收定义的名称
// 如 func (f *Query) StringToIntVar()
// 会返回 Query.StringToIntVar
func receiverName(pass *analysis.Pass, node ast.Expr) string {
	var name string
	switch st := node.(type) {
	case *ast.StarExpr:
		return "*" + receiverName(pass, st.X)
	case *ast.Ident:
		// func (User) APIName
		return st.Name
	case *ast.IndexExpr:
		//  func (os objects[T]) MarshalLogArray(arr net.Addr) error
		//  struct 的字段：p atomic.Pointer[T]
		switch xvx := st.X.(type) {
		default:
			panic(fmt.Sprintf("not support %T", xvx))
		case *ast.Ident:
			name = xvx.Name
		case *ast.SelectorExpr:
			name = selectorExprFullName(xvx)
		}

		switch xvx := st.Index.(type) {
		default:
			panic(fmt.Sprintf("not support %T", xvx))
		case *ast.Ident:
			name += "[" + xvx.Name + "]"
		case *ast.IndexExpr:
			name += receiverName(pass, xvx)
		}
	case *ast.IndexListExpr:
		// func (os objectValues[T, P]) MarshalLogArray(arr net.Addr) error
		switch xvx := st.X.(type) {
		default:
			panic(fmt.Sprintf("not support %T", xvx))
		case *ast.Ident:
			name = xvx.Name
		}
		var tpNames []string
		for _, exp := range st.Indices {
			switch xvx := exp.(type) {
			default:
				panic(fmt.Sprintf("not support %T", xvx))
			case *ast.Ident:
				tpNames = append(tpNames, xvx.Name)
			}
		}
		name += "[" + strings.Join(tpNames, ",") + "]"

	default:
		panic(fmt.Sprintf("not support: %T", st))
	}
	return name
}

func doFuncDecl(pass *analysis.Pass, node *ast.FuncDecl) {
	doc := newDocLine(pass)
	doc.Type = "func"

	if node.Doc != nil {
		doc.AddUsage(node.Doc.Text())
	}

	if node.Recv != nil {
		doc.Type = "method"
		doc.Name = receiverName(pass, node.Recv.List[0].Type)
		doc.Name += "." + node.Name.Name
	} else {
		doc.Name = node.Name.Name
	}

	if doc.IsPrivate() {
		return
	}

	for _, p := range node.Type.Params.List {
		doc.Params = append(doc.Params, paramName(pass, "Params", p.Type))
	}

	if node.Type.Results != nil {
		for _, p := range node.Type.Results.List {
			doc.Results = append(doc.Results, paramName(pass, "Results", p.Type))
		}
	}
	doc.Print()
}

func selectorExprFullName(n *ast.SelectorExpr) string {
	x := n.X.(*ast.Ident)
	name := x.Name + "." + n.Sel.Name
	// todo add pkg path
	return name
}

func doValueSpec(pass *analysis.Pass, node *ast.ValueSpec) {}

func doGenDecl(pass *analysis.Pass, node *ast.GenDecl) {}

func doTypeSpec(pass *analysis.Pass, node *ast.TypeSpec) {
	doc := newDocLine(pass)
	doc.Name = node.Name.Name
	if doc.IsPrivate() {
		return
	}
	if node.Doc != nil {
		doc.AddUsage(node.Doc.Text())
	}
	if node.Comment != nil {
		doc.AddUsage(node.Comment.Text())
	}

	if node.TypeParams != nil {
		var tpNames []string
		for _, f := range node.TypeParams.List {
			for _, n := range f.Names {
				tpNames = append(tpNames, n.Name)
			}
		}
		doc.Name += "[" + strings.Join(tpNames, ",") + "]"
	}

	defer doc.Print()
	switch vt := node.Type.(type) {
	case *ast.StructType:
		doc.Type = "struct"
		for _, f := range vt.Fields.List {
			attr := Attr{}
			if f.Doc != nil {
				attr.AddUsage(f.Doc.Text())
			}
			if f.Comment != nil {
				attr.AddUsage(f.Comment.Text())
			}
			for _, name := range f.Names {
				if !ast.IsExported(name.Name) {
					continue
				}
				if attr.Name != "" {
					attr.Name += "."
				}
				attr.Name += name.Name
			}
			// 当包含一个 struct 的时候，是没有 Names 的
			if len(f.Names) > 0 && !ast.IsExported(attr.Name) {
				continue
			}

			attr.Type = paramName(pass, "struct Fields", f.Type)
			doc.Attrs = append(doc.Attrs, attr)
		}
	case *ast.Ident:
		// type MyType int
		doc.Type = "type"
	case *ast.InterfaceType:
		// type ABC interface{}
		doc.Type = "interface"
	case *ast.FuncType:
		// type myFunc func(xxx int)
		doc.Type = "func"
	case *ast.MapType:
		// type myType map[int]int
		doc.Type = "map"
	case *ast.ArrayType:
		doc.Type = "array"
	// type List []StartShutdown
	case *ast.SelectorExpr:
		// type Condition = xmatcher.Config
		doc.Type = "type"
	case *ast.ChanType:
		doc.Type = "chan"
	case *ast.StarExpr:
		doc.Type = "*" + paramName(pass, "doTypeSpec", vt.X)
	default:
		panic(fmt.Sprintf("not support: %T", vt))
	}
}

type Attr struct {
	Name  string `json:",omitempty"` // 属性名称。若是被包含，值为空
	Type  string // 属性类型
	Usage string `json:",omitempty"` // 使用文档
}

func (d *Attr) AddUsage(txt string) {
	if d.Usage != "" {
		d.Usage += "\n"
	}
	d.Usage += strings.TrimSpace(txt)
}

func newDocLine(pass *analysis.Pass) *DocLine {
	return &DocLine{
		Path: pass.Pkg.Path(),
	}
}

type DocLine struct {
	Name    string   // 名称，如 os，ral.RAL
	Path    string   // 所在包名，如 net, icode.baidu.com/baidu/gdp/net/ral
	Type    string   // 数据类型
	Usage   string   `json:",omitempty"` // 使用文档
	Attrs   []Attr   `json:",omitempty"` // 该类型包含哪几个公共属性
	Params  []string `json:",omitempty"` // 方法的入参类型
	Results []string `json:",omitempty"` // 方法的返回值类型
}

func (d *DocLine) AddUsage(txt string) {
	if d.Usage != "" {
		d.Usage += "\n"
	}
	d.Usage += strings.TrimSpace(txt)
}

// IsPrivate 使用 Name 字段，判断是否私有的
func (d *DocLine) IsPrivate() bool {
	if !ast.IsExported(d.Name) {
		return true
	}
	arr := strings.Split(d.Name, ".")
	for _, v := range arr {
		if !ast.IsExported(v) {
			return true
		}
	}
	return false
}

func (d *DocLine) String() string {
	bf, _ := json.Marshal(d)
	return string(bf)
}

func (d *DocLine) Print() {
	fmt.Println(d.String())
}

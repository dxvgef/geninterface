package pkg

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// 为指定结构体生成 Getter/Setter，并生成接口
// 不为嵌套结构体生成整体 Getter/Setter，仅为嵌套字段生成
// 生成的 getter.go 和 setter.go 文件名前缀为源文件名
func Generator(filePath string, structName string, genSetter bool) error {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return err
	}

	// 获取源文件名（不带扩展名）作为生成文件前缀
	fileBase := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	dir := filepath.Dir(filePath)

	// 收集所有具名结构体
	structMap := map[string]*ast.StructType{}
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if st, ok := typeSpec.Type.(*ast.StructType); ok {
				structMap[typeSpec.Name.Name] = st
			}
		}
	}

	rootStruct, ok := structMap[structName]
	if !ok {
		return fmt.Errorf("struct %s not found in %s", structName, filePath)
	}

	var getters, setters, interfaceMethods []string

	var generate func(prefix, accessPath string, st *ast.StructType)
	generate = func(prefix, accessPath string, st *ast.StructType) {
		for _, field := range st.Fields.List {
			isEmbedded := len(field.Names) == 0

			var names []string
			if isEmbedded { // 匿名结构体字段
				names = []string{exprString(field.Type)}
			} else {
				for _, n := range field.Names {
					if n.IsExported() {
						names = append(names, n.Name)
					}
				}
			}

			var inlineStruct *ast.StructType
			if stType, ok := field.Type.(*ast.StructType); ok {
				inlineStruct = stType
			}

			for _, name := range names {
				typ := exprString(field.Type)

				// 字段访问路径
				var fieldAccess string
				if isEmbedded {
					fieldAccess = name
				} else {
					if accessPath == "" {
						fieldAccess = name
					} else {
						fieldAccess = accessPath + "." + name
					}
				}

				namedStruct, isNamedStruct := typToStructMap(typ, structMap)
				isInlineStruct := inlineStruct != nil
				isStructField := isNamedStruct || isInlineStruct

				// ↓ 修复 gocritic: dupBranchBody — 统一前缀生成方式
				nextPrefix := prefix + name

				// Struct 字段不生成 Getter/Setter（包括匿名 struct），但递归展开字段
				if isStructField {
					var sub *ast.StructType
					if isNamedStruct {
						sub = namedStruct
					} else {
						sub = inlineStruct
					}

					generate(nextPrefix, fieldAccess, sub)
					continue
				}

				// 生成方法名
				methodName := nextPrefix

				// 生成 Getter
				getters = append(getters, fmt.Sprintf(
					`func (p *%s) Get%s() %s {
	return p.%s
}`, structName, methodName, typ, fieldAccess))

				interfaceMethods = append(interfaceMethods, fmt.Sprintf("Get%s() %s", methodName, typ))

				// 生成 Setter
				if genSetter {
					setters = append(setters, fmt.Sprintf(
						`func (p *%s) Set%s(v %s) {
	p.%s = v
}`, structName, methodName, typ, fieldAccess))

					interfaceMethods = append(interfaceMethods, fmt.Sprintf("Set%s(v %s)", methodName, typ))
				}
			}
		}
	}

	generate("", "", rootStruct)

	// ----- 写 getter.go -----
	getterFile := filepath.Join(dir, fileBase+"_getter.go")
	getterCode := fmt.Sprintf("package %s\n\n%s\n", f.Name.Name, strings.Join(getters, "\n\n"))
	if err = os.WriteFile(getterFile, []byte(getterCode), 0600); err != nil {
		return err
	}

	// ----- 写 setter.go -----
	setterFile := filepath.Join(dir, fileBase+"_setter.go")
	setterCode := ""
	if genSetter {
		setterCode = fmt.Sprintf("package %s\n\n%s\n", f.Name.Name, strings.Join(setters, "\n\n"))
	}
	if err := os.WriteFile(setterFile, []byte(setterCode), 0600); err != nil {
		return err
	}

	// ----- 写 interface.go -----
	interfaceFile := filepath.Join(dir, fileBase+"_interface.go")
	interfaceCode := fmt.Sprintf("package %s\n\ntype %sInterface interface {\n\t%s\n}\n",
		f.Name.Name, structName, strings.Join(interfaceMethods, "\n\t"))

	return os.WriteFile(interfaceFile, []byte(interfaceCode), 0600)
}

// typToStructMap 判断类型是否为 struct
func typToStructMap(typ string, structMap map[string]*ast.StructType) (*ast.StructType, bool) {
	base := strings.TrimPrefix(typ, "*")
	st, ok := structMap[base]
	return st, ok
}

// exprString 转换 AST 字段类型为字符串
func exprString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprString(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprString(t.Key), exprString(t.Value))
	default:
		return "interface{}"
	}
}

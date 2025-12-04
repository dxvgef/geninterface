package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Output struct {
	Getters         []string
	Setters         []string
	InterfaceMethod []string
}

// 生成器
func generator(filePath string, structNames []string, genSetter bool, filePerm os.FileMode, useAnyType bool, interfaceNameSuffix string, interfaceFileSuffix string, getterFileSuffix string, setterFileSuffix string) error {
	if filePerm == 0 {
		filePerm = 0600
	}

	parsed, structMap, err := parseGoFile(filePath)
	if err != nil {
		return err
	}

	targetStructs := resolveTargetStructs(structMap, structNames)
	if len(targetStructs) == 0 {
		return errors.New("no valid structs to process")
	}

	dir := filepath.Dir(filePath)

	for _, structName := range targetStructs {
		rootStruct := structMap[structName]
		output := buildAccessorsForStruct(structName, rootStruct, structMap, genSetter, useAnyType)

		if useAnyType {
			replaceInterfaceWithAny(&output)
		}

		files := buildFileNames(dir, structName, interfaceFileSuffix, getterFileSuffix, setterFileSuffix)
		if err := writeGeneratedFiles(files, parsed.PkgName, output, genSetter, filePerm, interfaceNameSuffix); err != nil {
			return err
		}
	}

	return nil
}

//
// ------------ 解析阶段 --------------
//

type ParsedFile struct {
	PkgName string
	Src     []byte
}

func parseGoFile(filePath string) (*ParsedFile, map[string]*ast.StructType, error) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, src, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

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

	return &ParsedFile{PkgName: f.Name.Name, Src: src}, structMap, nil
}

func resolveTargetStructs(structMap map[string]*ast.StructType, requested []string) []string {
	if len(requested) == 0 {
		all := make([]string, 0, len(structMap))
		for k := range structMap {
			all = append(all, k)
		}
		return all
	}

	valid := []string{}
	for _, name := range requested {
		if _, exists := structMap[name]; exists {
			valid = append(valid, name)
		}
	}
	return valid
}

//
// -------------- 生成 Getter/Setter --------------
//

func buildAccessorsForStruct(structName string, st *ast.StructType, structMap map[string]*ast.StructType, genSetter bool, useAnyType bool) Output {
	var result Output
	receiverName := strings.ToLower(structName[0:1])

	var generate func(prefix, accessPath string, st *ast.StructType)
	generate = func(prefix, accessPath string, st *ast.StructType) {
		for _, field := range st.Fields.List {
			isEmbedded := len(field.Names) == 0

			names := resolveFieldNames(field, isEmbedded, useAnyType)
			inlineStruct := inlineStructType(field)
			typ := exprString(field.Type, useAnyType)

			for _, name := range names {
				fieldAccess := buildAccessPath(name, accessPath, isEmbedded)
				nextPrefix := prefix + name
				if isEmbedded {
					nextPrefix = prefix
				}

				if isStructField(typ, inlineStruct, structMap) {
					sub := resolveSubStruct(typ, inlineStruct, structMap)
					generate(nextPrefix, fieldAccess, sub)
					continue
				}
				result.Getters = append(result.Getters, fmt.Sprintf(
					`func (%s *%s) Get%s() %s {
	return %s.%s
}`, receiverName, structName, nextPrefix, typ, receiverName, fieldAccess))

				result.InterfaceMethod = append(result.InterfaceMethod, fmt.Sprintf("Get%s() %s", nextPrefix, typ))

				if genSetter {
					result.Setters = append(result.Setters, fmt.Sprintf(
						`func (%s *%s) Set%s(v %s) {
	%s.%s = v
}`, receiverName, structName, nextPrefix, typ, receiverName, fieldAccess))

					result.InterfaceMethod = append(result.InterfaceMethod, fmt.Sprintf("Set%s(v %s)", nextPrefix, typ))
				}
			}
		}
	}

	generate("", "", st)
	return result
}

func resolveFieldNames(field *ast.Field, embedded bool, useAnyType bool) []string {
	if embedded {
		return []string{exprString(field.Type, useAnyType)}
	}
	var names []string
	for _, n := range field.Names {
		if n.IsExported() {
			names = append(names, n.Name)
		}
	}
	return names
}

func buildAccessPath(name, access string, embedded bool) string {
	// 匿名嵌入（embedded）
	if embedded {
		// 如果父访问路径为空，表示该嵌入在顶层：提升字段为顶层字段（不加前缀）
		if access == "" {
			return ""
		}
		// 如果父访问路径非空，表示嵌入在一个命名字段内部：使用父访问路径作为基准
		return access
	}

	// 普通字段按原逻辑拼接访问路径
	if access == "" {
		return name
	}
	return access + "." + name
}

//
// -------------- struct 类型识别 --------------
//

func inlineStructType(field *ast.Field) *ast.StructType {
	if st, ok := field.Type.(*ast.StructType); ok {
		return st
	}
	return nil
}

func resolveSubStruct(typ string, inline *ast.StructType, structMap map[string]*ast.StructType) *ast.StructType {
	if inline != nil {
		return inline
	}
	s, _ := typToStructMap(typ, structMap)
	return s
}

func isStructField(typeName string, inline *ast.StructType, structMap map[string]*ast.StructType) bool {
	_, ok := typToStructMap(typeName, structMap)
	return ok || inline != nil
}

//
// -------------- 类型替换 --------------

func replaceInterfaceWithAny(o *Output) {
	replace := func(list []string) []string {
		for i := range list {
			list[i] = strings.ReplaceAll(list[i], "interface{}", "any")
		}
		return list
	}

	o.Getters = replace(o.Getters)
	o.Setters = replace(o.Setters)
	o.InterfaceMethod = replace(o.InterfaceMethod)
}

//
// -------------- 写文件阶段 --------------
//

type FileSet struct {
	Getter    string
	Setter    string
	Interface string
	TypeName  string
}

func buildFileNames(dir, name string, interfaceFileSuffix string, getterFileSuffix string, setterFileSuffix string) FileSet {
	filePrefix := toSnakeCase(name)
	interfaceFile := toSnakeCase(interfaceFileSuffix)
	getterFile := toSnakeCase(getterFileSuffix)
	setterFile := toSnakeCase(setterFileSuffix)

	return FileSet{
		Getter:    filepath.Join(dir, filePrefix+getterFile+".go"),
		Setter:    filepath.Join(dir, filePrefix+setterFile+".go"),
		Interface: filepath.Join(dir, filePrefix+interfaceFile+".go"),
		TypeName:  name,
	}
}

func writeGeneratedFiles(files FileSet, pkg string, out Output, genSetter bool, perm os.FileMode, interfaceNameSuffix string) error {
	if err := os.WriteFile(files.Getter, []byte(formatFile(pkg, out.Getters)), perm); err != nil {
		return err
	}

	if genSetter {
		if err := os.WriteFile(files.Setter, []byte(formatFile(pkg, out.Setters)), perm); err != nil {
			return err
		}
	}

	interfaceName := files.TypeName + interfaceNameSuffix
	interfaceCode := formatInterface(pkg, interfaceName, out.InterfaceMethod)

	return os.WriteFile(files.Interface, []byte(interfaceCode), perm)
}

func formatFile(pkg string, body []string) string {
	return fmt.Sprintf("package %s\n\n%s\n", pkg, strings.Join(body, "\n\n"))
}

func formatInterface(pkg, structName string, methods []string) string {
	return fmt.Sprintf("package %s\n\ntype %s interface {\n\t%s\n}\n", pkg, structName, strings.Join(methods, "\n\t"))
}

//
// -------------- AST 类型转字符串 --------------

func typToStructMap(typ string, structMap map[string]*ast.StructType) (*ast.StructType, bool) {
	base := strings.TrimPrefix(typ, "*")
	st, ok := structMap[base]
	return st, ok
}

func exprString(expr ast.Expr, useAnyType bool) string {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Name == "interface{}" && useAnyType {
			return "any"
		}
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X, useAnyType)
	case *ast.SelectorExpr:
		return exprString(t.X, useAnyType) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprString(t.Elt, useAnyType)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprString(t.Key, useAnyType), exprString(t.Value, useAnyType))
	default:
		if useAnyType {
			return "any"
		}
		return "interface{}"
	}
}

func toSnakeCase(s string) string {
	// 预分配长度，最坏情况下每个字符前插入一个 '_'
	result := make([]rune, 0, len(s)*2)

	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

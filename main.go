package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var Version = "dev"

func main() {
	var perm os.FileMode

	showVersion := flag.Bool("version", false, "Show version")

	filePath := flag.String("file", "", "target file (file or directory)")
	structName := flag.String("struct", "", "struct name, use commas to separate multiple names")
	genSetter := flag.Bool("setter", false, "generate setter method")
	filePerm := flag.String("perm", "0600", "generated file permission")
	useAnyType := flag.Bool("any", true, "use the any type instead of the interface{} type")
	getterFileSuffix := flag.String("getter_file_suffix", "_getter", "generated getter file suffix")
	setterFileSuffix := flag.String("setter_file_suffix", "_setter", "generated setter file suffix")
	interfaceNameSuffix := flag.String("interface_name_suffix", "Interface", "generated interface name suffix")
	interfaceFileSuffix := flag.String("interface_file_suffix", "_interface", "generated interface file suffix")
	flag.Parse()

	if *showVersion {
		fmt.Println(Version)
		return
	}

	if *filePath == "" {
		fmt.Println("error: please provide a Go file path using -filePath")
		os.Exit(1)
	}

	files, err := collectGoFiles(*filePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if perm, err = parseFilePerm(*filePerm); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	structs := strings.Split(strings.Join(strings.Fields(*structName), ""), ",")

	for k := range files {
		fmt.Println("generating:", files[k], "for", structs)
		err = generator(
			files[k],
			structs,
			*genSetter,
			perm,
			*useAnyType,
			*interfaceNameSuffix,
			*interfaceFileSuffix,
			*getterFileSuffix,
			*setterFileSuffix,
		)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	fmt.Println("done")
}

// 收集所有的go文件，包含子目录
func collectGoFiles(filePath string) ([]string, error) {
	// 先清理路径并转换为绝对路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法解析路径: %w", err)
	}

	// 获取文件或目录信息
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("路径不存在或无法访问: %w", err)
	}

	// 情况一：输入为文件
	if info.Mode().IsRegular() {
		// 判断是否为 .go 文件
		if strings.HasSuffix(info.Name(), ".go") {
			return []string{absPath}, nil
		}
		return nil, errors.New("提供的文件不是 .go 文件")
	}

	// 情况二：输入为目录
	if info.IsDir() {
		var goFiles []string

		// 递归扫描目录
		err = filepath.Walk(absPath, func(path string, fi os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			// 跳过目录，只处理文件
			if fi.Mode().IsRegular() && strings.HasSuffix(fi.Name(), ".go") {
				goFiles = append(goFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("扫描目录失败: %w", err)
		}

		if len(goFiles) == 0 {
			return nil, errors.New("目录内未找到任何 .go 文件")
		}
		return goFiles, nil
	}

	return nil, errors.New("提供的路径既不是文件也不是目录")
}
func parseFilePerm(s string) (os.FileMode, error) {
	// 清理可能的 "0o644" 或 "0x" 风格前缀（可选）
	s = strings.TrimPrefix(s, "0o") // go-like octal
	s = strings.TrimPrefix(s, "0")  // allow input like "644"

	val, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid permission %q: must be octal (e.g., 644, 600, 755)", s)
	}

	if val > 0777 {
		return 0, fmt.Errorf("permission %q exceeds valid range (0000–0777)", s)
	}

	return os.FileMode(val), nil
}

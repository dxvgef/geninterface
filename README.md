# geninterface

This is a command-line tool that generates interfaces as well as Getter and Setter methods for structs in Go source files. By using the generated Getter methods for read-only access to the struct, it prevents direct reassignment of struct fields (thereby protecting field immutability after object construction).

## Usage:

```shell
geninterface -file=generator_test.go -struct=Config -setter=true
```

## Parameters:

- file: the Go source file for which the code will be generated, can be a file or directory
- struct: the name of the struct to generate code for, use commas to separate multiple names
- setter: whether to generate Setter methods. When set to false, only Getter methods will be generated
- perm: generated file permission
- any: use the any type instead of the interface{} type
- getter_file_suffix: generated getter file suffix
- setter_file_suffix: generated setter file suffix
- interface_name_suffix: generated interface name suffix
- interface_file_suffix: generated interface file suffix

## Generated files：

The following files will be created in the same directory as the target file:

- <struct><interface_file_suffix>.go
- <struct><getter_file_suffix>.go
- <struct><setter_file_suffix>.go

## Note:

If a large number of structs implement interfaces with many methods (typically 20 or more), compilation speed may decrease and the size of the resulting binary will increase.

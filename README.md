# geninterface

This is a command-line tool that generates interfaces as well as Getter and Setter methods for structs in Go source files. By using the generated Getter methods for read-only access to the struct, it prevents direct reassignment of struct fields (thereby protecting field immutability after object construction).

## Usage:

```shell
geninterface -file=main.go -struct=Config -setter=true
```

## Parameters:

- file: The Go source file for which the code will be generated
- struct: The name of the struct to generate code for
- setter: Whether to generate Setter methods. When set to false, only Getter methods will be generated

## Generated files：

The following files will be created in the same directory as the target file:

- {file}_interface.go
- {file}_getter.go
- {file}_setter.go

{file} is the name of the target file

## Note:

If a large number of structs implement interfaces with many methods (typically 20 or more), compilation speed may decrease and the size of the resulting binary will increase.

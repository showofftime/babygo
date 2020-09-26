package main

var sourceFiles = []string{
	"runtime/runtime.go",
	"/dev/stdin",
}

func main() {

	universe := createUniverse()

	for _, sourceFile := range sourceFiles {

		fmtPrintf("# file: %s\n", sourceFile)

		pkg = new(PkgContainer)
		stringIndex = 0
		stringLiterals = nil

		var f = parseFile(sourceFile)
		resolveUniverse(f, universe)
		pkg.name = f.Name

		walk(pkg, f)
		generateCode(pkg)
	}
}

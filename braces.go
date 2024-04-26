package braces

import (
	"github.com/pierre-primary/go-braces/syntax"
)

func Compile(pattern string) (*Walker, error) {
	ast, buffer := syntax.Parse(pattern, nil)
	return &Walker{ast, buffer, pattern}, nil
}

func MustCompile(pattern string) *Walker {
	w, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return w
}

func PrintTree(input string) {
	ast, _ := syntax.Parse(input, nil)
	ast.Print()
}

func Walk(input string, callback WalkHandler) {
	ast, buffer := syntax.Parse(input, nil)
	ast.Walk(callback, buffer)
}

func Expand(input string) []string {
	ast, buffer := syntax.Parse(input, nil)
	data, _ := ast.Expand(nil, buffer)
	return data
}

func AppendExpand(data []string, input string) []string {
	ast, buffer := syntax.Parse(input, nil)
	data, _ = ast.Expand(data, buffer)
	return data
}

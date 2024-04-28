package braces

import (
	"github.com/pierre-primary/go-braces/syntax"
)

func Compile(pattern string) (*Walker, error) {
	ast, buffer, err := syntax.Parse(pattern, nil, syntax.StrictMode)
	if err != nil {
		return nil, err
	}
	return &Walker{ast, buffer, pattern}, nil
}

func MustCompile(pattern string) *Walker {
	w, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return w
}

func Walk(input string, callback WalkHandler) {
	ast, buffer, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	ast.Walk(callback, buffer)
}

func Expand(input string) []string {
	ast, buffer, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	data, _ := ast.Expand(nil, buffer)
	return data
}

func AppendExpand(data []string, input string) []string {
	ast, buffer, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	data, _ = ast.Expand(data, buffer)
	return data
}

func PrintTree(input string) {
	ast, _, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	ast.Print()
}

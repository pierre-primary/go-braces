package braces

import (
	"github.com/pierre-primary/go-braces/syntax"
)

type WalkHandler = syntax.WalkHandler

func Walk(input string, handler WalkHandler) {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	exp.Walk(handler)
}

func Expand(input string) []string {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	return exp.Expand(nil)
}

func AppendExpand(data []string, input string) []string {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	return exp.Expand(data)
}

func PrintTree(input string) {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	exp.Print()
}

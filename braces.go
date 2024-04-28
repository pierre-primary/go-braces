package braces

import (
	"github.com/pierre-primary/go-braces/syntax"
)

type WalkHandler = syntax.WalkHandler

func Walk(input string, handler WalkHandler, flags ...syntax.ExpandFlags) {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	exp.Walk(handler, nil, flags...)
}

func Expand(input string, flags ...syntax.ExpandFlags) []string {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	return exp.Expand(nil, flags...)
}

func AppendExpand(data []string, input string, flags ...syntax.ExpandFlags) []string {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	return exp.Expand(data, flags...)
}

func PrintTree(input string) {
	exp, err := syntax.Parse(input)
	if err != nil {
		panic(err)
	}
	exp.Print()
}

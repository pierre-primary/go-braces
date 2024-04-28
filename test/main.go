package main

import (
	"fmt"

	"github.com/pierre-primary/go-braces/syntax"
)

func main() {
	pattern := `abc\"def`

	exp, buffer, _ := syntax.Parse(pattern, nil, syntax.IgnoreEscape)

	exp.Walk(func(str string) {
		fmt.Println(str)
	}, buffer, syntax.KeepEscape)

	// braces.Walk(pattern, func(str string) {
	// 	fmt.Println(str)
	// })

	// braces.PrintTree(pattern)
}

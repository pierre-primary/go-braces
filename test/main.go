package main

import (
	"fmt"

	"github.com/pierre-primary/go-braces"
)

func main() {
	pattern := `a{a{a,b}b`

	braces.Walk(pattern, func(str string) {
		fmt.Println(str)
	})

	braces.PrintTree(pattern)
}

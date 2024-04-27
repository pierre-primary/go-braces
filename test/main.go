package main

import (
	"fmt"

	"github.com/pierre-primary/go-braces"
)

func main() {
	pattern := `{1..9}`

	braces.Walk(pattern, func(str string) {
		fmt.Println(str)
	})

	braces.PrintTree(pattern)
}

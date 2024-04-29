package main

import (
	"fmt"

	"github.com/pierre-primary/go-braces"
)

func main() {
	input := `abc{a,b}def`

	braces.Walk(input, func(str string) {
		fmt.Println(str)
	})

	// regexp.MustCompile("")
}

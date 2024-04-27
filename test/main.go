package main

import (
	"fmt"

	"github.com/pierre-primary/go-braces"
)

func main() {
	pattern := `{{a..z..2}\,abc}`

	braces.Walk(pattern, func(str string) {
		fmt.Println(str)
	})

	braces.PrintTree(pattern)

	// fmt.Println(braces.MustCompile("{a..z..2}").Equal(braces.MustCompile("{a..y..2}")))
}

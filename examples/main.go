package main

import (
	"fmt"

	"github.com/pierre-primary/go-braces"
)

func main() {
	pattern := `你{好,在{那,学校,公司,家}}{,吗}`

	braces.Walk(pattern, func(str string) {
		fmt.Println(str)
	})

	braces.PrintTree(pattern)
}

package braces

import (
	"github.com/pierre-primary/go-braces/syntax"
)

type WalkHandler = syntax.WalkHandler

type BraceExp struct {
	exp    *syntax.BraceExp
	buffer []byte
	src    string
}

func (b *BraceExp) String() string {
	return b.src
}

func (b *BraceExp) Equal(t *BraceExp) bool {
	return b.exp.Equal(t.exp)
}

func (b *BraceExp) Walk(callback WalkHandler) {
	b.buffer = b.exp.Walk(callback, b.buffer)
}

func (b *BraceExp) Expand(data []string) []string {
	data, b.buffer = b.exp.Expand(data, b.buffer)
	return data
}

func (b *BraceExp) Print() {
	b.exp.Print()
}

func Compile(pattern string) (*BraceExp, error) {
	exp, buffer, err := syntax.Parse(pattern, nil, syntax.StrictMode)
	if err != nil {
		return nil, err
	}
	return &BraceExp{exp, buffer, pattern}, nil
}

func MustCompile(pattern string) *BraceExp {
	w, err := Compile(pattern)
	if err != nil {
		panic(err)
	}
	return w
}

func Walk(input string, callback WalkHandler) {
	exp, buffer, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	exp.Walk(callback, buffer)
}

func Expand(input string) []string {
	exp, buffer, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	data, _ := exp.Expand(nil, buffer)
	return data
}

func AppendExpand(data []string, input string) []string {
	exp, buffer, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	data, _ = exp.Expand(data, buffer)
	return data
}

func PrintTree(input string) {
	exp, _, err := syntax.Parse(input, nil)
	if err != nil {
		panic(err)
	}
	exp.Print()
}

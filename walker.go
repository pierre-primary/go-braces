package braces

import (
	"github.com/pierre-primary/go-braces/syntax"
)

type WalkHandler = syntax.WalkHandler

type Walker struct {
	ast    *syntax.Node
	buffer []byte
	src    string
}

func (w *Walker) String() string {
	return w.src
}

func (w *Walker) Equal(t *Walker) bool {
	return w.ast.Equal(t.ast)
}

func (w *Walker) Walk(callback WalkHandler) {
	w.buffer = w.ast.Walk(callback, w.buffer)
}

func (w *Walker) Expand(data []string) []string {
	data, w.buffer = w.ast.Expand(data, w.buffer)
	return data
}

func (w *Walker) Print() {
	w.ast.Print()
}

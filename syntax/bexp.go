package syntax

import (
	"fmt"
	"unsafe"
)

type Op int

const (
	OpUnknown Op = iota
	OpEmpty
	OpLiteral
	OpEscape
	OpQuote
	OpIntegerRange
	OpCharRange
	OpConcat
	OpAlternate
)

func (op Op) String() string {
	switch op {
	case OpEmpty:
		return "Empty"
	case OpLiteral:
		return "Literal"
	case OpEscape:
		return "Escape"
	case OpQuote:
		return "Quote"
	case OpIntegerRange:
		return "IntegerRange"
	case OpCharRange:
		return "CharRange"
	case OpConcat:
		return "Concat"
	case OpAlternate:
		return "Alternate"
	default:
		return "Unknown"
	}
}

type BraceExp struct {
	Op   Op
	Subs []*BraceExp
	Next *BraceExp
	Val  []byte
	Val0 [2]byte
}

const opPseudo Op = 128 // where pseudo-ops start

func (n *BraceExp) link(next *BraceExp) {
	switch n.Op {
	case OpAlternate:
		for _, item := range n.Subs {
			item.link(next)
		}
	case OpConcat:
		if subs := n.Subs; len(subs) > 0 {
			subs[len(subs)-1].link(next)
		}
		fallthrough
	default:
		n.Next = next
	}
}

func printExp(exp *BraceExp, deepth int) {
	switch exp.Op {
	case OpCharRange:
		rg := unsafe.Slice((*int)(unsafe.Pointer(&exp.Val[0])), 4)
		sta, num, sep := rg[0], rg[1], rg[2]
		s := fmt.Sprintf("%c..%c", rune(sta), rune(sta+num*sep))
		if sep > 1 || sep < -1 {
			s = fmt.Sprintf("%s..%d", s, sep)
		}
		fmt.Printf("%*s - %s (\"%s\")\n", deepth<<1, "", exp.Op, s)
	case OpIntegerRange:
		rg := unsafe.Slice((*int)(unsafe.Pointer(&exp.Val[0])), 5)
		sta, num, sep, wid := rg[0], rg[1], rg[2], rg[3]
		s := fmt.Sprintf("%*d..%*d", wid, sta, wid, sta+num*sep)
		if sep > 1 || sep < -1 {
			s = fmt.Sprintf("%s..%d", s, sep)
		}
		fmt.Printf("%*s - %s (\"%s\")\n", deepth<<1, "", exp.Op, s)
	case OpConcat, OpAlternate:
		fmt.Printf("%*s - %s\n", deepth<<1, "", exp.Op)
		for _, item := range exp.Subs {
			printExp(item, deepth+1)
		}
	default:
		fmt.Printf("%*s - %s (%q)\n", deepth<<1, "", exp.Op, exp.Val)
	}
}

func (n *BraceExp) Print() {
	printExp(n, 0)
}

func (n *BraceExp) Equal(t *BraceExp) bool {
	if n == nil || t == nil {
		return n == t
	}
	if n.Op != t.Op {
		return false
	}
	switch n.Op {
	case OpLiteral, OpEscape, OpQuote, OpIntegerRange, OpCharRange:
		if len(n.Val) != len(t.Val) {
			return false
		}
		for i, r := range n.Val {
			if r != t.Val[i] {
				return false
			}
		}
	case OpConcat, OpAlternate:
		if len(n.Subs) != len(t.Subs) {
			return false
		}
		for i, item := range n.Subs {
			if !item.Equal(t.Subs[i]) {
				return false
			}
		}
	}
	return true
}

func (n *BraceExp) walk(handler WalkHandler, buffer []byte, flags ...ExpandFlags) []byte {
	var flag ExpandFlags
	for _, f := range flags {
		flag |= f
	}
	return walk(n, flag, handler, buffer)
}

func (n *BraceExp) Walk(handler WalkHandler, flags ...ExpandFlags) {
	n.walk(handler, nil, flags...)
}

func (n *BraceExp) Expand(data []string, flags ...ExpandFlags) []string {
	n.Walk(func(str string) { data = append(data, str) }, flags...)
	return data
}

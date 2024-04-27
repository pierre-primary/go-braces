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

type Node struct {
	Op   Op
	Subs []*Node
	Next *Node
	Val  []byte
	Val0 [2]byte
}

const opPseudo Op = 128 // where pseudo-ops start

func (n *Node) link(next *Node) {
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

func (n *Node) Equal(t *Node) bool {
	if n == nil || t == nil {
		return n == t
	}
	if n.Op != t.Op {
		return false
	}
	switch n.Op {
	case OpLiteral, OpIntegerRange, OpCharRange:
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

func (n *Node) Walk(callback WalkHandler, buffer []byte) []byte {
	return walk(buffer[:0], n, callback)
}

func (n *Node) Expand(data []string, buffer []byte) ([]string, []byte) {
	buffer = walk(buffer[:0], n, func(str string) {
		data = append(data, str)
	})
	return data, buffer
}

func printNode(node *Node, deepth int) {
	switch node.Op {
	case OpCharRange:
		rg := unsafe.Slice((*int)(unsafe.Pointer(&node.Val[0])), 4)
		sta, num, sep := rg[0], rg[1], rg[2]
		s := fmt.Sprintf("%c..%c", rune(sta), rune(sta+num*sep))
		if sep > 1 || sep < -1 {
			s = fmt.Sprintf("%s..%d", s, sep)
		}
		fmt.Printf("%*s - %s (\"%s\")\n", deepth<<1, "", node.Op, s)
	case OpIntegerRange:
		rg := unsafe.Slice((*int)(unsafe.Pointer(&node.Val[0])), 5)
		sta, num, sep, wid := rg[0], rg[1], rg[2], rg[3]
		s := fmt.Sprintf("%*d..%*d", wid, sta, wid, sta+num*sep)
		if sep > 1 || sep < -1 {
			s = fmt.Sprintf("%s..%d", s, sep)
		}
		fmt.Printf("%*s - %s (\"%s\")\n", deepth<<1, "", node.Op, s)
	case OpConcat, OpAlternate:
		fmt.Printf("%*s - %s\n", deepth<<1, "", node.Op)
		for _, item := range node.Subs {
			printNode(item, deepth+1)
		}
	default:
		fmt.Printf("%*s - %s (%q)\n", deepth<<1, "", node.Op, node.Val)
	}
}

func (n *Node) Print() {
	printNode(n, 0)
}

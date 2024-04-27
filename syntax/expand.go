package syntax

import (
	"unicode/utf8"
	"unsafe"
)

type WalkHandler func(str string)

var ZEROS = [8]byte{'0', '0', '0', '0', '0', '0', '0', '0'}

func appendZero(buf []byte, bits int) []byte {
	for bits > 8 {
		buf = append(buf, ZEROS[:8]...)
		bits -= 8
	}
	if bits > 0 {
		buf = append(buf, ZEROS[:bits]...)
	}
	return buf
}

func appendNumber(buf []byte, num int, align int) []byte {
	neg := false
	u := uint(num)
	if num < 0 {
		neg = true
		u = ^u + 1
	}

	var a [32]byte
	i := len(a)

	for u >= 10 {
		i--
		a[i] = byte('0' + u%10)
		u /= 10
	}
	i--
	a[i] = byte('0' + u)

	if neg {
		align--
		buf = append(buf, '-')
	}

	if align -= len(a[i:]); align > 0 {
		buf = appendZero(buf, align)
	}
	return append(buf, a[i:]...)
}

func walkAlternate(buf []byte, node *Node, cb WalkHandler) []byte {
	offset := len(buf)
	for _, item := range node.Subs {
		buf = walk(buf[:offset], item, cb)
	}
	return buf
}

func walkCharRange(buf []byte, node *Node, cb WalkHandler) []byte {
	rg := unsafe.Slice((*int)(unsafe.Pointer(&node.Val[0])), 4)
	sta, num, sep := rg[0], rg[1], rg[2]

	offset := len(buf)
	buf = walk(utf8.AppendRune(buf, rune(sta)), node.Next, cb)
	for i := 0; i < num; i++ {
		sta += sep
		buf = walk(utf8.AppendRune(buf[:offset], rune(sta)), node.Next, cb)
	}
	return buf
}

func walkIntegerRange(buf []byte, node *Node, cb WalkHandler) []byte {
	rg := unsafe.Slice((*int)(unsafe.Pointer(&node.Val[0])), 4)
	sta, num, sep, wid := rg[0], rg[1], rg[2], rg[3]

	offset := len(buf)
	buf = walk(appendNumber(buf, sta, wid), node.Next, cb)
	for i := 0; i < num; i++ {
		sta += sep
		buf = walk(appendNumber(buf[:offset], sta, wid), node.Next, cb)
	}
	return buf
}

func walk(buf []byte, node *Node, cb WalkHandler) []byte {
	if node == nil {
		cb(string(buf))
		return buf
	}
	switch node.Op {
	case OpConcat:
		return walk(buf, node.Subs[0], cb)
	case OpAlternate:
		return walkAlternate(buf, node, cb)
	case OpCharRange:
		return walkCharRange(buf, node, cb)
	case OpIntegerRange:
		return walkIntegerRange(buf, node, cb)
	default:
		return walk(append(buf, node.Val...), node.Next, cb)
	}
}

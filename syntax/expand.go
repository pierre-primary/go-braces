package syntax

import (
	"unicode/utf8"
	"unsafe"
)

type WalkHandler func(str string)

var ZEROS = [8]byte{'0', '0', '0', '0', '0', '0', '0', '0'}

type ExpandFlags uint16

const (
	KeepEscape ExpandFlags = 1 << iota
	KeepQuote
)

type walker struct {
	flags   ExpandFlags
	handler WalkHandler
}

func (w *walker) walkAlternate(exp *BraceExp, buffer []byte) []byte {
	offset := len(buffer)
	for _, item := range exp.Subs {
		buffer = w.walk(item, buffer[:offset])
	}
	return buffer
}

func (w *walker) walkCharRange(exp *BraceExp, buffer []byte) []byte {
	rg := unsafe.Slice((*int)(unsafe.Pointer(&exp.Val[0])), 4)
	sta, num, sep := rg[0], rg[1], rg[2]

	offset := len(buffer)
	buffer = w.walk(exp.Next, utf8.AppendRune(buffer, rune(sta)))
	for i := 0; i < num; i++ {
		sta += sep
		buffer = w.walk(exp.Next, utf8.AppendRune(buffer[:offset], rune(sta)))
	}
	return buffer
}

func (w *walker) walkIntegerRange(exp *BraceExp, buffer []byte) []byte {
	rg := unsafe.Slice((*int)(unsafe.Pointer(&exp.Val[0])), 4)
	sta, num, sep, wid := rg[0], rg[1], rg[2], rg[3]

	offset := len(buffer)
	buffer = w.walk(exp.Next, appendNumber(buffer, sta, wid))
	for i := 0; i < num; i++ {
		sta += sep
		buffer = w.walk(exp.Next, appendNumber(buffer[:offset], sta, wid))
	}
	return buffer
}

func (w *walker) walkEscape(exp *BraceExp, buffer []byte) []byte {
	if w.flags&KeepEscape == 0 {
		return w.walk(exp.Next, append(buffer, exp.Val[1:]...))
	}
	return w.walk(exp.Next, append(buffer, exp.Val...))
}

func (w *walker) walkQuote(exp *BraceExp, buffer []byte) []byte {
	if w.flags&KeepQuote == 0 {
		return w.walk(exp.Next, buffer)
	}
	return w.walk(exp.Next, append(buffer, exp.Val...))
}

func (w *walker) walk(exp *BraceExp, buffer []byte) []byte {
	if exp == nil {
		w.handler(string(buffer))
		return buffer
	}
	switch exp.Op {
	case OpConcat:
		return w.walk(exp.Subs[0], buffer)
	case OpAlternate:
		return w.walkAlternate(exp, buffer)
	case OpCharRange:
		return w.walkCharRange(exp, buffer)
	case OpIntegerRange:
		return w.walkIntegerRange(exp, buffer)
	case OpEscape:
		return w.walkEscape(exp, buffer)
	case OpQuote:
		return w.walkQuote(exp, buffer)
	case OpEmpty:
		return w.walk(exp.Next, buffer)
	default:
		return w.walk(exp.Next, append(buffer, exp.Val...))
	}
}

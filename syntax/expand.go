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

func walkAlternate(exp *BraceExp, flags ExpandFlags, handler WalkHandler, buffer []byte) []byte {
	offset := len(buffer)
	for _, item := range exp.Subs {
		buffer = walk(item, flags, handler, buffer[:offset])
	}
	return buffer
}

func walkCharRange(exp *BraceExp, flags ExpandFlags, handler WalkHandler, buffer []byte) []byte {
	rg := unsafe.Slice((*int)(unsafe.Pointer(&exp.Val[0])), 4)
	sta, num, sep := rg[0], rg[1], rg[2]

	offset := len(buffer)
	buffer = walk(exp.Next, flags, handler, utf8.AppendRune(buffer, rune(sta)))
	for i := 0; i < num; i++ {
		sta += sep
		buffer = walk(exp.Next, flags, handler, utf8.AppendRune(buffer[:offset], rune(sta)))
	}
	return buffer
}

func walkIntegerRange(exp *BraceExp, flags ExpandFlags, handler WalkHandler, buffer []byte) []byte {
	rg := unsafe.Slice((*int)(unsafe.Pointer(&exp.Val[0])), 4)
	sta, num, sep, wid := rg[0], rg[1], rg[2], rg[3]

	offset := len(buffer)
	buffer = walk(exp.Next, flags, handler, appendNumber(buffer, sta, wid))
	for i := 0; i < num; i++ {
		sta += sep
		buffer = walk(exp.Next, flags, handler, appendNumber(buffer[:offset], sta, wid))
	}
	return buffer
}

func walkEscape(exp *BraceExp, flags ExpandFlags, handler WalkHandler, buffer []byte) []byte {
	if flags&KeepEscape == 0 {
		return walk(exp.Next, flags, handler, append(buffer, exp.Val[1:]...))
	}
	return walk(exp.Next, flags, handler, append(buffer, exp.Val...))
}

func walkQuote(exp *BraceExp, flags ExpandFlags, handler WalkHandler, buffer []byte) []byte {
	if flags&KeepQuote == 0 {
		return walk(exp.Next, flags, handler, buffer)
	}
	return walk(exp.Next, flags, handler, append(buffer, exp.Val...))
}

func walk(exp *BraceExp, flags ExpandFlags, handler WalkHandler, buffer []byte) []byte {
	if exp == nil {
		handler(string(buffer))
		return buffer
	}
	switch exp.Op {
	case OpConcat:
		return walk(exp.Subs[0], flags, handler, buffer)
	case OpAlternate:
		return walkAlternate(exp, flags, handler, buffer)
	case OpCharRange:
		return walkCharRange(exp, flags, handler, buffer)
	case OpIntegerRange:
		return walkIntegerRange(exp, flags, handler, buffer)
	case OpEscape:
		return walkEscape(exp, flags, handler, buffer)
	case OpQuote:
		return walkQuote(exp, flags, handler, buffer)
	case OpEmpty:
		return walk(exp.Next, flags, handler, buffer)
	default:
		return walk(exp.Next, flags, handler, append(buffer, exp.Val...))
	}
}

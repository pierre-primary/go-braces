package syntax

import (
	"math"
	"unicode/utf8"
	"unsafe"
)

const (
	opBraceOpen Op = iota + opPseudo
	opBraceDelim
	opBraceRange
)

var (
	ErrInvalidUTF8       = "invalid UTF-8"
	ErrTrailingBackslash = "trailing backslash at end of expression"
	ErrMissingQuote      = "missing closing quote character"
)

type ParseFlags uint16

const (
	IgnoreEscape ParseFlags = 1 << iota
	IgnoreQuote
	AnyCharRange
	StrictMode
)

type Parser struct {
	flags ParseFlags
	stack []*BraceExp
	free  *BraceExp
}

func NewParser(flags ...ParseFlags) *Parser {
	var flag ParseFlags
	for _, f := range flags {
		flag |= f
	}
	return &Parser{flags: flag}
}

func Parse(input string, flags ...ParseFlags) (*BraceExp, error) {
	return NewParser(flags...).Parse(input)
}

func (p *Parser) Parse(input string) (*BraceExp, error) {
	exp, _, err := p.parse(input, nil)
	return exp, err
}

func (p *Parser) newExp(op Op) (exp *BraceExp) {
	exp = p.free
	if exp != nil {
		p.free = exp.Next
		*exp = BraceExp{}
	} else {
		exp = new(BraceExp)
	}
	exp.Op = op
	return exp
}

func (p *Parser) reuse(exp *BraceExp) {
	exp.Next = p.free
	p.free = exp
}

func (p *Parser) push(exp *BraceExp) {
	p.stack = append(p.stack, exp)
}

func (p *Parser) op(op Op, val string) {
	exp := p.newExp(op)
	exp.Val = append(exp.Val0[:0], val...)
	p.push(exp)
}

func (p *Parser) literal(val string) {
	if len(p.stack) > 0 {
		if exp := p.stack[len(p.stack)-1]; exp.Op == OpLiteral {
			exp.Val = append(exp.Val, val...)
			return
		}
	}
	p.op(OpLiteral, val)
}

func (p *Parser) flatten(subs []*BraceExp, op Op, set []*BraceExp) []*BraceExp {
	for _, exp := range set {
		if exp.Op < opPseudo {
			if exp.Op == op {
				subs = append(subs, exp.Subs...)
				p.reuse(exp)
			} else {
				subs = append(subs, exp)
			}
		}
	}
	return subs
}

func (p *Parser) concat(offset int) {
	if offset < 0 {
		offset = len(p.stack)
		for offset > 0 && p.stack[offset-1].Op < opPseudo {
			offset--
		}
	}

	switch len(p.stack) - offset {
	case 0:
		p.op(OpEmpty, "")
		fallthrough
	case 1:
		return
	}

	set := p.stack[offset:]
	p.stack = p.stack[:offset]

	exp := p.newExp(OpConcat)
	exp.Subs = p.flatten(exp.Subs, OpConcat, set)
	if subs := exp.Subs; len(subs) > 1 {
		last := subs[0]
		for _, sub := range subs[1:] {
			last.link(sub)
			last = sub
		}
	}
	p.push(exp)
}

func (p *Parser) alternate(offset int) {
	set := p.stack[offset:]
	p.stack = p.stack[:offset]

	// ASSERT: at least two sub-exps required
	exp := p.newExp(OpAlternate)
	exp.Subs = p.flatten(exp.Subs, OpAlternate, set)
	// TODO: optimize exp.Subs
	p.push(exp)
}

func (p *Parser) literalize(offset int, backward bool, buffer []byte) []byte {
	if backward && offset > 0 && p.stack[offset-1].Op == OpLiteral {
		offset--
	}
	var first *BraceExp
	for idx := offset; idx < len(p.stack); idx++ {
		item := p.stack[idx]
		if item.Op == OpLiteral || item.Op >= opPseudo {
			if first == nil {
				first, buffer = item, buffer[:0]
				first.Op = OpLiteral
			} else {
				buffer = append(buffer, item.Val...)
				p.reuse(item)
				continue
			}
		} else if first != nil {
			first.Val = append(first.Val, buffer...)
			first = nil
		}
		p.stack[offset] = item
		offset++
	}
	if first != nil {
		first.Val = append(first.Val, buffer...)
	}
	p.stack = p.stack[:offset]
	return buffer
}

func careateRangeData(sta, end, sep, wid int) (bool, []byte) {
	var num int
	if sta == end {
		sep = 0
		num = 0
	} else {
		sep_u := absToUint(sep)
		if sep_u > math.MaxInt {
			return false, nil
		} else if sep_u == 0 {
			sep_u = 1
		}

		delta := uint(0)
		if sta < end {
			sep = int(sep_u)
			delta = uint(end - sta)
		} else if sta > end {
			sep = int(-sep_u)
			delta = uint(sta - end)
		}

		div := delta / sep_u
		if div+1 > math.MaxInt { // elem limit
			return false, nil
		}
		if div == 0 {
			sep = 0
			num = 0
		} else {
			num = int(div)
		}
	}

	opts := [4]int{sta, int(num), sep, wid}
	return true, unsafe.Slice((*byte)(unsafe.Pointer(&opts[0])), int(unsafe.Sizeof(opts)))
}

func (p *Parser) ranges(offset int) (ok bool) {
	set := p.stack[offset:]
	sep := 0
	switch len(set) {
	default:
		return false
	case 6:
		if _ = set[5]; set[4].Op != opBraceRange || set[5].Op != OpLiteral {
			return false
		}
		if ok, sep = parseInt(set[5].Val); !ok {
			return false
		}
		fallthrough
	case 4:
		if _ = set[3]; set[2].Op != opBraceRange {
			return false
		}
	}

	var vs, ve []byte
	if set[1].Op == OpLiteral {
		vs = set[1].Val
	} else if set[1].Op == OpEscape {
		vs = set[1].Val[1:]
	} else {
		return false
	}

	if set[3].Op == OpLiteral {
		ve = set[3].Val
	} else if set[3].Op == OpEscape {
		ve = set[3].Val[1:]
	} else {
		return false
	}

	ls, le := len(vs), len(ve)

	op, wid := OpUnknown, 0
	var sta, end int
	if ls == 1 && le == 1 {
		cs, ce := vs[0], ve[0]
		if isDigit(cs) && isDigit(ce) {
			op, sta, end = OpIntegerRange, int(cs-'0'), int(ce-'0')
		} else if p.flags&AnyCharRange != 0 && cs < utf8.RuneSelf && ce < utf8.RuneSelf {
			op, sta, end = OpCharRange, int(cs), int(ce)
		} else if (isUpperCase(cs) && isUpperCase(ce)) || (isLowerCase(cs) && isLowerCase(ce)) {
			op, sta, end = OpCharRange, int(cs), int(ce)
		}
	}

	switch {
	case op == OpUnknown:
		var ok bool
		if ok, sta = parseInt(vs); !ok {
			break
		}
		if ok, end = parseInt(ve); !ok {
			break
		}

		op = OpIntegerRange
		if ls >= 2 && (vs[0] == '0' || (vs[0] == '-' && vs[1] == '0')) {
			wid = ls
		}
		if wid < le && le >= 2 && (ve[0] == '0' || (ve[0] == '-' && ve[1] == '0')) {
			wid = le
		}
	}

	switch {
	case p.flags&AnyCharRange != 0 && op == OpUnknown:
		rs, rw := utf8.DecodeRune(vs)
		if rs == utf8.RuneError || rw != ls {
			break
		}
		re, rw := utf8.DecodeRune(ve)
		if re == utf8.RuneError || rw != le {
			break
		}
		op, sta, end = OpCharRange, int(rs), int(re)
	}

	if op == OpUnknown {
		return false
	}

	ok, data := careateRangeData(sta, end, sep, wid)
	if !ok {
		return false
	}

	p.stack = p.stack[:offset]
	for _, exp := range set {
		p.reuse(exp)
	}

	exp := p.newExp(op)
	exp.Val = data
	p.push(exp)
	return true
}

func (p *Parser) parse(input string, buffer []byte) (*BraceExp, []byte, error) {
	type block struct {
		base   int // Base Stack Index
		ranges int
		delims int
	}
	blocks := make([]block, 0, 4)
	var blk *block

	p.stack = p.stack[:0]
	sta := -1

	submit := func(idx int) {
		if sta >= 0 {
			p.literal(input[sta:idx])
			sta = ^sta
		}
	}

	literalize := func(offset int, backward bool) {
		buffer = p.literalize(offset, backward, buffer)
	}

	var que, esc byte
	var queSta int

	for end := 0; ; end++ {
		goto Skip
	Regular:
		if sta < 0 {
			sta = end
		}

		if input[end] < utf8.RuneSelf {
			end++
		} else if c, w := utf8.DecodeRuneInString(input[end:]); c == utf8.RuneError && p.flags&StrictMode != 0 {
			return nil, buffer, &Error{ErrInvalidUTF8, end}
		} else if w > 0 {
			end += w
		} else {
			break
		}

		/** Escape **/
		if esc > 0 {
			esc = 0
			p.op(OpEscape, input[sta:end])
			sta = ^sta
		}
	Skip:
		if end >= len(input) {
			break
		}
		ch := input[end]

		/** In Quoted **/
		if que > 0 {
			if que != ch || input[end-1] == '\\' {
				goto Regular
			}

			submit(end)

			que = 0
			p.op(OpQuote, string(ch))
			continue
		}

		switch ch {
		default:
			goto Regular
		case '\\':
			/** Escape Character **/
			if p.flags&IgnoreEscape != 0 {
				goto Regular
			}
			submit(end)

			if end+1 < len(input) {
				esc = ch
				sta = end
				end++
				goto Regular
			}

			if p.flags&StrictMode != 0 {
				return nil, buffer, &Error{ErrTrailingBackslash, -1}
			}

			p.op(OpEscape, "\\")
		case '"', '\'':
			/** Quoted Character **/
			if p.flags&IgnoreQuote != 0 {
				goto Regular
			}
			submit(end)

			que = ch
			queSta = end
			p.op(OpQuote, string(ch))
		case '{':
			/** Braces Open **/
			submit(end)

			blocks = append(blocks, block{base: len(p.stack), delims: 0, ranges: 0})
			blk = &blocks[len(blocks)-1]
			p.op(opBraceOpen, "{")
		case ',':
			/** Braces Comma Separator **/
			if blk == nil {
				goto Regular
			}
			submit(end)

			if blk.ranges < -1 || blk.ranges > 0 { // range mode => alternate mode
				blk.ranges = 0
				literalize(blk.base+1, false)
			}

			blk.delims++
			p.concat(-1)
			p.op(opBraceDelim, ",")
		case '.':
			/** Braces Range Separator **/
			if blk == nil || blk.delims > 0 || blk.ranges < 0 || blk.ranges >= 2 {
				goto Regular
			}
			if end+1 >= len(input) || input[end+1] != '.' {
				goto Regular
			}
			submit(end)

			if numSub := len(p.stack) - blk.base; numSub != 2 && numSub != 4 { // invalid range symbol
				blk.ranges = ^blk.ranges
				goto Regular
			}
			end++

			blk.ranges++
			p.op(opBraceRange, "..")
		case '}':
			/** Braces Close **/
			if blk == nil {
				goto Regular
			}

			blocks = blocks[:len(blocks)-1]
			b := blk
			if blk = nil; len(blocks) > 0 {
				blk = &blocks[len(blocks)-1]
			}

			submit(end)

			// Parse Alternate
			if b.delims > 0 {
				p.concat(-1)
				p.alternate(b.base)
				continue
			}

			// Parse Ranges
			if b.ranges > 0 && p.ranges(b.base) {
				continue
			}

			// Rollback to literal
			literalize(b.base, true)
			goto Regular
		}
	}

	if que > 0 && p.flags&StrictMode != 0 {
		return nil, buffer, &Error{ErrMissingQuote, queSta}
	}

	// Last Literal
	submit(len(input))

	// Non-Closed braces rollback to literal
	if len(blocks) > 0 {
		literalize(blocks[0].base, true)
	}

	// Finalize
	p.concat(0)
	return p.stack[0], buffer, nil
}

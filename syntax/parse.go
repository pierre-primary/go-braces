package syntax

import (
	"errors"
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
	ErrMissingQuote = errors.New("missing closing quote character")
)

type Flags uint16

const (
	IgnoreEscape Flags = 1 << iota
	IgnoreQuote
	AnyCharRange
	PermissiveMode
)

type Parser struct {
	flags Flags
	stack []*Node
	free  *Node
}

func (p *Parser) newNode(op Op) (node *Node) {
	node = p.free
	if node != nil {
		p.free = node.Next
		*node = Node{}
	} else {
		node = new(Node)
	}
	node.Op = op
	return node
}

func (p *Parser) reuse(node *Node) {
	node.Next = p.free
	p.free = node
}

func (p *Parser) push(node *Node) {
	p.stack = append(p.stack, node)
}

func (p *Parser) node(op Op, val string) {
	node := p.newNode(op)
	node.Val = append(node.Val0[:0], val...)
	p.push(node)
}

func (p *Parser) literal(val string) {
	if len(p.stack) > 0 {
		if node := p.stack[len(p.stack)-1]; node.Op == OpLiteral {
			node.Val = append(node.Val, val...)
			return
		}
	}
	p.node(OpLiteral, val)
}

func (p *Parser) flatten(subs []*Node, op Op, nodes []*Node) []*Node {
	for _, item := range nodes {
		if item.Op < opPseudo {
			if item.Op == op {
				subs = append(subs, item.Subs...)
				p.reuse(item)
			} else {
				subs = append(subs, item)
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
		p.node(OpEmpty, "")
		fallthrough
	case 1:
		return
	}

	nodes := p.stack[offset:]
	p.stack = p.stack[:offset]

	node := p.newNode(OpConcat)
	node.Subs = p.flatten(node.Subs, OpConcat, nodes)
	if subs := node.Subs; len(subs) > 1 {
		last := subs[0]
		for _, item := range subs[1:] {
			last.link(item)
			last = item
		}
	}
	p.push(node)
}

func (p *Parser) alternate(offset int) {
	nodes := p.stack[offset:]
	p.stack = p.stack[:offset]

	// ASSERT: at least two nodes required
	node := p.newNode(OpAlternate)
	node.Subs = p.flatten(node.Subs, OpAlternate, nodes)
	// TODO: optimize node.Subs
	p.push(node)
}

func (p *Parser) literalize(offset int, backward bool, buffer []byte) []byte {
	if backward && offset > 0 && p.stack[offset-1].Op == OpLiteral {
		offset--
	}
	var first *Node
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
	nodes := p.stack[offset:]
	sep := 0
	switch len(nodes) {
	default:
		return false
	case 6:
		if _ = nodes[5]; nodes[4].Op != opBraceRange || nodes[5].Op != OpLiteral {
			return false
		}
		if ok, sep = parseInt(nodes[5].Val); !ok {
			return false
		}
		fallthrough
	case 4:
		if _ = nodes[3]; nodes[2].Op != opBraceRange {
			return false
		}
	}

	var vs, ve []byte
	if nodes[1].Op == OpLiteral {
		vs = nodes[1].Val
	} else if nodes[1].Op == OpEscape {
		vs = nodes[1].Val[1:]
	} else {
		return false
	}

	if nodes[3].Op == OpLiteral {
		ve = nodes[3].Val
	} else if nodes[3].Op == OpEscape {
		ve = nodes[3].Val[1:]
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
	for _, item := range nodes {
		p.reuse(item)
	}

	node := p.newNode(op)
	node.Val = data
	p.push(node)
	return true
}

func (p *Parser) Parse(input string, buffer []byte) (*Node, []byte, error) {
	type block struct {
		base   int // Base Stack Index
		ranges int
		delims int
	}
	blocks := make([]block, 0, 4)
	var blk *block

	buffer = buffer[:0]
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

	for end := 0; ; end++ {
		goto Skip
	Regular:
		if sta < 0 {
			sta = end
		}
		if input[end] < utf8.RuneSelf {
			end++
		} else {
			_, w := utf8.DecodeRuneInString(input[end:])
			end += w
		}
		/** Escape **/
		if esc > 0 {
			esc = 0
			if que > 0 {
				goto Regular
			}
			p.node(OpEscape, input[sta:end])
			sta = ^sta
		}
	Skip:
		if end >= len(input) {
			break
		}

		ch := input[end]

		/** In Quoted **/
		if que > 0 {
			if ch == '\\' {
				esc = ch
				goto Regular
			}
			if que != ch {
				goto Regular
			}

			que = 0
			submit(end)
			p.node(OpQuote, string(ch))
			continue
		}

		switch ch {
		default:
			goto Regular
		case '\\':
			/** Escape Character **/
			if p.flags&IgnoreEscape != 0 || end+1 >= len(input) {
				goto Regular
			}

			esc = ch
			submit(end)
			sta = end
			end++
			goto Regular
		case '"', '\'', '`':
			/** Quoted Character **/
			if p.flags&IgnoreQuote != 0 {
				goto Regular
			}

			que = ch
			submit(end)
			p.node(OpQuote, string(ch))
		case '{':
			/** Braces Open **/
			submit(end)
			blocks = append(blocks, block{base: len(p.stack), delims: 0, ranges: 0})
			blk = &blocks[len(blocks)-1]
			p.node(opBraceOpen, "{")
		case ',':
			/** Braces Comma Separator **/
			if blk == nil {
				goto Regular
			}

			// range mode => alternate mode
			if blk.ranges < -1 || blk.ranges > 0 {
				blk.ranges = 0
				literalize(blk.base+1, false)
			}

			submit(end)

			p.concat(-1)
			p.node(opBraceDelim, ",")
			blk.delims++
		case '.':
			/** Braces Range Separator **/
			if blk == nil {
				goto Regular
			}
			if end+1 >= len(input) || input[end+1] != '.' {
				goto Regular
			}

			// invalid range
			if blk.delims > 0 || blk.ranges < 0 || blk.ranges >= 2 {
				goto Regular
			}

			submit(end)

			if numSub := len(p.stack) - blk.base; numSub != 2 && numSub != 4 {
				blk.ranges = ^blk.ranges
				goto Regular
			}

			p.node(opBraceRange, "..")
			blk.ranges++
			end++
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

	if p.flags&PermissiveMode == 0 && que > 0 {
		return nil, buffer, ErrMissingQuote
	}

	// Non-Closed braces rollback to literal
	if len(blocks) > 0 {
		literalize(blocks[0].base, true)
	}

	// Last Literal
	if sta >= 0 {
		p.literal(input[sta:])
	}

	// Finalize
	p.concat(0)
	return p.stack[0], buffer, nil
}

func NewParser(flags ...Flags) *Parser {
	var flag Flags
	for _, f := range flags {
		flag |= f
	}
	return &Parser{flags: flag}
}

func Parse(input string, buffer []byte, flags ...Flags) (*Node, []byte, error) {
	return NewParser(flags...).Parse(input, buffer)
}

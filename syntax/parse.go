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

type Parser struct {
	stack        []*Node
	free         *Node
	NoEmpty      bool
	IgnoreEscape bool
	KeepEscape   bool
	IgnoreQuote  bool
	KeepQuote    bool
	AnyCharRange bool
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

func (p *Parser) newGroup(op Op, nodes []*Node) *Node {
	node := p.newNode(op)
	subs := node.Subs[:0]
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
	node.Subs = subs
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

func (p *Parser) concat(offset int) {
	if offset < 0 {
		offset = len(p.stack)
		for offset > 0 && p.stack[offset-1].Op < opPseudo {
			offset--
		}
	}

	nodes := p.stack[offset:]
	switch len(nodes) {
	case 0:
		if !p.NoEmpty {
			p.node(OpEmpty, "")
		}
		return
	case 1:
		return
	}
	p.stack = p.stack[:offset]

	node := p.newGroup(OpConcat, nodes)
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
	node := p.newGroup(OpAlternate, nodes)
	// TODO: optimize node.Subs
	p.push(node)
}

func (p *Parser) literalize(offset int, backward bool, buffer []byte) []byte {
	if backward && offset > 0 && p.stack[offset-1].Op == OpLiteral {
		offset--
	}
	nodes := p.stack
	var first *Node
	for idx := offset; idx < len(nodes); idx++ {
		item := nodes[idx]
		if item.Op != OpLiteral && item.Op < opPseudo {
			if first != nil {
				first.Val = append(first.Val, buffer...)
			}
			first = nil
		} else if first == nil {
			first, buffer = item, buffer[:0]
			first.Op = OpLiteral
		} else {
			buffer = append(buffer, item.Val...)
			p.reuse(item)
			continue
		}
		if idx > offset {
			nodes[offset] = item
		}
		offset++
	}
	if first != nil {
		first.Val = append(first.Val, buffer...)
	}
	p.stack = nodes[:offset]
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
		num = int(div)
	}

	opts := [5]int{sta, end, int(num), sep, wid}
	const size = 16 << (^uint(0) >> 63)
	return true, unsafe.Slice((*byte)(unsafe.Pointer(&opts[0])), size)
}

func (p *Parser) ranges(offset int) bool {
	nodes := p.stack[offset:]
	sep := 0
	switch len(nodes) {
	case 4:
	case 6:
		if ok, v := parseInt(nodes[5].Val); ok {
			sep = v
			break
		}
		return false
	default:
		return false
	}

	_ = nodes[3] // assert bound

	vs, ve := nodes[1].Val, nodes[3].Val
	ls, le := len(vs), len(ve)

	op, wid := OpUnknown, 0
	var sta, end int
	if ls == 1 && le == 1 {
		cs, ce := vs[0], ve[0]
		if isDigit(cs) && isDigit(ce) {
			op, sta, end = OpIntegerRange, int(cs-'0'), int(ce-'0')
		} else if p.AnyCharRange && cs < utf8.RuneSelf && ce < utf8.RuneSelf {
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
	case p.AnyCharRange && op == OpUnknown:
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

func (p *Parser) Parse(input string, buffer []byte) (*Node, []byte) {
	type block struct {
		base   int // Base Stack Index
		ranges int
		delims int
	}
	blocks := make([]block, 0, 4)

	p.stack = p.stack[:0]
	sta := -1

	submit := func(idx int) {
		if sta >= 0 {
			p.literal(input[sta:idx])
			sta = ^sta
		}
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
	Skip:
		if end >= len(input) {
			break
		}

		/** After Escaped **/
		if esc > 0 {
			esc = 0
			goto Regular
		}

		ch := input[end]

		/** Escape Character **/
		if ch == '\\' {
			if p.IgnoreEscape || end >= len(input)-1 {
				goto Regular
			}

			esc = ch
			if p.KeepEscape {
				goto Regular
			}

			submit(end)
			continue
		}

		/** In Quoted **/
		if que > 0 {
			if que != ch {
				goto Regular
			}

			que = 0
			if p.KeepQuote {
				goto Regular
			}

			submit(end)
			continue
		}

		switch ch {
		default:
			goto Regular
		case '"', '\'', '`':
			/** Quoted Character **/
			if p.IgnoreQuote {
				goto Regular
			}

			que = ch
			if p.KeepQuote {
				goto Regular
			}

			submit(end)
		case '{':
			/** Braces Open **/
			submit(end)
			blocks = append(blocks, block{base: len(p.stack), delims: 0, ranges: 0})
			p.node(opBraceOpen, "{")
		case ',':
			/** Braces Comma Separator **/
			if len(blocks) == 0 {
				goto Regular
			}
			b := &blocks[len(blocks)-1]

			// range mode => alternate mode
			if b.ranges < -1 || b.ranges > 0 {
				b.ranges = 0
				buffer = p.literalize(b.base+1, false, buffer)
			}

			submit(end)
			p.concat(-1)
			p.node(opBraceDelim, ",")
			b.delims++
		case '.':
			/** Braces Range Separator **/
			if end+1 >= len(input) {
				break
			}
			if len(blocks) == 0 || input[end+1] != '.' {
				goto Regular
			}
			b := &blocks[len(blocks)-1]

			// invalid range
			if b.delims > 0 || b.ranges < 0 || b.ranges >= 2 {
				goto Regular
			}
			if numSub := len(p.stack) - b.base; numSub != 1 && numSub != 3 {
				b.ranges = ^b.ranges
				goto Regular
			}

			submit(end)

			p.node(opBraceRange, "..")
			b.ranges++
			end++
		case '}':
			/** Braces Close **/
			if len(blocks) == 0 {
				goto Regular
			}
			b := &blocks[len(blocks)-1]
			blocks = blocks[:len(blocks)-1]

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
			buffer = p.literalize(b.base, true, buffer)
			goto Regular
		}
	}

	// Non-Closed braces rollback to literal
	if len(blocks) > 0 {
		buffer = p.literalize(blocks[0].base, true, buffer)
	}

	// Last Literal
	if sta >= 0 {
		submit(len(input))
	}

	// Finalize
	p.concat(0)
	return p.stack[0], buffer
}

func Parse(input string, buffer []byte) (*Node, []byte) {
	return (&Parser{}).Parse(input, nil)
}

package syntax

type Error struct {
	Msg string
	idx int
}

func (e *Error) Error() string {
	var buf []byte
	buf = append(buf, "error parsing pattern: "...)
	buf = append(buf, e.Msg...)
	buf = appendNumber(buf, e.idx, 0)
	return string(buf)
}

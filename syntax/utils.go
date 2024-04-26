package syntax

const (
	intSize = 32 << (^uint(0) >> 63)
	MaxInt  = 1<<(intSize-1) - 1
)

func parseInt(val []byte) (ok bool, num int) {
	maxVal := uint(MaxInt)
	neg := false
	if len(val) > 1 {
		switch val[0] {
		case '-':
			neg = true
			maxVal += 1
			fallthrough
		case '+':
			val = val[1:]
		}
	}

	cutoff := uint(maxVal / 10)
	ok = true
	u := uint(0)
	for _, b := range val {
		if b < '0' || b > '9' {
			return false, 0
		}
		if u > cutoff {
			ok, u = false, maxVal
			break
		}
		u *= 10

		_u := u + uint(b-'0')
		if _u < u || _u > maxVal {
			ok, u = false, maxVal
			break
		}
		u = _u
	}

	if u == 0 {
		return ok, 0
	}

	if neg {
		return ok, ^int(u - 1)
	} else {
		return ok, int(u)
	}

}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isLowerCase(b byte) bool {
	return 'a' <= b && b <= 'z'
}

func isUpperCase(b byte) bool {
	return 'A' <= b && b <= 'Z'
}

func absToUint(x int) uint {
	if x < 0 {
		return uint(-x)
	}
	return uint(x)
}

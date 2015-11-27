package utils

const primeRK = 16777619

func hashStrRev(sep string) (uint32, uint32) {
	hash := uint32(0)
	for i := len(sep) - 1; i >= 0; i-- {
		hash = hash*primeRK + uint32(sep[i])
	}
	var pow, sq uint32 = 1, primeRK
	for i := len(sep); i > 0; i >>= 1 {
		if i&1 != 0 {
			pow *= sq
		}
		sq *= sq
	}
	return hash, pow
}

func LastNthIndexByte(s string, c byte, x int) int {
	y := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			y += 1
			if y == x {
				return i
			}
		}
	}
	return -1
}

func LastNthIndex(s, sep string, x int) int {
	n := len(sep)
	switch {
	case n == 0:
		return len(s)
	case n == 1:
		return LastNthIndexByte(s, sep[0], x)
	case n == len(s):
		if sep == s {
			return 0
		}
		return -1
	case n > len(s):
		return -1
	}
	// Rabin-Karp search from the end of the string
	y := 0
	hashsep, pow := hashStrRev(sep)
	last := len(s) - n
	var h uint32
	for i := len(s) - 1; i >= last; i-- {
		h = h*primeRK + uint32(s[i])
	}
	if h == hashsep && s[last:] == sep {
		return last
	}
	for i := last - 1; i >= 0; i-- {
		h *= primeRK
		h += uint32(s[i])
		h -= pow * uint32(s[i+n])
		if h == hashsep && s[i:i+n] == sep {
			y += 1
			if y == x {
				return i
			}
		}
	}
	return -1
}

package maker

import (
	"errors"
	"strconv"
	"strings"
)

func WriteIntLong(b []byte, offset int, v int64) {
	b[offset] = byte((v >> 0) & 0xFF)
	offset++
	b[offset] = byte((v >> 8) & 0xFF)
	offset++
	b[offset] = byte((v >> 16) & 0xFF)
	offset++
	b[offset] = byte((v >> 24) & 0xFF)
}

func IpString2Int64(IpStr string) (int64, error) {
	bits := strings.Split(IpStr, ".")
	if len(bits) != 4 {
		return 0, errors.New("ip format error")
	}

	var sum int64
	for i, n := range bits {
		bit, _ := strconv.ParseInt(n, 10, 64)
		sum += bit << uint(24-8*i)
	}

	return sum, nil
}

func IpInt642String(n int64) string {
	var m int64 = 8
	s := make([]string, 4)
	for i := 3; i >= 0; n >>= m {
		s[i] = strconv.Itoa(int(n & ((1 << m) - 1)))
		i--
	}
	ipStr := strings.Join(s, ".")
	return ipStr
}

func Ip2long(IpStr string) (int64, error) {
	bits := strings.Split(IpStr, ".")
	if len(bits) != 4 {
		return 0, errors.New("ip format error")
	}

	var sum int64
	for i, n := range bits {
		bit, _ := strconv.ParseInt(n, 10, 64)
		sum += bit << uint(24-8*i)
	}

	return sum, nil
}

func IpLong2String(n int64) string {
	var m int64 = 8
	s := make([]string, 4)
	for i := 3; i >= 0; n >>= m {
		s[i] = strconv.Itoa(int(n & ((1 << m) - 1)))
		i--
	}
	ipStr := strings.Join(s, ".")
	return ipStr
}

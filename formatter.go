package pg

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

func AppendQ(dst []byte, src string, args ...interface{}) ([]byte, error) {
	p := newQueryFormatter(dst, src)
	for _, arg := range args {
		if err := p.Format(arg); err != nil {
			return nil, err
		}
	}
	return p.Value()
}

func FormatQ(src string, args ...interface{}) (Q, error) {
	b, err := AppendQ(nil, src, args...)
	if err != nil {
		return "", err
	}
	return Q(b), nil
}

func MustFormatQ(src string, args ...interface{}) Q {
	q, err := FormatQ(src, args...)
	if err != nil {
		panic(err)
	}
	return q
}

func appendPgString(dst []byte, src string) []byte {
	dst = append(dst, '\'')
	for _, c := range []byte(src) {
		switch c {
		case '\'':
			dst = append(dst, "''"...)
		case '\000':
			continue
		default:
			dst = append(dst, c)
		}
	}
	dst = append(dst, '\'')
	return dst
}

func appendPgBytes(dst []byte, src []byte) []byte {
	tmp := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(tmp, src)

	dst = append(dst, "'\\x"...)
	dst = append(dst, tmp...)
	dst = append(dst, '\'')
	return dst
}

func appendPgSubString(dst []byte, src string) []byte {
	dst = append(dst, '"')
	for _, c := range []byte(src) {
		switch c {
		case '\'':
			dst = append(dst, "''"...)
		case '\000':
			continue
		case '\\':
			dst = append(dst, '\\', '\\')
		case '"':
			dst = append(dst, '\\', '"')
		default:
			dst = append(dst, c)
		}
	}
	dst = append(dst, '"')
	return dst
}

func appendValue(dst []byte, src interface{}) []byte {
	switch v := src.(type) {
	case bool:
		if v {
			return append(dst, "'t'"...)
		}
		return append(dst, "'f'"...)
	case int8:
		return strconv.AppendInt(dst, int64(v), 10)
	case int16:
		return strconv.AppendInt(dst, int64(v), 10)
	case int32:
		return strconv.AppendInt(dst, int64(v), 10)
	case int64:
		return strconv.AppendInt(dst, int64(v), 10)
	case int:
		return strconv.AppendInt(dst, int64(v), 10)
	case uint8:
		return strconv.AppendInt(dst, int64(v), 10)
	case uint16:
		return strconv.AppendInt(dst, int64(v), 10)
	case uint32:
		return strconv.AppendInt(dst, int64(v), 10)
	case uint64:
		return strconv.AppendInt(dst, int64(v), 10)
	case uint:
		return strconv.AppendInt(dst, int64(v), 10)
	case string:
		return appendPgString(dst, v)
	case time.Time:
		dst = append(dst, '\'')
		dst = append(dst, v.UTC().Format(datetimeFormat)...)
		dst = append(dst, '\'')
		return dst
	case []byte:
		return appendPgBytes(dst, v)
	case []string:
		if len(v) == 0 {
			return append(dst, "'{}'"...)
		}

		dst = append(dst, "'{"...)
		for _, s := range v {
			dst = appendPgSubString(dst, s)
			dst = append(dst, ',')
		}
		dst[len(dst)-1] = '}'
		dst = append(dst, '\'')
		return dst
	case []int:
		if len(v) == 0 {
			return append(dst, "'{}'"...)
		}

		dst = append(dst, "'{"...)
		for _, n := range v {
			dst = strconv.AppendInt(dst, int64(n), 10)
			dst = append(dst, ',')
		}
		dst[len(dst)-1] = '}'
		dst = append(dst, '\'')
		return dst
	case []int64:
		if len(v) == 0 {
			return append(dst, "'{}'"...)
		}

		dst = append(dst, "'{"...)
		for _, n := range v {
			dst = strconv.AppendInt(dst, n, 10)
			dst = append(dst, ',')
		}
		dst[len(dst)-1] = '}'
		dst = append(dst, '\'')
		return dst
	case map[string]string:
		if len(v) == 0 {
			return append(dst, "''"...)
		}

		dst = append(dst, '\'')
		for key, value := range v {
			dst = appendPgSubString(dst, key)
			dst = append(dst, '=', '>')
			dst = appendPgSubString(dst, value)
			dst = append(dst, ',')
		}
		dst[len(dst)-1] = '\''
		return dst
	case Appender:
		return v.Append(dst)
	default:
		panic(fmt.Sprintf("pg: unsupported src type: %T", src))
	}
}

type queryFormatter struct {
	*parser
	dst []byte
}

func newQueryFormatter(dst []byte, src string) *queryFormatter {
	return &queryFormatter{
		parser: &parser{b: []byte(src)},
		dst:    dst,
	}
}

func (f *queryFormatter) Format(v interface{}) (err error) {
	for f.Valid() {
		c := f.Next()
		if c == '?' {
			f.dst = appendValue(f.dst, v)
			return nil
		}
		f.dst = append(f.dst, c)
	}
	if err != nil {
		return err
	}
	return errExpectedPlaceholder
}

func (f *queryFormatter) Value() ([]byte, error) {
	for f.Valid() {
		c := f.Next()
		if c == '?' {
			return nil, errUnexpectedPlaceholder
		}
		f.dst = append(f.dst, c)
	}
	return f.dst, nil
}

package charset

import (
	"io"
	"strings"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// charsetMap maps charset names to their encoding
var charsetMap = map[string]encoding.Encoding{
	// ISO 8859 series
	"iso-8859-1":  charmap.ISO8859_1,
	"iso8859-1":   charmap.ISO8859_1,
	"iso-8859-2":  charmap.ISO8859_2,
	"iso8859-2":   charmap.ISO8859_2,
	"iso-8859-3":  charmap.ISO8859_3,
	"iso8859-3":   charmap.ISO8859_3,
	"iso-8859-4":  charmap.ISO8859_4,
	"iso8859-4":   charmap.ISO8859_4,
	"iso-8859-5":  charmap.ISO8859_5,
	"iso8859-5":   charmap.ISO8859_5,
	"iso-8859-6":  charmap.ISO8859_6,
	"iso8859-6":   charmap.ISO8859_6,
	"iso-8859-7":  charmap.ISO8859_7,
	"iso8859-7":   charmap.ISO8859_7,
	"iso-8859-8":  charmap.ISO8859_8,
	"iso8859-8":   charmap.ISO8859_8,
	"iso-8859-9":  charmap.ISO8859_9,
	"iso8859-9":   charmap.ISO8859_9,
	"iso-8859-10": charmap.ISO8859_10,
	"iso8859-10":  charmap.ISO8859_10,
	"iso-8859-13": charmap.ISO8859_13,
	"iso8859-13":  charmap.ISO8859_13,
	"iso-8859-14": charmap.ISO8859_14,
	"iso8859-14":  charmap.ISO8859_14,
	"iso-8859-15": charmap.ISO8859_15,
	"iso8859-15":  charmap.ISO8859_15,
	"iso-8859-16": charmap.ISO8859_16,
	"iso8859-16":  charmap.ISO8859_16,

	// Windows codepages
	"windows-1250": charmap.Windows1250,
	"windows-1251": charmap.Windows1251,
	"windows-1252": charmap.Windows1252,
	"windows-1253": charmap.Windows1253,
	"windows-1254": charmap.Windows1254,
	"windows-1255": charmap.Windows1255,
	"windows-1256": charmap.Windows1256,
	"windows-1257": charmap.Windows1257,
	"windows-1258": charmap.Windows1258,
	"cp1250":       charmap.Windows1250,
	"cp1251":       charmap.Windows1251,
	"cp1252":       charmap.Windows1252,

	// Unicode
	"utf-8":    unicode.UTF8,
	"utf8":     unicode.UTF8,
	"utf-16":   unicode.UTF16(unicode.BigEndian, unicode.UseBOM),
	"utf16":    unicode.UTF16(unicode.BigEndian, unicode.UseBOM),
	"utf-16be": unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM),
	"utf-16le": unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM),

	// Asian encodings
	"gb2312":     simplifiedchinese.HZGB2312,
	"gbk":        simplifiedchinese.GBK,
	"gb18030":    simplifiedchinese.GB18030,
	"big5":       traditionalchinese.Big5,
	"euc-jp":     japanese.EUCJP,
	"iso-2022-jp": japanese.ISO2022JP,
	"shift_jis":  japanese.ShiftJIS,
	"euc-kr":     korean.EUCKR,

	// Other common encodings
	"koi8-r":     charmap.KOI8R,
	"koi8-u":     charmap.KOI8U,
	"macintosh":  charmap.Macintosh,
	"us-ascii":   charmap.Windows1252, // Treat ASCII as Windows-1252
	"ascii":      charmap.Windows1252,
}

// NewReader creates a reader that decodes the given charset to UTF-8
func NewReader(r io.Reader, charsetName string) (io.Reader, error) {
	if charsetName == "" || strings.ToLower(charsetName) == "utf-8" || strings.ToLower(charsetName) == "utf8" {
		return r, nil
	}

	// Normalize charset name
	charsetName = strings.ToLower(charsetName)
	charsetName = strings.ReplaceAll(charsetName, "_", "-")

	// Try our custom charset map first
	if enc, ok := charsetMap[charsetName]; ok {
		return transform.NewReader(r, enc.NewDecoder()), nil
	}

	// Fall back to golang.org/x/net/html/charset
	return charset.NewReader(r, charsetName)
}

// DecodeString decodes a string from the given charset to UTF-8
func DecodeString(s, charsetName string) (string, error) {
	if charsetName == "" || strings.ToLower(charsetName) == "utf-8" || strings.ToLower(charsetName) == "utf8" {
		return s, nil
	}

	reader, err := NewReader(strings.NewReader(s), charsetName)
	if err != nil {
		return s, err // Return original string if decoding fails
	}

	decoded, err := io.ReadAll(reader)
	if err != nil {
		return s, err // Return original string if reading fails
	}

	return string(decoded), nil
}

// IsSupported checks if a charset is supported
func IsSupported(charsetName string) bool {
	if charsetName == "" {
		return true
	}

	charsetName = strings.ToLower(charsetName)
	charsetName = strings.ReplaceAll(charsetName, "_", "-")

	// Check our custom map
	if _, ok := charsetMap[charsetName]; ok {
		return true
	}

	// Check if golang.org/x/net/html/charset supports it
	enc, _ := charset.Lookup(charsetName)
	return enc != nil
}
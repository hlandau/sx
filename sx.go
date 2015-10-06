// Package sx provides utilities for parsing and serializing S-expressions.
package sx

import "io"
import "io/ioutil"
import "fmt"
import "bufio"
import "bytes"
import "strconv"
import "encoding/base64"
import "unicode/utf8"

// interface{} is one of
//   int
//   int64
//   uint64
//   string
//   []byte
//   List

// A S-expression format. There are many variant syntaxes. You cannot
// instantiate Format itself; you must use one of the instances provided.
type Format struct {
	// Allow quoted Unicode string: "foo"
	// Type: string (or []byte if useUnicode == false)
	allowQuotedString bool

	// Allow integers: 42
	// Type: int if sufficient, otherwise int64, otherwise uint64, otherwise error
	allowIntegers bool

	// Allow lists: (foo bar)
	// Type: ParenList
	allowLists bool

	// Allow raw binary strings: 4:blah
	// Note: this forces allowLengthPrefixes to true.
	// Type: []byte
	allowVerbatimBinaryString bool

	// Allow base64 binary strings: |...|
	// Type: []byte
	allowBase64BinaryString bool

	// Allow verbatim base64 binary strings: {...}
	// Type: (inline sequence of items)
	allowVerbatimBase64BinaryString bool

	// Allow hex binary strings: #01020304feff#
	// Type: []byte
	allowHexBinaryString bool

	// Allow bare tokens
	allowTokens bool

	maxListDepth  uint
	unicodeStream bool
}

var Csexp Format

// This package's own preferred syntax. Serializes in canonical form.
//
// The following syntactic elements are supported:
//
//   () Lists                                  -> []interface{}
//   Quoted strings with escape sequences      -> string
//   Verbatim length-prefixed strings  5:apple -> string
//   Base64 strings |...|                      -> string
//   Verbatim base64 {...}                     -> (inline item list)
//   Bare words (including integers)           -> string, int, int64, uint64
//
var SX Format

func init() {
	Csexp = Format{
		allowQuotedString:               true,
		allowIntegers:                   true,
		allowLists:                      true,
		allowVerbatimBinaryString:       true,
		allowBase64BinaryString:         true,
		allowVerbatimBase64BinaryString: true,
		allowTokens:                     true,
		allowHexBinaryString:            true,
		maxListDepth:                    255,
		unicodeStream:                   false,
	}

	SX = Format{
		allowQuotedString:               true,
		allowIntegers:                   true,
		allowLists:                      true,
		allowVerbatimBinaryString:       true,
		allowBase64BinaryString:         true,
		allowVerbatimBase64BinaryString: true,
		allowTokens:                     true,
		allowHexBinaryString:            true,
		maxListDepth:                    255,
		unicodeStream:                   true,
	}
}

// Advanced incremental parse interface. Write data to be parsed to the Parser
// and ensure you call Close. Then tokens will be available in Tokens.
//
// Ordinarily you will want to use the methods defined on Format.
type Parser struct {
	f         *Format
	state     int
	s         string
	b         []byte
	xL        uint64
	i         uint64
	neg       bool
  lenhint   bool // superfluous length hint present?
	sub       bool // is subparser for verbatim {base64} syntax?
	bytemode  byte
	reissue   int
	tokens    []interface{}
	stack     [][]interface{}
	depth     uint
	eof       bool
	b64dec    io.Reader
	b64sr     switchableReader
	sublexing bool // in verbatim base64 context?
	subb64    *writeDecoder
}

const (
	pstateDrifting = iota
	pstateInteger
	pstateNegIntegerStart
	pstateLengthQuotedString
	pstateLengthByteString
	pstateQuotedString
	pstateQuotedStringEscape
  pstateQuotedStringHexEscape
  pstateQuotedStringHexEscape2
  pstateQuotedStringOctalEscape
  pstateQuotedStringOctalEscape2
  pstateQuotedStringOctalEscape3
  pstateQuotedStringEscapeCR
  pstateQuotedStringEscapeLF
	pstateBase64String
	pstateToken
	pstateHexString
	pstateHexStringOdd
)

type err struct {
	r rune
}

func (e *err) Error() string {
	return fmt.Sprintf("invalid token: unexpected character %v", e.r)
}

func (p *Parser) init() {
	if !p.f.unicodeStream {
		p.bytemode++
	}
}

var ErrDepthLimitExceeded = fmt.Errorf("list depth limit exceeded")
var ErrListEnd = fmt.Errorf("attempted to close a list while not in a list")

func (p *Parser) Write(b []byte) (int, error) {
	if p.sublexing {
		idx := bytes.IndexByte(b, '}')
		if idx < 0 {
			return p.subb64.Write(b)
		} else {
			n, err := p.subb64.Write(b[0:idx])
			if err != nil {
				return n, err
			}
			p.sublexing = false
			n2, err := p.write(b[idx+1:])
			return n + n2, err
		}
	}

	return p.write(b)
}

type writerFunc func(b []byte) (int, error)

func (w writerFunc) Write(b []byte) (int, error) {
	return w(b)
}

func isTokenStartChar(r rune) bool {
	return (r >= 'A' && r <= 'Z') || r == '_' || (r >= 'a' && r <= 'z') ||
		r == '.' || r == '.' || r == '/' || r == ':' ||
		r == '*' || r == '+' || r == '=' || r == '-'
}

func isTokenChar(r rune) bool {
	return isTokenStartChar(r) || (r >= '0' && r <= '9')
}

func dechex(r rune) (byte, bool) {
	if r >= '0' && r <= '9' {
		return byte(r - '0'), true
	} else if r >= 'a' && r <= 'z' {
		return byte(r - 'a' + 10), true
	} else if r >= 'A' && r <= 'Z' {
		return byte(r - 'A' + 10), true
	} else {
		return 0, false
	}
}

func decoct(r rune) (byte, bool) {
  if r >= '0' && r <= '7' {
    return byte(r - '0'), true
  } else {
    return 0, false
  }
}

const useUnicode = true

func (p *Parser) write(b []byte) (int, error) {
	i := 0
	var r rune
	for {
		if p.reissue > 0 {
			p.reissue--
		} else {
			if i >= len(b) {
				break
			}

			if !useUnicode || p.bytemode != 0 {
        r = rune(b[i])
        i += 1
			} else {
			  var sz int
				r, sz = utf8.DecodeRune(b[i:])
				// ignore errors
				i += sz
			}
		}

		switch p.state {
		case pstateDrifting:
			if p.eof {
				return i, nil
			}

			switch {
			case r == ' ' || r == '\t' || r == '\r' || r == '\n':
				// nop
			case r >= '0' && r <= '9' && p.f.allowIntegers:
				p.state = pstateInteger
				p.reissue++
			case r == '-' && p.f.allowIntegers:
				p.state = pstateNegIntegerStart
			case r == '(' && p.f.allowLists:
				if p.depth >= p.f.maxListDepth {
					return i, ErrDepthLimitExceeded
				}
				p.depth++
				p.stack = append(p.stack, p.tokens)
				p.tokens = make([]interface{}, 0)
			case r == ')' && p.f.allowLists:
				if p.depth == 0 {
					return i, ErrListEnd
				}
				p.depth--
				ptok := p.stack[len(p.stack)-1]
				p.stack = p.stack[0 : len(p.stack)-1]
				ptok = append(ptok, p.tokens)
				p.tokens = ptok
			case r == '"' && p.f.allowQuotedString:
				p.state = pstateQuotedString
			case r == '|' && p.f.allowBase64BinaryString:
				p.state = pstateBase64String
				p.b64dec = base64.NewDecoder(base64.StdEncoding, &filteringReader{&p.b64sr,})
			case r == '{' && p.f.allowVerbatimBase64BinaryString && !p.sublexing:
				p.sublexing = true
				p.subb64 = newWriteDecoder(writerFunc(p.write))
				return p.Write(b[i:]) // i indexes next character, not this one
			case p.f.allowTokens && isTokenStartChar(r):
				p.state = pstateToken
				p.reissue++
			case p.f.allowHexBinaryString && r == '#':
				p.state = pstateHexString
			default:
				return i, &err{r}
			}
		case pstateToken:
			if !isTokenChar(r) {
				p.reissue++
				p.state = pstateDrifting
				p.push(p.s)
				p.s = ""
			} else {
				p.s += string(r)
			}
		case pstateNegIntegerStart:
			switch {
			case r >= '0' && r <= '9':
				p.state = pstateInteger
				p.neg = true
				p.reissue++
			default:
        p.state = pstateToken
        p.s = "-"
        p.reissue++
			}
		case pstateInteger:
			switch {
			case r >= '0' && r <= '9':
				p.i = p.i*10 + uint64(r-'0')
			case r == '"' && p.f.allowQuotedString && !p.neg:
				p.xL = p.i
				p.i = 0
				p.state = pstateLengthQuotedString
      case r == '#' && p.f.allowHexBinaryString && !p.neg:
        p.xL = p.i
        p.i = 0
        p.state = pstateHexString
        p.lenhint = true
      case r == '|' && p.f.allowBase64BinaryString && !p.neg:
        p.xL = p.i
        p.i = 0
        p.state = pstateBase64String
        p.lenhint = true
			case r == ':' && p.f.allowVerbatimBinaryString && !p.neg:
				p.xL = p.i
				p.i = 0
				p.state = pstateLengthByteString
        p.lenhint = true
				p.bytemode++
			default:
				if p.neg {
					// These negations work even for INT_MIN since the cast operators
					// here operate like reinterpret_casts, and -INT_MIN == INT_MIN.
					if p.i <= 0x80000000 {
						p.push(-int(p.i))
					} else {
						p.push(-int64(p.i))
					}
				} else {
					if p.i <= 0x7FFFFFFF {
						p.push(int(p.i))
					} else {
						p.push(p.i)
					}
				}
				p.i = 0
				p.neg = false
				p.reissue++
				p.state = pstateDrifting
			}
		case pstateLengthByteString:
			if p.xL == 0 {
				p.bytemode--
				p.state = pstateDrifting
				p.push(p.b)
				p.b = []byte{}
				p.reissue++
			} else {
				p.b = append(p.b, byte(r))
				p.xL--
			}
		case pstateLengthQuotedString:
			if p.xL == 0 {
				if r != '"' {
					// error
				}
				p.state = pstateDrifting
				p.push(p.s)
				p.s = ""
				// consume trailing quote
			} else {
				p.s += string(r)
				p.xL--
			}
		case pstateQuotedString:
			switch r {
			case '"':
				p.state = pstateDrifting
				p.push(p.s)
				p.s = ""
      case '\\':
				p.state = pstateQuotedStringEscape
			default:
				p.s += string(r)
			}
		case pstateQuotedStringEscape:
			p.state = pstateQuotedString
			switch r {
			case 'a':
				p.s += "\a"
			case 'b':
				p.s += "\b"
			case 'f':
				p.s += "\f"
			case 'n':
				p.s += "\n"
			case 'r':
				p.s += "\r"
			case 't':
				p.s += "\t"
			case 'v':
				p.s += "\v"
      case '\r':
        p.state = pstateQuotedStringEscapeLF
      case '\n':
        p.state = pstateQuotedStringEscapeCR
      case 'x':
        p.state = pstateQuotedStringHexEscape
			default:
        if r >= '0' && r <= '7' {
          p.state = pstateQuotedStringOctalEscape
          p.i = 0
          p.reissue++
        } else {
          p.s += string(r)
        }
			}
    case pstateQuotedStringHexEscape:
      v, ok := dechex(r)
      if !ok {
        return i, &err{r}
      }
      p.i = uint64(v)
      p.state = pstateQuotedStringHexEscape2
    case pstateQuotedStringHexEscape2:
      v, ok := dechex(r)
      if !ok {
        return i, &err{r}
      }
      p.s += string([]byte{byte(p.i << 4) | v})
      p.state = pstateQuotedString
      p.i = 0
    case pstateQuotedStringOctalEscape, pstateQuotedStringOctalEscape2, pstateQuotedStringOctalEscape3:
      v, ok := decoct(r)
      if !ok {
        return i, &err{r}
      }
      p.i = uint64(byte(p.i << 3) | v)
      if p.state == pstateQuotedStringOctalEscape3 {
        p.state = pstateQuotedString
        p.s += string([]byte{byte(p.i)})
        p.i = 0
      } else {
        p.state++
      }
    case pstateQuotedStringEscapeLF:
      if r != '\n' {
        p.reissue++
      }
      p.state = pstateQuotedString
    case pstateQuotedStringEscapeCR:
      if r == '\r' {
        p.reissue++
      }
      p.state = pstateQuotedString
		case pstateBase64String:
			// i indexes the nest character to read, not the current one, so -1 everything
			idx := bytes.IndexByte(b[i-1:], '|')
			if idx < 0 {
				p.b64sr.Reader = bytes.NewReader(b[i-1:])
				i = len(b)
			} else {
				p.b64sr.Reader = bytes.NewReader(b[i-1 : i-1+idx])
				i += idx
			}
			buf, _ := ioutil.ReadAll(p.b64dec)
			p.s += string(buf)
			if idx >= 0 {
        if p.lenhint && uint64(len(p.s)) != p.xL {
          return i, &err{r}
        }
				p.state = pstateDrifting
				p.push(p.s)
				p.s = ""
			}
		case pstateHexString:
			if r == '#' {
        if p.lenhint && uint64(len(p.b)) != p.xL {
          return i, &err{r}
        }
				p.state = pstateDrifting
				p.push(p.b)
				p.b = nil
				p.i = 0
        p.lenhint = false
			} else if r == ' ' || r == '\r' || r == '\n' || r == '\t' {
				// skip
			} else {
				hv, ok := dechex(r)
				if !ok {
					return i, &err{r}
				}

				p.i = uint64(hv)
				p.state = pstateHexStringOdd
			}

		case pstateHexStringOdd:
			if r == ' ' || r == '\r' || r == '\n' || r == '\t' {
				// skip
			} else {
				hv, ok := dechex(r)
				if !ok {
					return i, &err{r}
				}

				p.b = append(p.b, (byte(p.i)<<4)|hv)
				p.state = pstateHexString
			}
		default:
			panic("invalid state")
		}
	}
	return len(b), nil
}

func (p *Parser) push(tok interface{}) {
	p.tokens = append(p.tokens, tok)
}

func (p *Parser) Close() error {
	p.eof = true
	_, err := p.Write([]byte{0})
	return err
}

func (p *Parser) Tokens() []interface{} {
	return p.tokens
}

// Create a new parser using the format.
func (fmt *Format) NewParser() *Parser {
	p := &Parser{
		f: fmt,
	}
	p.init()
	return p
}

// Parse a S-expression string and return a slice of the values parsed or an
// error.
func (fmt *Format) Parse(b []byte) ([]interface{}, error) {
	p := fmt.NewParser()
	_, err := p.Write(b)
	if err != nil {
		return nil, err
	}
	err = p.Close()
	if err != nil {
		return nil, err
	}
	return p.Tokens(), nil
}

// Writes the slice as an S-expression string to the io.Writer.
func (fmt *Format) Write(vs []interface{}, w io.Writer) error {
	return write(vs, w, fmt)
}

// Formats the S-expression as a string and returns it or an error.
func (f *Format) String(vs []interface{}) (string, error) {
	b := bytes.Buffer{}
	err := write(vs, &b, f)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

var ErrUnsupportedType = fmt.Errorf("unsupported SX type")

func write(vs []interface{}, w io.Writer, fmt *Format) error {
	b := bufio.NewWriter(w)
	err := writeList(vs, b, fmt)
	if err != nil {
		return err
	}
	return b.Flush()
}

func writeInt(vs int64, b *bufio.Writer, fmt *Format) {
	b.WriteString(strconv.FormatInt(vs, 10))
}

func writeUint(vs uint64, b *bufio.Writer, fmt *Format) {
	b.WriteString(strconv.FormatUint(vs, 10))
}

type spacer struct {
	prevType rune
}

func (s *spacer) write(b *bufio.Writer, t rune) {
	if s.prevType == 'i' {
		b.WriteRune(' ')
	}
	s.prevType = t
}

func writeList(vs []interface{}, b *bufio.Writer, f *Format) error {
	var spacer spacer
	for _, v := range vs {
		switch vv := v.(type) {
		case string:
			writeUint(uint64(len(vv)), b, f)
			b.WriteRune(':')
			b.WriteString(vv)
		case []byte:
			writeUint(uint64(len(vv)), b, f)
			b.WriteRune(':')
			b.Write(vv)
		case int:
			spacer.write(b, 'i')
			writeInt(int64(vv), b, f)
		case int64:
			spacer.write(b, 'i')
			writeInt(vv, b, f)
		case uint64:
			spacer.write(b, 'i')
			writeUint(vv, b, f)
		case []interface{}:
			b.WriteRune('(')
			if err := writeList(vv, b, f); err != nil {
				return err
			}
			b.WriteRune(')')
		default:
			return ErrUnsupportedType
		}
	}

	return nil
}

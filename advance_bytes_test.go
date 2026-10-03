package vte

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// recorder writes down every call a parser makes, with its arguments.
type recorder struct {
	p   *Parser
	got []string
}

func (r *recorder) Print(c rune)   { r.got = append(r.got, fmt.Sprintf("print %q", c)) }
func (r *recorder) Execute(b byte) { r.got = append(r.got, fmt.Sprintf("exec %x", b)) }
func (r *recorder) Put(b byte)     { r.got = append(r.got, fmt.Sprintf("put %x", b)) }
func (r *recorder) Unhook()        { r.got = append(r.got, "unhook") }
func (r *recorder) Hook(params [][]uint16, inter []byte, ignore bool, final rune) {
	r.got = append(r.got, fmt.Sprintf("hook %v %q %v %q", params, inter, ignore, final))
}
func (r *recorder) OscDispatch(params [][]byte, bell bool) {
	r.got = append(r.got, fmt.Sprintf("osc %q %v", params, bell))
}
func (r *recorder) CsiDispatch(params [][]uint16, inter []byte, ignore bool, final rune) {
	r.got = append(r.got, fmt.Sprintf("csi %v %q %v %q", params, inter, ignore, final))
}
func (r *recorder) EscDispatch(inter []byte, ignore bool, final byte) {
	r.got = append(r.got, fmt.Sprintf("esc %q %v %q", inter, ignore, final))
}
func (r *recorder) SosPmApcDispatch(kind SosPmApcKind, data []byte, bell bool) {
	r.got = append(r.got, fmt.Sprintf("sospmapc %v %q %v", kind, data, bell))
}

// calls returns the calls a parser makes for in, cut into writes at
// cuts, fed with AdvanceBytes or byte by byte with Advance.
func calls(in []byte, cuts []int, whole bool) []string {
	r := &recorder{}
	p := NewParser(r)
	from := 0
	for _, to := range append(cuts, len(in)) {
		if to < from || to > len(in) {
			continue
		}
		if whole {
			p.AdvanceBytes(in[from:to])
		} else {
			for _, b := range in[from:to] {
				p.Advance(b)
			}
		}
		from = to
	}
	return r.got
}

func TestAdvanceBytesDoesWhatAdvanceDoes(t *testing.T) {
	samples := []string{
		"plain text",
		"\x1b[38;2;10;20;30m\x1b[48;2;1;2;3m▀▀ ünïcödé 日本語 🙂 e\u0301",
		"\x1b]0;title\x07\x1b]52;c;aGk=\x1b\\",
		"bad \xff\xfe utf8 \xe2\x82 cut \xc0\xaf overlong \xed\xa0\x80 surrogate \xf4\x90\x80\x80 too big",
		"\x1bP1;2|data\x1b\\\x1b_apc\x1b\\\x1b^pm\x07",
		"\x9b31m C1 \x90dcs\x9c",
		"tab\there\r\nnew\x08\x7f del",
		"\x1b[" + strings.Repeat("1;", 40) + "99999999m after",
		"\x1b[?" + strings.Repeat("12;", 33) + "h\x1b[;;;5;m",
	}
	rng := rand.New(rand.NewSource(1))
	alphabet := []byte("\x1b[];?0123456789m\x07\\ABCPXabc_^\x18\x1a\xe2\x96\x80\xf0\x9f\x99\x82\xc3\xa9\xff\x80\x90\x9c\x9b \t\r\n")
	for i := 0; i < 2000; i++ {
		n := rng.Intn(80)
		var b strings.Builder
		for j := 0; j < n; j++ {
			b.WriteByte(alphabet[rng.Intn(len(alphabet))])
		}
		samples = append(samples, b.String())
	}
	for i, s := range samples {
		in := []byte(s)
		var cuts []int
		for k := 0; k < rng.Intn(4); k++ {
			cuts = append(cuts, rng.Intn(len(in)+1))
		}
		for k := 1; k < len(cuts); k++ {
			if cuts[k] < cuts[k-1] {
				cuts[k], cuts[k-1] = cuts[k-1], cuts[k]
			}
		}
		want, got := calls(in, cuts, false), calls(in, cuts, true)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("sample %d %q cut at %v:\n got %q\nwant %q", i, s, cuts, got, want)
		}
	}
}

func BenchmarkAdvanceBytesHalfBlocks(b *testing.B) {
	var s strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&s, "\x1b[38;2;%d;90;160m\x1b[48;2;40;%d;160m▀", i%256, (i*7)%256)
	}
	in := []byte(s.String())
	p := NewParser(&recorder{})
	b.SetBytes(int64(len(in)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.performer.(*recorder).got = nil
		p.AdvanceBytes(in)
	}
}

// The quick ways AdvanceBytes takes rest on what the state table says.
func TestTheTableAgreesWithTheQuickWays(t *testing.T) {
	for b := 0; b < 256; b++ {
		any := stateTable[AnywhereState][b]
		if b >= '0' && b <= '9' || b == ';' {
			if any != 0 || stateTable[CsiParamState][b] != uint16(ParamAction)<<8 {
				t.Errorf("byte %#x in a CSI's parameters is %#x, %#x", b, any, stateTable[CsiParamState][b])
			}
		}
		if b >= 0x20 && b < 0x7f {
			if any != 0 || stateTable[GroundState][b] != uint16(PrintAction)<<8 {
				t.Errorf("byte %#x in text is %#x, %#x", b, any, stateTable[GroundState][b])
			}
		}
	}
}

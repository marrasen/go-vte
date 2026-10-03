package vte

import "unicode/utf8"

// AdvanceBytes advances the state machine over every byte of bs, as
// calling Advance on each in turn would. Text takes a quicker way:
// printable ASCII and whole UTF-8 characters go to Print without the
// state table, which is most of what a program writes. It is a change
// of this fork.
func (p *Parser) AdvanceBytes(bs []byte) {
	for i := 0; i < len(bs); {
		if p.state == CsiParamState {
			// The digits and separators of a CSI's parameters, as the
			// colours of a true colour SGR are: Advance would find
			// ParamAction in the table, and stay in the state.
			for ; i < len(bs); i++ {
				c := bs[i]
				switch {
				case p.params.IsFull() && (c >= '0' && c <= '9' || c == ';'):
					p.ignoring = true
					continue
				case c >= '0' && c <= '9':
					p.param = saddu16(smulu16(p.param, 10), uint16(c-'0'))
				case c == ';':
					p.params.Push(p.param)
					p.param = 0
				default:
				}
				if c < '0' || c > '9' && c != ';' {
					break
				}
				p.hasParam = true
			}
			if i < len(bs) {
				p.Advance(bs[i])
				i++
			}
			continue
		}
		if p.state != GroundState {
			p.Advance(bs[i])
			i++
			continue
		}
		b := bs[i]
		switch {
		case b >= 0x20 && b < 0x7f:
			p.prtcb(rune(b))
			i++
		case b >= 0xc2 && b <= 0xf4:
			r, n := utf8.DecodeRune(bs[i:])
			if r == utf8.RuneError && n <= 1 {
				// Cut short at the end, or not UTF-8: the state machine
				// keeps the start for the next write, or says what is
				// wrong, as it always did.
				p.Advance(b)
				i++
				continue
			}
			p.prtcb(r)
			i += n
		default:
			p.Advance(b)
			i++
		}
	}
}

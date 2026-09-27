# Offering this fork's changes upstream

This fork is [danielgatis/go-vte](https://github.com/danielgatis/go-vte)
v1.0.11 with two changes, on the `gt` branch, tagged `v1.0.11-gt.1`.
gridterm uses it through a `replace` line in its `go.mod`.

The two changes are independent, so they go upstream as two pull
requests. The first is a plain fix. The second changes what a
`Performer` may do with the slices it is handed, so upstream may want it
behind an option, or not at all. If both are merged and released,
gridterm can drop the fork.

Below is the text for each pull request.

---

## Pull request 1: Cap SOS, PM and APC strings

Branch from upstream `main`, and take commit `b6e8c73` ("Cap SOS, PM and
APC strings at a megabyte").

### Title

Cap SOS, PM and APC strings, and make the tests build again

### Description

The package's tests do not build at v1.0.11. `TestApcMaxBufferSize`
uses a constant, `maxSosPmApcRaw`, that the parser does not define:

    vet: ./parser_apc_test.go:146:14: undefined: maxSosPmApcRaw

The test describes a cap on how much of a SOS, PM or APC string the
parser keeps. The parser has no such cap: every byte of the string is
appended to `sosPmApcRaw`. A program that starts an APC string and never
ends it makes the parser hold every byte it sends after that. In a
terminal emulator, that program can be anything the user runs, or
anything printed by a command such as `cat`.

This change adds the constant and the cap:

- `maxSosPmApcRaw` is 1 MiB (`1 << 20`). That is room for any real use.
  The kitty graphics protocol, for example, sends images in APC chunks
  of 4096 bytes.
- Past the cap, bytes are read and dropped. The string still ends where
  it ends, so nothing after it is misread.

With this, `go test ./...` builds and passes. `TestApcMaxBufferSize`
checks the cap.

One line of `gofmt` output changes as well: a doubled blank line above
`SosPmApcKind` becomes one.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

---

## Pull request 2: Hand out the parser's own params and intermediates

Branch from upstream `main` after pull request 1 (the tests need it to
build), and take commit `51df1ee` ("Hand out the parser's own params
and intermediates").

### Title

Stop allocating params and intermediates for every sequence

### Description

`Parser.Params()` builds a new `[][]uint16` for every CSI and DCS
sequence, growing it one parameter at a time. `Parser.Intermediates()`
copies its bytes into a new slice. Both are called for every sequence
dispatched.

For most programs this is small. For a terminal showing an animation in
true colour it is most of what the parser allocates. Every cell of a
screen drawn in half blocks sends two SGR sequences of five parameters,
such as `ESC[38;2;r;g;bm ESC[48;2;r;g;bm`. That makes 37,500 sequences
for a 250 by 75 screen, many times a second.

This change makes both functions hand out the parser's own buffers:

- `Params()` fills a slice kept on the `Parser`, reused from one
  sequence to the next. A sequence with no parameters still gets `nil`,
  as before.
- `Intermediates()` returns a slice of the parser's intermediates
  array.

**This changes the contract for a `Performer`.** The slices passed to
`Hook`, `CsiDispatch` and `EscDispatch` now hold good for the length of
the call only. A performer that keeps them past the call must copy
them. The `Performer` documentation now says so. The tests' own
dispatcher kept them, so it now copies them, with two small helpers,
`keepParams` and `keepIntermediates`.

A performer that reads the parameters during the call, which is the
usual case, needs no change.

### Measurements

The package's own benchmark, `BenchmarkStateChanges`:

| | Allocations per run |
|---|---|
| Before | 4,006 |
| After | 2,007 |

In gridterm, a terminal emulator built on go-vte, parsing one frame of a
250 by 75 screen of true colour half blocks:

| | Time per frame | Allocations per frame | Memory per frame |
|---|---|---|---|
| Before | 21 ms | 112,504 | 12.6 MB |
| After | 11.3 ms | 0 | 4 bytes |

That is three allocations for each of the 37,500 sequences.

### If the contract change is unwelcome

The same saving can be kept behind an option instead, for example a
`NewParserReusing` constructor, or a field on `Parser` that turns
reuse on. Copying stays the default, and no existing performer changes.
Happy to rework it that way.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

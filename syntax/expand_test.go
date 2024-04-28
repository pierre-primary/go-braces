package syntax_test

import (
	"testing"

	"github.com/pierre-primary/go-braces/syntax"
)

type E = []string

const (
	IgnoreEscape = syntax.IgnoreEscape
	IgnoreQuote  = syntax.IgnoreQuote
	AnyCharRange = syntax.AnyCharRange
	StrictMode   = syntax.StrictMode
)

//go:noinline
func discard(s string) {}

func BenchmarkExpand(t *testing.B) {
	expand := func(input string) func(t *testing.B) {
		return func(t *testing.B) {
			ast, _, _ := syntax.Parse(input, nil)
			t.ReportAllocs()
			t.ResetTimer()
			for i := 0; i < t.N; i++ {
				ast.Walk(discard, nil)
			}
		}
	}
	t.Run("Literal", expand("abcdefg"))
	t.Run("Alternate", expand("{abc,def,ghi}"))
	t.Run("Mixed", expand("aaa{abc,def,ghi}bbb"))
	t.Run("CharRange:[0-9]", expand("{0..9}"))
	t.Run("CharRange:[a-z]", expand("{a..z}"))
	t.Run("CharRange:[A-Z]", expand("{A..Z}"))
	t.Run("IntRange:10", expand("{1..10}"))
	t.Run("IntRange:100", expand("{1..100}"))
	t.Run("Unicode", expand("{你好吗,你在吗,你在哪}"))
}

func DefineExpand(t *testing.T) func(string, []string, ...syntax.Flags) {
	return func(input string, expected []string, flags ...syntax.Flags) {
		p := syntax.NewParser(flags...)
		ast, _, _ := p.Parse(input, nil)
		result, _ := ast.Expand(nil, nil)
		if len(result) != len(expected) {
			t.Fatal(result)
			return
		}
		for i := range result {
			if result[i] != expected[i] {
				t.Fatal(result)
				return
			}
		}
	}
}

// func TestExpand(t *testing.T) {
// 	equal := DefineExpand(t)

// 	equal(`aaa/bbb/ccc`, nil, []s{"aaa/bbb/ccc"})

// 	equal(`a/{b,c}/d`, nil, []s{"a/b/d", "a/c/d"})

// 	/** Range expansion. **/

// 	equal(`{1..2}{2..1}`, nil, []s{"12", "11", "22", "21"})
// 	equal(`a{1..2}b{2..1}c`, nil, []s{"a1b2c", "a1b1c", "a2b2c", "a2b1c"})

// 	equal(`{0..8..2}`, nil, []s{"0", "2", "4", "6", "8"})
// 	equal(`{1..8..-2}`, nil, []s{"1", "3", "5", "7"})

// 	equal(`{-1..-2}{-2..-1}`, nil, []s{"-1-2", "-1-1", "-2-2", "-2-1"})

// 	equal(`{-2..2..-1}`, nil, []s{"-2", "-1", "0", "1", "2"})
// 	equal(`{2..-2..1}`, nil, []s{"2", "1", "0", "-1", "-2"})

// 	equal(`{000..127..8}`, nil, []s{
// 		"000", "008", "016", "024", "032", "040", "048", "056",
// 		"064", "072", "080", "088", "096", "104", "112", "120",
// 	})
// 	equal(`{00..127..8}`, nil, []s{
// 		"00", "08", "16", "24", "32", "40", "48", "56",
// 		"64", "72", "80", "88", "96", "104", "112", "120",
// 	})

// 	equal(`{-01..5..2}`, nil, []s{"-01", "001", "003", "005"})
// 	equal(`{-1..05..2}`, nil, []s{"-1", "01", "03", "05"})
// 	equal(`{0000000001..3}`, nil, []s{"0000000001", "0000000002", "0000000003"})

// 	equal(`{a..b}{c..d}`, nil, []s{"ac", "ad", "bc", "bd"})

// 	equal(`{1..1}`, nil, []s{"1"})
// 	equal(`{1..2..9223372036854775807}`, nil, []s{"1"})
// 	equal(`{1..2..-9223372036854775808}`, nil, []s{"{1..2..-9223372036854775808}"})
// 	equal(`{0..9223372036854775807}`, nil, []s{"{0..9223372036854775807}"})

// 	equal(`{Z..a}`, &opt{AnyCharRange: true}, []s{"Z", "[", `\`, "]", "^", "_", "`", "a"})
// 	equal(`{中..丰}`, &opt{AnyCharRange: true}, []s{"中", "丮", `丯`, "丰"})
// 	equal(`{你x..好}`, &opt{AnyCharRange: true}, []s{"{你x..好}"})
// 	equal(`{你..好x}`, &opt{AnyCharRange: true}, []s{"{你..好x}"})

// 	equal(`\{a..b}`, nil, []s{`{a..b}`})
// 	equal(`\{a..b}`, &syntax.Parser{KeepEscape: true}, []s{`\{a..b}`})
// 	equal(`\{a..b}`, &syntax.Parser{IgnoreEscape: true}, []s{`\a`, `\b`})

// 	equal(`"{a..b}"`, nil, []s{`{a..b}`})
// 	equal(`"{a..b}"`, &syntax.Parser{KeepQuote: true}, []s{`"{a..b}"`})
// 	equal(`"{a..b}"`, &syntax.Parser{IgnoreQuote: true}, []s{`"a"`, `"b"`})

// 	equal(`{`, nil, []s{"{"})
// 	equal(`}`, nil, []s{"}"})
// 	equal(`{}`, nil, []s{"{}"})
// 	equal(`a{a{a,b}b`, nil, []s{"a{aab", "a{abb"})
// 	equal(`{a..b..c}`, nil, []s{"{a..b..c}"})
// 	equal(`{a..b..2..1}`, nil, []s{"{a..b..2..1}"})
// 	equal(`{a,a..b}`, nil, []s{"a", "a..b"})
// 	equal(`{a..b,a}`, nil, []s{"a..b", "a"})
// 	equal(`{..a..b}`, nil, []s{"{..a..b}"})
// 	equal(`{a.`, nil, []s{"{a."})
// 	equal(`a..b,a`, nil, []s{"a..b,a"})
// 	equal(`{a..b..1{a,b}}`, nil, []s{"{a..b..1a}", "{a..b..1b}"})
// 	equal(`{1..ab}`, nil, []s{"{1..ab}"})

// 	/** Empty expansion. **/

// 	braces.PrintTree("你{好{1..2..2},{a..b..2},在{那,学校,公司,家}}{,吗}")
// 	braces.MustCompile("aaaa").Equal(braces.MustCompile("aaaa"))
// 	braces.MustCompile("aaaa").Equal(braces.MustCompile("aaa"))
// 	braces.MustCompile("aaaa").Equal(braces.MustCompile("bbbb"))
// 	braces.MustCompile("{1..9}").Equal(braces.MustCompile("{1..9}"))
// 	braces.MustCompile("{1..9}").Equal(braces.MustCompile("{1..9..1}"))
// 	braces.MustCompile("{1..9}").Equal(braces.MustCompile("{1..9..2}"))
// 	braces.MustCompile("{1..2}").Equal(braces.MustCompile("{1,2}"))
// 	braces.MustCompile("a{1..2}b").Equal(braces.MustCompile("a{1,2}b"))
// 	braces.MustCompile("a{1..2}b").Equal(braces.MustCompile("a{1..2}"))
// }

func TestEmptyNode(t *testing.T) {
	equal := DefineExpand(t)

	equal("", E{""})
	equal("{,}", E{"", ""})
	equal("{,,}", E{"", "", ""})

	equal("{a,}", E{"a", ""})
	equal("{,a}", E{"", "a"})
	equal("{a,,}", E{"a", "", ""})
	equal("{,,a}", E{"", "", "a"})
	equal("{,a,}", E{"", "a", ""})
	equal("{a,,a}", E{"a", "", "a"})
}

func TestCharRange(t *testing.T) {
	equal := DefineExpand(t)

	equal("{a..b}", E{"a", "b"})
	equal("{a..c}", E{"a", "b", "c"})
	equal("{a..c..2}", E{"a", "c"})
	equal("{a..c..-2}", E{"a", "c"})
	equal("{a..c..1000}", E{"a"})

	equal("{b..a}", E{"b", "a"})
}

func TestEscape(t *testing.T) {
	equal := DefineExpand(t)

	equal(`\{a..b}`, E{"{a..b}"})
	equal(`\{a..b}`, E{`\a`, `\b`}, IgnoreEscape)

	equal(`{\a..b}`, E{"a", "b"})
	equal(`{\a..b}`, E{`{\a..b}`}, IgnoreEscape)

	equal(`{a\..b}`, E{"{a..b}"})
	equal(`{a\..b}`, E{`{a\..b}`}, IgnoreEscape)

	equal(`{a.\.b}`, E{"{a..b}"})
	equal(`{a.\.b}`, E{`{a.\.b}`}, IgnoreEscape)

	equal(`{a..\b}`, E{"a", "b"})
	equal(`{a..\b}`, E{`{a..\b}`}, IgnoreEscape)

	equal(`{a..b\}`, E{"{a..b}"})
	equal(`{a..b\}`, E{`{a..b\}`}, IgnoreEscape)

	equal(`{a..b\..3}`, E{"{a..b..3}"})
	equal(`{a..b\..3}`, E{`{a..b\..3}`}, IgnoreEscape)
}

package syntax_test

import (
	"testing"

	"github.com/pierre-primary/go-braces"
	"github.com/pierre-primary/go-braces/syntax"
)

//go:noinline
func discard(s string) {}

func BenchmarkExpand(t *testing.B) {
	expand := func(input string) func(t *testing.B) {
		return func(t *testing.B) {
			ast, _ := syntax.Parse(input, nil)
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
}

func TestExpand(t *testing.T) {
	define := func(t *testing.T, name string, p *syntax.Parser, group func(func(string, []string))) {
		t.Run(name, func(t *testing.T) {
			var buffer []byte
			group(func(input string, expected []string) {
				t.Run(input, func(t *testing.T) {
					var ast *syntax.Node
					ast, buffer = p.Parse(input, buffer)
					result, _ := ast.Expand(nil, buffer)
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
				})
			})
		})
	}
	type S = string

	define(t, "Literal", &syntax.Parser{}, func(equal func(string, []string)) {
		equal(`aaa/bbb/ccc`, []S{"aaa/bbb/ccc"})
	})

	define(t, "Alternate", &syntax.Parser{}, func(equal func(string, []string)) {
		equal(`a/{b,c}/d`, []S{"a/b/d", "a/c/d"})
	})

	define(t, "Range", &syntax.Parser{}, func(equal func(string, []string)) {

		equal(`{1..2}{2..1}`, []S{"12", "11", "22", "21"})
		equal(`a{1..2}b{2..1}c`, []S{"a1b2c", "a1b1c", "a2b2c", "a2b1c"})

		equal(`{0..8..2}`, []S{"0", "2", "4", "6", "8"})
		equal(`{1..8..-2}`, []S{"1", "3", "5", "7"})

		equal(`{-1..-2}{-2..-1}`, []S{"-1-2", "-1-1", "-2-2", "-2-1"})

		equal(`{-2..2..-1}`, []S{"-2", "-1", "0", "1", "2"})
		equal(`{2..-2..1}`, []S{"2", "1", "0", "-1", "-2"})

		equal(`{000..127..8}`, []S{
			"000", "008", "016", "024", "032", "040", "048", "056",
			"064", "072", "080", "088", "096", "104", "112", "120",
		})
		equal(`{00..127..8}`, []S{
			"00", "08", "16", "24", "32", "40", "48", "56",
			"64", "72", "80", "88", "96", "104", "112", "120",
		})

		equal(`{-01..5..2}`, []S{"-01", "001", "003", "005"})
		equal(`{-1..05..2}`, []S{"-1", "01", "03", "05"})
		equal(`{0000000001..3}`, []S{"0000000001", "0000000002", "0000000003"})

		equal(`{a..b}{c..d}`, []S{"ac", "ad", "bc", "bd"})

		equal(`{1..1}`, []S{"1"})
		equal(`{1..2..9223372036854775807}`, []S{"1"})
		equal(`{1..2..-9223372036854775808}`, []S{"{1..2..-9223372036854775808}"})
		equal(`{0..9223372036854775807}`, []S{"{0..9223372036854775807}"})
	})

	define(t, "AnyCharRange", &syntax.Parser{AnyCharRange: true}, func(equal func(string, []string)) {
		equal(`{Z..a}`, []S{"Z", "[", `\`, "]", "^", "_", "`", "a"})
		equal(`{中..丰}`, []S{"中", "丮", `丯`, "丰"})
		equal(`{你x..好}`, []S{"{你x..好}"})
		equal(`{你..好x}`, []S{"{你..好x}"})
	})

	define(t, "Escape", &syntax.Parser{}, func(equal func(string, []string)) {
		equal(`\{a..b}`, []S{`{a..b}`})
	})
	define(t, "KeepEscape", &syntax.Parser{KeepEscape: true}, func(equal func(string, []string)) {
		equal(`\{a..b}`, []S{`\{a..b}`})
	})
	define(t, "IgnoreEscape", &syntax.Parser{IgnoreEscape: true}, func(equal func(string, []string)) {
		equal(`\{a..b}`, []S{`\a`, `\b`})
	})
	define(t, "Quote", &syntax.Parser{}, func(equal func(string, []string)) {
		equal(`"{a..b}"`, []S{`{a..b}`})
	})
	define(t, "KeepEscape", &syntax.Parser{KeepQuote: true}, func(equal func(string, []string)) {
		equal(`"{a..b}"`, []S{`"{a..b}"`})
	})
	define(t, "IgnoreEscape", &syntax.Parser{IgnoreQuote: true}, func(equal func(string, []string)) {
		equal(`"{a..b}"`, []S{`"a"`, `"b"`})
	})

	define(t, "Rollback", &syntax.Parser{}, func(equal func(string, []string)) {
		equal(`{`, []S{"{"})
		equal(`}`, []S{"}"})
		equal(`{}`, []S{"{}"})
		equal(`a{a{a,b}b`, []S{"a{aab", "a{abb"})
		equal(`{a..b..c}`, []S{"{a..b..c}"})
		equal(`{a..b..2..1}`, []S{"{a..b..2..1}"})
		equal(`{a,a..b}`, []S{"a", "a..b"})
		equal(`{a..b,a}`, []S{"a..b", "a"})
		equal(`{..a..b}`, []S{"{..a..b}"})
		equal(`{a.`, []S{"{a."})
		equal(`a..b,a`, []S{"a..b,a"})
		equal(`{a..b..1{a,b}}`, []S{"{a..b..1a}", "{a..b..1b}"})
		equal(`{1..ab}`, []S{"{1..ab}"})
	})

	define(t, "xx", &syntax.Parser{}, func(equal func(string, []string)) {
		equal(`{,eno,thro,ro}ugh`, []S{"ugh", "enough", "through", "rough"})
		equal(`{,{,eno,thro,ro}ugh}{,out}`, []S{"", "out", "ugh", "ughout", "enough", "enoughout", "through", "throughout", "rough", "roughout"})
		equal(`{{,eno,thro,ro}ugh,}{,out}`, []S{"ugh", "ughout", "enough", "enoughout", "through", "throughout", "rough", "roughout", "", "out"})
		equal(`{,{,a,b}z}{,c}`, []S{"", "c", "z", "zc", "az", "azc", "bz", "bzc"})
		equal(`{,{,a,b}z}{c,}`, []S{"c", "", "zc", "z", "azc", "az", "bzc", "bz"})
		equal(`{,{,a,b}z}{,c,}`, []S{"", "c", "", "z", "zc", "z", "az", "azc", "az", "bz", "bzc", "bz"})
		equal(`{,{,a,b}z}{c,d}`, []S{"c", "d", "zc", "zd", "azc", "azd", "bzc", "bzd"})
		equal(`{{,a,b}z,}{,c}`, []S{"z", "zc", "az", "azc", "bz", "bzc", "", "c"})
		equal(`{,a{,b}z,}{,c}`, []S{"", "c", "az", "azc", "abz", "abzc", "", "c"})
		equal(`{,a{,b},}{,c}`, []S{"", "c", "a", "ac", "ab", "abc", "", "c"})
		equal(`{,a{,b}}{,c}`, []S{"", "c", "a", "ac", "ab", "abc"})
		equal(`{,b}{,d}`, []S{"", "d", "b", "bd"})
		equal(`{a,b}{,d}`, []S{"a", "ad", "b", "bd"})
		equal(`{,a}{z,c}`, []S{"z", "c", "az", "ac"})
		equal(`{,{a,}}{z,c}`, []S{"z", "c", "az", "ac", "z", "c"})
		equal(`{,{,a}}{z,c}`, []S{"z", "c", "z", "c", "az", "ac"})
		equal(`{,{,a},}{z,c}`, []S{"z", "c", "z", "c", "az", "ac", "z", "c"})
		equal(`{{,,a}}{z,c}`, []S{"{}z", "{}c", "{}z", "{}c", "{a}z", "{a}c"})
		equal(`{{,a},}{z,c}`, []S{"z", "c", "az", "ac", "z", "c"})
		equal(`{,,a}{z,c}`, []S{"z", "c", "z", "c", "az", "ac"})
		equal(`{,{,}}{z,c}`, []S{"z", "c", "z", "c", "z", "c"})
		equal(`{,{a,b}}{,c}`, []S{"", "c", "a", "ac", "b", "bc"})
		equal(`{,{a,}}{,c}`, []S{"", "c", "a", "ac", "", "c"})
		equal(`{,{,b}}{,c}`, []S{"", "c", "", "c", "b", "bc"})
		equal(`{,{,}}{,c}`, []S{"", "c", "", "c", "", "c"})
		equal(`{,a}{,c}`, []S{"", "c", "a", "ac"})
		equal(`{,{,a}b}`, []S{"", "b", "ab"})
		equal(`{,b}`, []S{"", "b"})
		equal(`{,b{,a}}`, []S{"", "b", "ba"})
		equal(`{b,{,a}}`, []S{"b", "", "a"})
		equal(`{,b}{,d}`, []S{"", "d", "b", "bd"})
		equal(`{a,b}{,d}`, []S{"a", "ad", "b", "bd"})
	})
	braces.PrintTree("你{好{1..2..2},{a..b..2},在{那,学校,公司,家}}{,吗}")
	braces.MustCompile("aaaa").Equal(braces.MustCompile("aaaa"))
	braces.MustCompile("aaaa").Equal(braces.MustCompile("aaa"))
	braces.MustCompile("aaaa").Equal(braces.MustCompile("bbbb"))
	braces.MustCompile("{1..9}").Equal(braces.MustCompile("{1..9}"))
	braces.MustCompile("{1..9}").Equal(braces.MustCompile("{1..9..1}"))
	braces.MustCompile("{1..9}").Equal(braces.MustCompile("{1..9..2}"))
	braces.MustCompile("{1..2}").Equal(braces.MustCompile("{1,2}"))
	braces.MustCompile("a{1..2}b").Equal(braces.MustCompile("a{1,2}b"))
	braces.MustCompile("a{1..2}b").Equal(braces.MustCompile("a{1..2}"))
}

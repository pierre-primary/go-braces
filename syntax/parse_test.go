package syntax_test

import (
	"testing"

	"github.com/pierre-primary/go-braces/syntax"
)

func BenchmarkParse(t *testing.B) {
	Parse := func(input string) func(t *testing.B) {
		return func(t *testing.B) {
			t.ReportAllocs()
			t.ResetTimer()
			for i := 0; i < t.N; i++ {
				syntax.Parse(input, nil)
			}
		}
	}
	t.Run("Literal", Parse("abcdefg"))
	t.Run("Alternate", Parse("{abc,def,ghi}"))
	t.Run("Mixed", Parse("aaa{abc,def,ghi}bbb"))
	t.Run("CharRange:[0-9]", Parse("{0..9}"))
	t.Run("CharRange:[a-z]", Parse("{a..z}"))
	t.Run("CharRange:[A-Z]", Parse("{A..Z}"))
	t.Run("IntRange:10", Parse("{1..10}"))
	t.Run("IntRange:100", Parse("{1..100}"))
	t.Run("Unicode", Parse("{你好吗,你在吗,你在哪}"))
}

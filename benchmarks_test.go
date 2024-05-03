package main

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkStringBuilderConcatenation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		for j := 0; j < 1000; j++ {
			sb.WriteString("h")
		}
		_ = sb.String()
	}
}

func BenchmarkStringConcatenation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var s string
		for j := 0; j < 1000; j++ {
			s += "h"
		}
	}
}

func BenchmarkFmtSprintfConcatenation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var s string
		for j := 0; j < 1000; j++ {
			s = fmt.Sprintf("%s%s", s, "h")
		}
	}
}

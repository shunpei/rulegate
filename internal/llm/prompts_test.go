package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadPrompts(t *testing.T) {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("cannot find project root")
		}
		dir = parent
	}

	path := filepath.Join(dir, "docs", "prompts.md")
	prompts, err := LoadPrompts(path)
	if err != nil {
		t.Fatalf("LoadPrompts: %v", err)
	}

	if prompts.RewriteSystem == "" {
		t.Error("RewriteSystem is empty")
	}
	if prompts.RewriteUser == "" {
		t.Error("RewriteUser is empty")
	}
	if prompts.AnswerSystem == "" {
		t.Error("AnswerSystem is empty")
	}
	if prompts.AnswerUser == "" {
		t.Error("AnswerUser is empty")
	}
}

func TestRenderTemplate(t *testing.T) {
	tmpl := "Hello {{name}}, question: {{question_ja}}"
	result := RenderTemplate(tmpl, map[string]string{
		"name":        "user",
		"question_ja": "テスト質問",
	})
	expected := "Hello user, question: テスト質問"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEnforceWordLimit(t *testing.T) {
	tests := []struct {
		input    string
		maxWords int
		wantMax  int
	}{
		{"short quote", 25, 2},
		{"one two three four five six", 3, 3},
		{"", 25, 0},
		{"a b c d e f g h i j k l m n o p q r s t u v w x y z", 25, 25},
	}

	for _, tt := range tests {
		result := enforceWordLimit(tt.input, tt.maxWords)
		words := strings.Fields(result)
		// When truncated, the last "word" may have "..." appended, but word count should still be <= maxWords+1.
		if len(words) > tt.maxWords+1 {
			t.Errorf("enforceWordLimit(%q, %d) = %q (%d words), want <= %d", tt.input, tt.maxWords, result, len(words), tt.maxWords)
		}
	}
}

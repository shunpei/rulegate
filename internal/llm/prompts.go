package llm

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// PromptTemplates holds parsed prompt templates from docs/prompts.md.
type PromptTemplates struct {
	RewriteSystem string
	RewriteUser   string
	AnswerSystem  string
	AnswerUser    string
}

// LoadPrompts parses the prompts.md file and extracts named templates.
// Expected format: ## template_name followed by a fenced code block.
func LoadPrompts(path string) (*PromptTemplates, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read prompts file: %w", err)
	}

	sections := parsePromptSections(string(data))

	get := func(name string) (string, error) {
		v, ok := sections[name]
		if !ok {
			return "", fmt.Errorf("prompt section %q not found in %s", name, path)
		}
		return v, nil
	}

	pt := &PromptTemplates{}
	if pt.RewriteSystem, err = get("query_rewrite_system"); err != nil {
		return nil, err
	}
	if pt.RewriteUser, err = get("query_rewrite_user"); err != nil {
		return nil, err
	}
	if pt.AnswerSystem, err = get("answer_system"); err != nil {
		return nil, err
	}
	if pt.AnswerUser, err = get("answer_user"); err != nil {
		return nil, err
	}

	return pt, nil
}

var sectionHeaderRe = regexp.MustCompile(`(?m)^## (.+)$`)

// parsePromptSections extracts named sections from a markdown file.
// Each section is a ## heading followed by a fenced code block.
func parsePromptSections(content string) map[string]string {
	sections := make(map[string]string)

	matches := sectionHeaderRe.FindAllStringSubmatchIndex(content, -1)
	for i, match := range matches {
		name := strings.TrimSpace(content[match[2]:match[3]])

		// Get the content between this header and the next one (or EOF).
		start := match[1]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		body := content[start:end]
		sections[name] = extractCodeBlock(body)
	}

	return sections
}

// extractCodeBlock extracts the content of the first fenced code block from text.
func extractCodeBlock(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	inBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inBlock {
				break
			}
			inBlock = true
			continue
		}
		if inBlock {
			result = append(result, line)
		}
	}
	return strings.TrimSpace(strings.Join(result, "\n"))
}

// RenderTemplate replaces {{key}} placeholders in a template string.
func RenderTemplate(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

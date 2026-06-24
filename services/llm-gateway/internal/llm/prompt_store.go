package llm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type PromptStore struct {
	baseDir string
}

func NewPromptStore(baseDir string) (*PromptStore, error) {
	if strings.TrimSpace(baseDir) == "" {
		return nil, fmt.Errorf("prompt base dir is empty")
	}
	if _, err := os.Stat(baseDir); err != nil {
		return nil, fmt.Errorf("stat prompt dir: %w", err)
	}
	return &PromptStore{baseDir: baseDir}, nil
}

func (s *PromptStore) Render(name string, variables map[string]any) (string, error) {
	path := filepath.Join(s.baseDir, normalizePromptName(name))
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt %s: %w", name, err)
	}

	tpl, err := template.New(name).Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("parse prompt %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, variables); err != nil {
		return "", fmt.Errorf("render prompt %s: %w", name, err)
	}
	return buf.String(), nil
}

func normalizePromptName(name string) string {
	if strings.HasSuffix(name, ".txt") {
		return name
	}
	return name + ".txt"
}

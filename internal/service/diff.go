package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func (s *SnippetService) DiffVersions(ctx context.Context, snippetID uuid.UUID, fromVersion, toVersion int) (string, error) {
	fromContent, fromVer, err := s.GetSnippetVersion(ctx, snippetID, fromVersion)
	if err != nil {
		return "", fmt.Errorf("failed to get version %d: %w", fromVersion, err)
	}

	toContent, toVer, err := s.GetSnippetVersion(ctx, snippetID, toVersion)
	if err != nil {
		return "", fmt.Errorf("failed to get version %d: %w", toVersion, err)
	}

	if fromVer.ContentHash == toVer.ContentHash {
		return "Versions are identical (no changes)", nil
	}

	dmp := diffmatchpatch.New()

	// Diff linha por linha usando LineDiff
	chars1, chars2, lineArray := dmp.DiffLinesToChars(string(fromContent), string(toContent))
	lineDiffs := dmp.DiffMain(chars1, chars2, false)
	finalDiffs := dmp.DiffCharsToLines(lineDiffs, lineArray)

	var result strings.Builder
	fmt.Fprintf(&result, "Comparing Version %d → Version %d\n", fromVersion, toVersion)
	fmt.Fprintf(&result, "─────────────────────────────────────────────────────────────────────────\n")
	fmt.Fprintf(&result, "Version %d: %s (%d bytes)\n", fromVersion, fromVer.CreatedAt.Format("2006-01-02 15:04:05"), fromVer.SizeBytes)
	fmt.Fprintf(&result, "Version %d: %s (%d bytes)\n", toVersion, toVer.CreatedAt.Format("2006-01-02 15:04:05"), toVer.SizeBytes)
	fmt.Fprintf(&result, "─────────────────────────────────────────────────────────────────────────\n\n")

	// Processa o diff linha por linha
	for _, diff := range finalDiffs {
		for line := range strings.SplitSeq(diff.Text, "\n") {
			if line == "" {
				continue
			}

			switch diff.Type {
			case diffmatchpatch.DiffInsert:
				// Verde para adições
				fmt.Fprintf(&result, "\033[32m+ %s\033[0m\n", line)
			case diffmatchpatch.DiffDelete:
				// Vermelho para remoções
				fmt.Fprintf(&result, "\033[31m- %s\033[0m\n", line)
			case diffmatchpatch.DiffEqual:
				// Sem cor para linhas iguais
				fmt.Fprintf(&result, "  %s\n", line)
			}
		}
	}

	return result.String(), nil
}

func (s *SnippetService) DiffStats(ctx context.Context, snippetID uuid.UUID, fromVersion, toVersion int) (int, int, error) {
	fromContent, _, err := s.GetSnippetVersion(ctx, snippetID, fromVersion)
	if err != nil {
		return 0, 0, err
	}

	toContent, _, err := s.GetSnippetVersion(ctx, snippetID, toVersion)
	if err != nil {
		return 0, 0, err
	}

	fromLines := len(strings.Split(string(fromContent), "\n"))
	toLines := len(strings.Split(string(toContent), "\n"))

	return fromLines, toLines, nil
}

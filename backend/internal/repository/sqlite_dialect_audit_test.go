package repository

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductionSQLUsesSQLiteDialect(t *testing.T) {
	backendRoot := filepath.Clean(filepath.Join("..", ".."))
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`github\.com/lib/pq`),
		regexp.MustCompile(`\bANY\s*\(`),
		regexp.MustCompile(`\bALL\s*\(`),
		regexp.MustCompile(`(?i)\bILIKE\b`),
		regexp.MustCompile(`(?i)::\s*(bigint|boolean|date|float[48]|int(?:eger)?|jsonb?|numeric|real|regclass|text|timestamp|uuid)\b`),
		regexp.MustCompile(`(?i)\bFOR\s+UPDATE\b`),
		regexp.MustCompile(`(?i)\bSKIP\s+LOCKED\b`),
		regexp.MustCompile(`(?i)\bDISTINCT\s+ON\b`),
		regexp.MustCompile(`(?i)\bNULLS\s+(FIRST|LAST)\b`),
		regexp.MustCompile(`(?i)\bDATE_TRUNC\s*\(`),
		regexp.MustCompile(`(?i)\bEXTRACT\s*\(`),
		regexp.MustCompile(`(?i)\bINTERVAL\s+'`),
		regexp.MustCompile(`(?i)\bjsonb_[a-z_]+\s*\(`),
		regexp.MustCompile(`(?i)\bpg_(try_)?advisory_(xact_)?(un)?lock\s*\(`),
	}

	var violations []string
	for _, relativeRoot := range []string{"internal", "cmd"} {
		root := filepath.Join(backendRoot, relativeRoot)
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "ent" || entry.Name() == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "wire_gen.go") {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = file.Close() }()

			scanner := bufio.NewScanner(file)
			for lineNumber := 1; scanner.Scan(); lineNumber++ {
				line := scanner.Text()
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "import (") {
					continue
				}
				if !strings.Contains(line, "`") && !strings.Contains(line, "\"") {
					continue
				}
				for _, pattern := range patterns {
					if match := pattern.FindString(line); match != "" {
						rel, _ := filepath.Rel(backendRoot, path)
						violations = append(violations, rel+":"+sqliteAuditItoa(lineNumber)+": "+match)
						break
					}
				}
			}
			return scanner.Err()
		})
		require.NoError(t, err)
	}

	sort.Strings(violations)
	require.Empty(t, violations, "PostgreSQL-only production SQL remains:\n%s", strings.Join(violations, "\n"))
}

func sqliteAuditItoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}

package repository

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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
		regexp.MustCompile(`(?i)\bCROSS\s+JOIN\s+LATERAL\b`),
		regexp.MustCompile(`(?i)\bFULL\s+OUTER\s+JOIN\b`),
		regexp.MustCompile(`(?i)\bGROUPING\s+SETS\b`),
		regexp.MustCompile(`(?i)\bPERCENTILE_CONT\s*\(`),
		regexp.MustCompile(`(?i)\bDATE_BIN\s*\(`),
		regexp.MustCompile(`(?i)\bMAKE_INTERVAL\s*\(`),
		regexp.MustCompile(`(?i)\bSPLIT_PART\s*\(`),
		regexp.MustCompile(`(?i)\bAT\s+TIME\s+ZONE\b`),
		regexp.MustCompile(`(?i)\bTIMESTAMPTZ\b`),
		regexp.MustCompile(`(?i)\bREGEXP_REPLACE\s*\(`),
		regexp.MustCompile(`(?i)\bUNNEST\s*\(`),
		regexp.MustCompile(`(?i)\bARRAY\s*\[`),
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

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return true
				}
				value, err := strconv.Unquote(literal.Value)
				if err != nil {
					return true
				}
				for _, pattern := range patterns {
					if match := pattern.FindString(value); match != "" {
						rel, _ := filepath.Rel(backendRoot, path)
						line := fset.Position(literal.Pos()).Line
						violations = append(violations, rel+":"+sqliteAuditItoa(line)+": "+match)
						break
					}
				}
				return true
			})
			return nil
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

package architecture_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDomainLayerDependencies verifies that domain layer does not import
// infrastructure packages (Kafka, SQL, HTTP, adapters, consumer).
// Verifies: Requirements 10.3, 10.4.
func TestDomainLayerDependencies(t *testing.T) {
	domainDir := "../../../internal/worker/domain"
	require.DirExists(t, domainDir, "domain directory should exist")

	violations := checkDirectoryForForbiddenImports(t, domainDir, []string{
		"github.com/segmentio/kafka-go",
		"database/sql",
		"net/http",
		"internal/worker/adapters",
		"internal/worker/adapters/inbound/messaging",
		"internal/worker/infrastructure",
		"github.com/lib/pq",
	})

	assert.Empty(t, violations, "domain layer should not import infrastructure packages")
}

// TestApplicationLayerDependencies verifies that application layer only imports
// domain and ports packages, not infrastructure implementations.
// Verifies: Requirements 10.3, 10.4.
func TestApplicationLayerDependencies(t *testing.T) {
	appDir := "../../../internal/worker/application"
	require.DirExists(t, appDir, "application directory should exist")

	// Application layer should only import domain and ports
	allowedPrefixes := []string{
		"github.com/fleet-platform/backend/internal/worker/domain",
		"github.com/fleet-platform/backend/internal/worker/ports",
		"context",
		"time",
		"errors",
		"fmt",
		"sync",
	}

	violations := checkDirectoryForUnexpectedImports(t, appDir, allowedPrefixes)
	assert.Empty(t, violations, "application layer should only import domain and ports packages")
}

// checkDirectoryForForbiddenImports scans all Go files in a directory and
// returns violations for forbidden imports.
func checkDirectoryForForbiddenImports(t *testing.T, dir string, forbiddenImports []string) []string {
	violations := []string{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			fileViolations := checkFileForForbiddenImports(t, path, forbiddenImports)
			violations = append(violations, fileViolations...)
		}

		return nil
	})
	require.NoError(t, err, "should be able to walk directory")

	return violations
}

// checkFileForForbiddenImports parses a single Go file and checks for forbidden imports.
func checkFileForForbiddenImports(t *testing.T, filePath string, forbiddenImports []string) []string {
	violations := []string{}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Logf("warning: could not parse file %s: %v", filePath, err)
		return violations
	}

	// Check each import
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		for _, forbidden := range forbiddenImports {
			if strings.Contains(importPath, forbidden) {
				violations = append(violations, fmt.Sprintf("%s imports %s", filePath, importPath))
			}
		}
	}

	return violations
}

// checkDirectoryForUnexpectedImports scans all Go files in a directory and
// returns violations for imports that are not in the allowed list.
func checkDirectoryForUnexpectedImports(t *testing.T, dir string, allowedPrefixes []string) []string {
	violations := []string{}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			fileViolations := checkFileForUnexpectedImports(t, path, allowedPrefixes)
			violations = append(violations, fileViolations...)
		}

		return nil
	})
	require.NoError(t, err, "should be able to walk directory")

	return violations
}

// checkFileForUnexpectedImports parses a single Go file and checks for imports
// that are not in the allowed list.
func checkFileForUnexpectedImports(t *testing.T, filePath string, allowedPrefixes []string) []string {
	violations := []string{}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		t.Logf("warning: could not parse file %s: %v", filePath, err)
		return violations
	}

	// Check each import
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		allowed := false

		// Check if import matches any allowed prefix
		for _, prefix := range allowedPrefixes {
			if strings.HasPrefix(importPath, prefix) {
				allowed = true
				break
			}
		}

		// Also allow standard library imports (no dots in first path component)
		if !allowed && !strings.Contains(strings.Split(importPath, "/")[0], ".") {
			allowed = true
		}

		if !allowed {
			violations = append(violations, fmt.Sprintf("%s imports unexpected package: %s", filePath, importPath))
		}
	}

	return violations
}

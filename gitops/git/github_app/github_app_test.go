package github_app

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestGetFilesToCommit(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "gitops-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test directory structure
	testDirs := []string{
		"buildkite-agents/ai-services/config",
		"buildkite-agents/airflow-dev/settings",
		"buildkite-agents/single-file-dir",
	}

	testFiles := []string{
		"buildkite-agents/ai-services/config/file1.yaml",
		"buildkite-agents/ai-services/config/file2.yaml",
		"buildkite-agents/airflow-dev/settings/settings.json",
		"buildkite-agents/single-file-dir/single.txt",
	}

	// Create directories
	for _, dir := range testDirs {
		err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create files with some content
	for _, file := range testFiles {
		err := os.WriteFile(
			filepath.Join(tmpDir, file),
			[]byte("test content"),
			0644,
		)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	tests := []struct {
		name          string
		inputPaths    []string
		expectedFiles []string
		expectedError bool
	}{
		{
			name: "Single directory with multiple files",
			inputPaths: []string{
				"buildkite-agents/ai-services",
			},
			expectedFiles: []string{
				"buildkite-agents/ai-services/config/file1.yaml",
				"buildkite-agents/ai-services/config/file2.yaml",
			},
			expectedError: false,
		},
		{
			name: "Multiple directories",
			inputPaths: []string{
				"buildkite-agents/ai-services",
				"buildkite-agents/airflow-dev",
			},
			expectedFiles: []string{
				"buildkite-agents/ai-services/config/file1.yaml",
				"buildkite-agents/ai-services/config/file2.yaml",
				"buildkite-agents/airflow-dev/settings/settings.json",
			},
			expectedError: false,
		},
		{
			name: "Single file",
			inputPaths: []string{
				"buildkite-agents/single-file-dir/single.txt",
			},
			expectedFiles: []string{
				"buildkite-agents/single-file-dir/single.txt",
			},
			expectedError: false,
		},
		{
			name: "Mixed files and directories",
			inputPaths: []string{
				"buildkite-agents/ai-services",
				"buildkite-agents/single-file-dir/single.txt",
			},
			expectedFiles: []string{
				"buildkite-agents/ai-services/config/file1.yaml",
				"buildkite-agents/ai-services/config/file2.yaml",
				"buildkite-agents/single-file-dir/single.txt",
			},
			expectedError: false,
		},
		{
			name: "Non-existent path",
			inputPaths: []string{
				"buildkite-agents/non-existent",
			},
			expectedFiles: nil,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileEntries, err := getFilesToCommit(tmpDir, tt.inputPaths)

			// Check error condition
			if (err != nil) != tt.expectedError {
				t.Errorf("getFilesToCommit() error = %v, expectedError %v", err, tt.expectedError)
				return
			}

			if tt.expectedError {
				return
			}

			// Get actual relative paths
			var actualFiles []string
			for _, entry := range fileEntries {
				actualFiles = append(actualFiles, entry.RelativePath)
			}

			// Sort both slices for comparison
			sort.Strings(actualFiles)
			sort.Strings(tt.expectedFiles)

			// Compare results
			if !reflect.DeepEqual(actualFiles, tt.expectedFiles) {
				t.Errorf("getFilesToCommit() got = %v, want %v", actualFiles, tt.expectedFiles)
			}

			// Verify that FullPath exists for each entry
			for _, entry := range fileEntries {
				if _, err := os.Stat(entry.FullPath); os.IsNotExist(err) {
					t.Errorf("FullPath %s does not exist", entry.FullPath)
				}
			}
		})
	}
}

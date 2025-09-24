package slack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/douhashi/soba/internal/config"
	"github.com/douhashi/soba/pkg/logging"
)

func TestTemplateManager_LoadTemplates(t *testing.T) {
	logger := logging.NewMockLogger()
	tm := NewTemplateManager(logger)

	// Create temporary template directory for testing
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, "templates", "slack")
	err := os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create test templates
	testTemplates := map[string]string{
		"notify.json": `{
			"blocks": [
				{
					"type": "section",
					"text": {
						"type": "mrkdwn",
						"text": "{{.Text}}"
					}
				}
			]
		}`,
		"phase_start.json": `{
			"blocks": [
				{
					"type": "header",
					"text": {
						"type": "plain_text",
						"text": "Phase: {{.Phase}} #{{.IssueNumber}}",
						"emoji": true
					}
				}
			]
		}`,
		"pr_merged.json": `{
			"blocks": [
				{
					"type": "header",
					"text": {
						"type": "plain_text",
						"text": "✅ PR #{{.PRNumber}} Merged",
						"emoji": true
					}
				}
			]
		}`,
		"error.json": `{
			"blocks": [
				{
					"type": "header",
					"text": {
						"type": "plain_text",
						"text": "❌ Error: {{.Title}}",
						"emoji": true
					}
				}
			]
		}`,
	}

	for filename, content := range testTemplates {
		writeErr := os.WriteFile(filepath.Join(templateDir, filename), []byte(content), 0644)
		require.NoError(t, writeErr)
	}

	// Change working directory to temp dir for testing
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test loading templates
	err = tm.LoadTemplates()
	assert.NoError(t, err)
}

func TestTemplateManager_RenderTemplate(t *testing.T) {
	logger := logging.NewMockLogger()
	tm := NewTemplateManager(logger)

	// Create temporary template directory for testing
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, "templates", "slack")
	err := os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create test template
	templateContent := `{
		"blocks": [
			{
				"type": "section",
				"text": {
					"type": "mrkdwn",
					"text": "{{.Text}}"
				}
			}
		]
	}`
	writeErr := os.WriteFile(filepath.Join(templateDir, "notify.json"), []byte(templateContent), 0644)
	require.NoError(t, writeErr)

	// Change working directory to temp dir for testing
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Load templates
	err = tm.LoadTemplates()
	require.NoError(t, err)

	// Test rendering
	data := NotifyData{Text: "Test message"}
	result, err := tm.RenderTemplate("notify", data)
	assert.NoError(t, err)

	// Verify the result is valid JSON
	var blockMsg BlockMessage
	err = json.Unmarshal(result, &blockMsg)
	assert.NoError(t, err)
	assert.Len(t, blockMsg.Blocks, 1)
}

func TestTemplateManager_RenderTemplate_NonExistentTemplate(t *testing.T) {
	logger := logging.NewMockLogger()
	tm := NewTemplateManager(logger)

	data := NotifyData{Text: "Test message"}
	_, err := tm.RenderTemplate("non_existent", data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template non_existent not found")
}

func TestTemplateManager_LoadTemplates_MissingDirectory(t *testing.T) {
	logger := logging.NewMockLogger()
	tm := NewTemplateManager(logger)

	// Create temporary directory but no templates subdirectory
	tempDir := t.TempDir()
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test loading templates from non-existent directory
	err = tm.LoadTemplates()
	assert.Error(t, err)
}

func TestTemplateManagerWithFS_LoadTemplates(t *testing.T) {
	logger := logging.NewMockLogger()

	// Test with actual embedded templates from config package
	tm := NewTemplateManagerWithFS(logger, config.GetSlackTemplatesFS())

	// Test loading embedded templates
	err := tm.LoadTemplates()
	assert.NoError(t, err, "Should load embedded templates successfully")
}

func TestTemplateManagerWithFS_RenderTemplate(t *testing.T) {
	logger := logging.NewMockLogger()

	// Use the real embedded FS from config package
	tm := NewTemplateManagerWithFS(logger, config.GetSlackTemplatesFS())

	// Load templates
	err := tm.LoadTemplates()
	require.NoError(t, err)

	// Test rendering different template types
	t.Run("Render notify template", func(t *testing.T) {
		data := NotifyData{Text: "Test notification"}
		result, err := tm.RenderTemplate("notify", data)
		assert.NoError(t, err)

		// Verify the result is valid JSON
		var blockMsg BlockMessage
		err = json.Unmarshal(result, &blockMsg)
		assert.NoError(t, err)
		assert.NotEmpty(t, blockMsg.Blocks)
	})

	t.Run("Render phase_start template", func(t *testing.T) {
		data := PhaseStartData{
			Phase:       "plan",
			IssueNumber: 123,
			IssueURL:    "https://github.com/test/repo/issues/123",
			Repository:  "test/repo",
		}
		result, err := tm.RenderTemplate("phase_start", data)
		assert.NoError(t, err)

		// Verify the result is valid JSON
		var blockMsg BlockMessage
		err = json.Unmarshal(result, &blockMsg)
		assert.NoError(t, err)
		assert.NotEmpty(t, blockMsg.Blocks)
	})

	t.Run("Render pr_merged template", func(t *testing.T) {
		data := PRMergedData{
			PRNumber:    456,
			IssueNumber: 123,
			PRURL:       "https://github.com/test/repo/pull/456",
			IssueURL:    "https://github.com/test/repo/issues/123",
		}
		result, err := tm.RenderTemplate("pr_merged", data)
		assert.NoError(t, err)

		// Verify the result is valid JSON
		var blockMsg BlockMessage
		err = json.Unmarshal(result, &blockMsg)
		assert.NoError(t, err)
		assert.NotEmpty(t, blockMsg.Blocks)
	})

	t.Run("Render error template", func(t *testing.T) {
		data := ErrorData{
			Title:        "Test Error",
			ErrorMessage: "Something went wrong",
		}
		result, err := tm.RenderTemplate("error", data)
		assert.NoError(t, err)

		// Verify the result is valid JSON
		var blockMsg BlockMessage
		err = json.Unmarshal(result, &blockMsg)
		assert.NoError(t, err)
		assert.NotEmpty(t, blockMsg.Blocks)
	})
}

func TestTemplateManagerWithFS_FallbackToFilesystem(t *testing.T) {
	logger := logging.NewMockLogger()

	// Create temporary template directory for testing
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, "templates", "slack")
	err := os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	// Create test template file
	templateContent := `{
		"blocks": [
			{
				"type": "section",
				"text": {
					"type": "mrkdwn",
					"text": "{{.Text}}"
				}
			}
		]
	}`
	writeErr := os.WriteFile(filepath.Join(templateDir, "notify.json"), []byte(templateContent), 0644)
	require.NoError(t, writeErr)

	// Change working directory to temp dir for testing
	oldDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(oldDir)
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test filesystem template manager (fallback mode)
	tm := NewTemplateManager(logger)
	err = tm.LoadTemplates()
	assert.NoError(t, err)

	// Test rendering
	data := NotifyData{Text: "Filesystem template test"}
	result, err := tm.RenderTemplate("notify", data)
	assert.NoError(t, err)

	// Verify the result is valid JSON
	var blockMsg BlockMessage
	err = json.Unmarshal(result, &blockMsg)
	assert.NoError(t, err)
	assert.Len(t, blockMsg.Blocks, 1)
}

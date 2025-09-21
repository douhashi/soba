package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStatusCmd(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "status command exists",
			args: []string{"--help"},
			want: "Display the current status of soba",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newStatusCmd()
			require.NotNil(t, cmd)

			// Set up buffer to capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			assert.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.want)
		})
	}
}

func TestStatusCommand_BasicAttributes(t *testing.T) {
	cmd := newStatusCmd()

	assert.Equal(t, "status", cmd.Use)
	assert.Equal(t, "Display the current status of soba", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

package execution

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func captureProcessStdout(t *testing.T) func() string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "stdout")
	require.NoError(t, err)
	previous := os.Stdout
	os.Stdout = file
	t.Cleanup(func() {
		os.Stdout = previous
		_ = file.Close()
	})
	return func() string {
		t.Helper()
		_, err := file.Seek(0, io.SeekStart)
		require.NoError(t, err)
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		return string(data)
	}
}

func TestDisplayerWithWriterDebug(t *testing.T) {
	stdout := captureProcessStdout(t)
	var output bytes.Buffer
	result := &ExecutionResult{
		ExecutionID:  "isolated-debug-result",
		PlaybookName: "writer-test.yml",
		Status:       "success",
	}
	NewDisplayerWithWriter(DisplayDebug, false, &output).DisplayExecution(result)

	assert.Contains(t, output.String(), "Execution Summary")
	assert.Contains(t, output.String(), "Complete Execution Data (JSON)")
	assert.Contains(t, output.String(), `"execution_id": "isolated-debug-result"`)
	assert.Empty(t, stdout(), "debug JSON must use the displayer's writer")
}

func TestSignalHandlerWithWriter(t *testing.T) {
	stdout := captureProcessStdout(t)
	input, err := os.CreateTemp(t.TempDir(), "stdin")
	require.NoError(t, err)
	t.Cleanup(func() { _ = input.Close() })
	_, err = input.WriteString("n\n")
	require.NoError(t, err)
	_, err = input.Seek(0, io.SeekStart)
	require.NoError(t, err)
	previous := os.Stdin
	os.Stdin = input
	t.Cleanup(func() { os.Stdin = previous })

	var output bytes.Buffer
	handler := NewSignalHandlerWithWriter(context.Background(), time.Second, &output)
	defer handler.Close()
	defer handler.Cancel()
	handler.RegisterCleanup(func() error { return errors.New("writer cleanup error") })
	assert.False(t, handler.promptSaveState())
	handler.executeCleanups()

	assert.Contains(t, output.String(), "Save state/audit data before exiting? [Y/n]: ")
	assert.Contains(t, output.String(), "State will be discarded")
	assert.Contains(t, output.String(), "writer cleanup error")
	assert.Empty(t, stdout(), "signal prompts and cleanup logs must use the handler's writer")
}

func TestJSONPrinterZeroValue(t *testing.T) {
	stdout := captureProcessStdout(t)
	var printer JSONPrinter
	printer.PrintJSON(&ExecutionResult{ExecutionID: "default-stdout"})
	assert.Contains(t, stdout(), `"execution_id": "default-stdout"`)
}

package internal_test

// Tests for GetEnvOrFile
import (
	"os"
	"path/filepath"
	"testing"

	i "github.com/jonhadfield/soba/internal"
	"github.com/stretchr/testify/require"
)

func TestGetEnvOrFile(t *testing.T) {
	// Clean up env after test
	envVar := "TEST_ENV_VAR"
	fileEnvVar := envVar + "_FILE"

	defer os.Unsetenv(envVar)
	defer os.Unsetenv(fileEnvVar)

	// 1. Env var set, should return value and true
	require.NoError(t, os.Setenv(envVar, "envvalue"))

	val, ok := i.GetEnvOrFile(envVar)
	require.True(t, ok)
	require.Equal(t, "envvalue", val)

	// 2. Env var set to empty, should return "" and true
	require.NoError(t, os.Setenv(envVar, ""))

	val, ok = i.GetEnvOrFile(envVar)
	require.True(t, ok)
	require.Equal(t, "", val)

	// 3. Env var unset, file var set, file exists
	require.NoError(t, os.Unsetenv(envVar))
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	require.NoError(t, os.WriteFile(tmpFile, []byte("filevalue\n"), 0o600))
	require.NoError(t, os.Setenv(fileEnvVar, tmpFile))

	val, ok = i.GetEnvOrFile(envVar)
	require.True(t, ok)
	require.Equal(t, "filevalue", val)

	// 4. Env var unset, file var set, file does not exist
	require.NoError(t, os.Setenv(fileEnvVar, "/nonexistent/file"))

	val, ok = i.GetEnvOrFile(envVar)
	require.False(t, ok)
	require.Equal(t, "", val)

	// 5. Neither env nor file var set
	require.NoError(t, os.Unsetenv(fileEnvVar))

	val, ok = i.GetEnvOrFile(envVar)
	require.False(t, ok)
	require.Equal(t, "", val)
}

package cmd

import (
	"gleaner/internal/projectpath"
	"gleaner/testHelpers"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// In order to test log level, we need to run the binary directly
// so we can test cobra's ability to parse flags property and set
// the log level before we enter any internal code
func TestLogLevel(t *testing.T) {
	minioHandle, err := testHelpers.NewMinioHandle("minio/minio:latest")
	require.NoError(t, err)

	url, _, err := minioHandle.ConnectionStrings()
	require.NoError(t, err)
	defer testcontainers.TerminateContainer(minioHandle.Container)

	t.Run("log level error", func(t *testing.T) {
		cmdStr := []string{"go", "run", projectpath.Root,
			"--cfg", "../testHelpers/sampleConfigs/justHu02.yaml",
			"--address", strings.Split(url, ":")[0],
			"--port", strings.Split(url, ":")[1],
			"--accesskey", minioHandle.Container.Username,
			"--secretkey", minioHandle.Container.Password,
			"--log-level", "ERROR",
			"--setup",
		}
		cmd := exec.Command(cmdStr[0], cmdStr[1:]...)
		bytes, err := cmd.CombinedOutput()
		require.NoError(t, err, "command failed with output: %s", string(bytes))
		require.Empty(t, string(bytes), "set log level error but output was not empty, signifying that other logs were produced or the command failed")
	})

	t.Run("log level info", func(t *testing.T) {
		cmdStr := []string{"go", "run", projectpath.Root,
			"--cfg", "../testHelpers/sampleConfigs/justHu02.yaml",
			"--address", strings.Split(url, ":")[0],
			"--port", strings.Split(url, ":")[1],
			"--accesskey", minioHandle.Container.Username,
			"--secretkey", minioHandle.Container.Password,
			"--log-level", "INFO",
			"--setup",
		}
		cmd := exec.Command(cmdStr[0], cmdStr[1:]...)
		bytes, err := cmd.CombinedOutput()
		require.NoError(t, err, "command failed with output: %s", string(bytes))
		require.NotEmpty(t, string(bytes))
		require.Contains(t, string(bytes), "info", "set log level info but info was never found in output")
	})

}

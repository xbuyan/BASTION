package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/google/uuid"
	"github.com/lucciano/bastion/internal/models"
	"go.uber.org/zap"
)

// Executor defines the code execution interface.
type Executor interface {
	Execute(ctx context.Context, userCode, testCode string) (*models.ExecutionResult, error)
}

// ─── Docker Executor ──────────────────────────────────────────────────────────

// DockerExecutor runs Go code in isolated Docker containers.
type DockerExecutor struct {
	client *client.Client
	logger *zap.Logger
}

func NewDockerExecutor(logger *zap.Logger) *DockerExecutor {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logger.Warn("docker client init failed, falling back to local", zap.Error(err))
		return nil
	}
	return &DockerExecutor{client: cli, logger: logger}
}

// Execute runs user code in a sandboxed Docker container.
func (e *DockerExecutor) Execute(ctx context.Context, userCode, testCode string) (*models.ExecutionResult, error) {
	start := time.Now()

	// Build the combined Go file
	combined := userCode
	if testCode != "" {
		combined += "\n\n" + testCode
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(combined))

	script := fmt.Sprintf(`
set -e
mkdir -p /tmp/prds
echo '%s' | base64 -d > /tmp/prds/main.go
cd /tmp/prds
go mod init prdsexec 2>/dev/null || true
go run main.go 2>&1
`, encoded)

	containerName := "prds-exec-" + uuid.NewString()[:8]

	resp, err := e.client.ContainerCreate(ctx, &container.Config{
		Image:      "golang:1.21-alpine",
		Cmd:        []string{"sh", "-c", script},
		WorkingDir: "/tmp/prds",
	}, &container.HostConfig{
		Resources: container.Resources{
			Memory:   256 * 1024 * 1024, // 256MB
			NanoCPUs: 1e9,               // 1 CPU
		},
		ReadonlyRootfs: false,
		NetworkMode:    "none",
		AutoRemove:     true,
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,size=64m",
		},
	}, nil, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("container create: %w", err)
	}

	execCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := e.client.ContainerStart(execCtx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return nil, fmt.Errorf("container start: %w", err)
	}

	// Wait for completion
	statusCh, errCh := e.client.ContainerWait(execCtx, resp.ID, container.WaitConditionNotRunning)
	var exitCode int
	select {
	case err := <-errCh:
		if err != nil {
			return &models.ExecutionResult{
				Error:    "execution timed out or failed",
				ExitCode: 124,
				Duration: time.Since(start),
			}, nil
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	}

	// Collect logs
	logReader, err := e.client.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	var stdout, stderr string
	if err == nil {
		defer logReader.Close()
		var buf bytes.Buffer
		io.Copy(&buf, logReader)
		stdout = buf.String()
		// Docker multiplexes stdout/stderr; strip 8-byte headers
		if len(stdout) > 8 {
			stdout = stdout[8:]
		}
	}

	return &models.ExecutionResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: time.Since(start),
		Passed:   exitCode == 0,
	}, nil
}

// ─── Local Executor (fallback when Docker not available) ─────────────────────

// LocalExecutor runs code as a subprocess (dev only, not sandboxed).
type LocalExecutor struct {
	logger *zap.Logger
}

func NewLocalExecutor(logger *zap.Logger) *LocalExecutor {
	return &LocalExecutor{logger: logger}
}

func (e *LocalExecutor) Execute(ctx context.Context, userCode, testCode string) (*models.ExecutionResult, error) {
	start := time.Now()

	dir, err := os.MkdirTemp("", "prds-exec-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	combined := userCode
	if testCode != "" {
		combined += "\n\n" + testCode
	}

	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte(combined), 0600); err != nil {
		return nil, err
	}

	// go mod init
	cmd := exec.CommandContext(ctx, "go", "mod", "init", "prdsexec")
	cmd.Dir = dir
	cmd.Run()

	// go run
	execCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	runCmd := exec.CommandContext(execCtx, "go", "run", "main.go")
	runCmd.Dir = dir
	var outBuf, errBuf bytes.Buffer
	runCmd.Stdout = &outBuf
	runCmd.Stderr = &errBuf

	runErr := runCmd.Run()
	exitCode := 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return &models.ExecutionResult{
		Stdout:   outBuf.String(),
		Stderr:   errBuf.String(),
		ExitCode: exitCode,
		Duration: time.Since(start),
		Passed:   exitCode == 0,
	}, nil
}

// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build packaging

package packaging

import (
	"archive/tar"
	"bytes"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcexec "github.com/testcontainers/testcontainers-go/exec"
	"github.com/testcontainers/testcontainers-go/wait"
)

//go:embed images/Dockerfile.*
var images embed.FS

const (
	// imageRepo is the repository of the images built from images/.
	imageRepo = "bdot-pkgtest"
	// pkgDir is where packages are copied inside the container. It must not
	// be /tmp, which systemd replaces with a tmpfs on some distros.
	pkgDir = "/pkgtest"
	// execTimeout bounds a single command run inside a container.
	execTimeout = 5 * time.Minute
)

// buildImage builds the systemd-enabled image for a distro and returns its
// tag. The image is kept so that later runs reuse the Docker layer cache.
func buildImage(t *testing.T, d distro) string {
	t.Helper()

	dockerfile, err := images.ReadFile(path.Join("images", d.Dockerfile))
	require.NoError(t, err)

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "Dockerfile", Mode: 0o644, Size: int64(len(dockerfile))}))
	_, err = tw.Write(dockerfile)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	provider, err := testcontainers.NewDockerProvider()
	require.NoError(t, err)
	defer provider.Close()

	var buildLog bytes.Buffer
	baseImage := d.BaseImage
	pull := os.Getenv("BDOT_PKGTEST_PULL") != ""
	start := time.Now()
	tag, err := provider.BuildImage(t.Context(), &testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			ContextArchive: bytes.NewReader(buf.Bytes()),
			BuildArgs:      map[string]*string{"BASE_IMAGE": &baseImage},
			Repo:           imageRepo,
			Tag:            d.Name,
			KeepImage:      true,
			BuildLogWriter: &buildLog,
			BuildOptionsModifier: func(o *client.ImageBuildOptions) {
				o.PullParent = pull
			},
		},
	})
	if err != nil {
		t.Logf("image build log:\n%s", buildLog.String())
		msg := "build image from images/" + d.Dockerfile
		if strings.Contains(buildLog.String()+err.Error(), "x86-64-v") {
			msg += " (the base image requires a newer x86-64 microarchitecture level, e.g. x86-64-v3/AVX2 for EL10, than this host provides)"
		}
		require.NoError(t, err, msg)
	}
	t.Logf("built image %s from %s in %s", tag, d.BaseImage, time.Since(start).Round(time.Millisecond))
	return tag
}

// box is a running systemd container that a scenario operates on.
type box struct {
	t   *testing.T
	ctr *testcontainers.DockerContainer
	env *suiteEnv
}

// startBox starts a fresh container from image and waits for systemd to
// finish booting. The container is removed when the test ends. When the test
// fails, diagnostics are logged before the container is removed.
func startBox(t *testing.T, env *suiteEnv, image string) *box {
	t.Helper()

	ctr, err := testcontainers.Run(t.Context(), image,
		testcontainers.WithHostConfigModifier(func(hc *container.HostConfig) {
			// systemd needs a writable cgroupfs, which Docker only mounts
			// for privileged containers. A private cgroup namespace keeps
			// the container out of the host's cgroup tree.
			hc.Privileged = true
			hc.CgroupnsMode = container.CgroupnsModePrivate
		}),
		testcontainers.WithTmpfs(map[string]string{
			"/run":      "rw,nosuid,nodev,mode=755",
			"/run/lock": "rw,nosuid,nodev,noexec",
		}),
		testcontainers.WithWaitStrategyAndDeadline(2*time.Minute, &systemdReady{}),
	)
	// Cleanups run last in, first out: diagnostics run before the
	// container is terminated.
	if os.Getenv("BDOT_PKGTEST_KEEP") == "" {
		testcontainers.CleanupContainer(t, ctr, testcontainers.StopTimeout(time.Second))
	} else if ctr != nil {
		t.Logf("keeping container %s (BDOT_PKGTEST_KEEP is set)", ctr.GetContainerID())
	}
	b := &box{t: t, ctr: ctr, env: env}
	t.Cleanup(func() {
		if t.Failed() && ctr != nil {
			b.dumpDiagnostics()
		}
	})
	require.NoError(t, err, "start container from %s", image)
	return b
}

// execResult is the outcome of a command run inside a container.
type execResult struct {
	code   int
	stdout string
	stderr string
}

func (r execResult) String() string {
	return fmt.Sprintf("exit code: %d\nstdout:\n%s\nstderr:\n%s", r.code, r.stdout, r.stderr)
}

// exec runs a shell script inside the container.
func (b *box) exec(ctx context.Context, script string) (execResult, error) {
	code, r, err := b.ctr.Exec(ctx, []string{"/bin/sh", "-c", script},
		tcexec.WithEnv([]string{"DEBIAN_FRONTEND=noninteractive", "LC_ALL=C"}))
	if err != nil {
		return execResult{}, fmt.Errorf("exec %q: %w", script, err)
	}
	var stdout, stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, r); err != nil {
		return execResult{}, fmt.Errorf("read output of %q: %w", script, err)
	}
	return execResult{code: code, stdout: stdout.String(), stderr: stderr.String()}, nil
}

// run runs a shell script and returns the result, whatever its exit code.
func (b *box) run(script string) execResult {
	b.t.Helper()
	ctx, cancel := context.WithTimeout(b.t.Context(), execTimeout)
	defer cancel()
	res, err := b.exec(ctx, script)
	require.NoError(b.t, err)
	return res
}

// mustRun runs a shell script, fails the test when it exits non-zero, and
// returns its stdout with surrounding whitespace removed.
func (b *box) mustRun(script string) string {
	b.t.Helper()
	res := b.run(script)
	require.Zero(b.t, res.code, "command failed: %s\n%s", script, res)
	return strings.TrimSpace(res.stdout)
}

// copyPackage streams a host file into pkgDir and returns its path inside
// the container. The file is streamed because testcontainers' copy helpers
// buffer the whole (large) package in memory.
func (b *box) copyPackage(hostPath string) string {
	b.t.Helper()

	f, err := os.Open(hostPath) // #nosec G304 -- path comes from the dist dir or the release cache
	require.NoError(b.t, err)
	defer f.Close()
	fi, err := f.Stat()
	require.NoError(b.t, err)

	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		hdr := &tar.Header{Name: filepath.Base(hostPath), Mode: 0o644, Size: fi.Size(), ModTime: fi.ModTime()}
		if err := tw.WriteHeader(hdr); err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(tw, f); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.CloseWithError(tw.Close())
	}()

	_, err = b.env.docker.CopyToContainer(b.t.Context(), b.ctr.GetContainerID(), client.CopyToContainerOptions{
		DestinationPath: pkgDir,
		Content:         pr,
	})
	// Unblock the writer if the copy failed before reading everything.
	_ = pr.CloseWithError(io.ErrClosedPipe)
	require.NoError(b.t, err, "copy %s into container", hostPath)
	return pkgDir + "/" + filepath.Base(hostPath)
}

// dumpDiagnostics logs the state of the container. It runs during test
// cleanup, after t.Context() is cancelled, so it uses its own context.
func (b *box) dumpDiagnostics() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var sb strings.Builder
	sb.WriteString("=== diagnostics ===\n")
	if logs, err := b.ctr.Logs(ctx); err == nil {
		out, _ := io.ReadAll(logs)
		_ = logs.Close()
		fmt.Fprintf(&sb, "--- container logs ---\n%s\n", out)
	}
	for _, script := range []string{
		"systemctl status " + packageName + " --no-pager -l",
		"journalctl -u " + packageName + " --no-pager -n 200",
		"tail -n 100 " + installDir + "/log/collector.log",
		"ls -la " + installDir,
		"systemctl --failed --no-pager",
	} {
		res, err := b.exec(ctx, script+" 2>&1")
		if err != nil {
			fmt.Fprintf(&sb, "--- %s ---\nerror: %v\n", script, err)
			continue
		}
		fmt.Fprintf(&sb, "--- %s (exit %d) ---\n%s\n", script, res.code, res.stdout)
	}
	b.t.Log(sb.String())
}

// systemdReady waits until systemd inside the container has finished
// booting. A "degraded" system counts as ready: some units fail in
// containers without affecting the package under test. The deadline is set
// by the caller, see startBox. It must be used as a pointer: testcontainers
// calls reflect.Value.IsNil on wait strategies.
type systemdReady struct{}

var _ wait.Strategy = (*systemdReady)(nil)

func (*systemdReady) String() string { return "systemd is running" }

func (*systemdReady) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	last := ""
	for {
		if state, err := target.State(ctx); err == nil && !state.Running {
			return fmt.Errorf("container exited with code %d; systemd failed to boot (is Docker using cgroup v2?)", state.ExitCode)
		}
		if _, r, err := target.Exec(ctx, []string{"systemctl", "is-system-running", "--wait"}, tcexec.Multiplexed()); err == nil {
			out, _ := io.ReadAll(r)
			last = strings.TrimSpace(string(out))
			switch last {
			case "running", "degraded":
				return nil
			case "maintenance", "stopping", "offline":
				return fmt.Errorf("systemd is in state %q", last)
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("systemd not ready, last state %q: %w", last, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

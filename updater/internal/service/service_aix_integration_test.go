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

//go:build aix && integration

package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestAIXServiceLifecycle drives a real SRC subsystem, under test names so it
// can't collide with an installed collector. Run as root.
func TestAIXServiceLifecycle(t *testing.T) {
	installDir := t.TempDir()
	// Stand-in collector: records its arguments and environment, then waits to be stopped.
	script := "#!/usr/bin/sh\necho \"$@\" > \"$BINDPLANE_COLLECTOR_HOME/args.out\"\nenv > \"$BINDPLANE_COLLECTOR_HOME/env.out\"\nexec /usr/bin/sleep 600\n"
	require.NoError(t, os.WriteFile(filepath.Join(installDir, "observiq-otel-collector"), []byte(script), 0700))

	envFile := filepath.Join(installDir, "collector.env")
	require.NoError(t, os.WriteFile(envFile, []byte("BINDPLANE_COLLECTOR_HOME="+installDir+" BINDPLANE_COLLECTOR_STORAGE="+filepath.Join(installDir, "storage")+"\n"), 0600))

	a := aixService{
		src:    srcService{subsystem: "bp1177test", group: "bp1177tg", installDir: installDir, envFile: envFile},
		poller: newSRCPoller("bp1177test"),
		logger: zaptest.NewLogger(t),
	}
	t.Cleanup(func() {
		_ = exec.Command("stopsrc", "-s", "bp1177test").Run()
		_ = exec.Command("rmitab", "bp1177tg").Run()
		_ = exec.Command("rmssys", "-s", "bp1177test").Run()
	})

	active := func() bool {
		out, _ := exec.Command("lssrc", "-s", "bp1177test").Output()
		return srcActive(out)
	}

	require.NoError(t, a.install())
	inittab, err := exec.Command("lsitab", "bp1177tg").Output()
	require.NoError(t, err)
	require.Contains(t, string(inittab), "startsrc -g bp1177tg")
	require.False(t, active())

	require.NoError(t, a.Start())
	require.True(t, active(), "Start returns once the subsystem is active")
	require.Eventually(t, func() bool {
		_, err := os.Stat(filepath.Join(installDir, "env.out"))
		return err == nil
	}, 10*time.Second, 200*time.Millisecond)
	args, err := os.ReadFile(filepath.Join(installDir, "args.out"))
	require.NoError(t, err)
	require.Equal(t, "--config "+filepath.Join(installDir, "config.yaml")+" --manager "+filepath.Join(installDir, "manager.yaml")+" --logging "+filepath.Join(installDir, "logging.yaml"), strings.TrimSpace(string(args)))
	env, err := os.ReadFile(filepath.Join(installDir, "env.out"))
	require.NoError(t, err)
	require.Contains(t, string(env), "BINDPLANE_COLLECTOR_STORAGE="+filepath.Join(installDir, "storage"))

	require.NoError(t, a.Stop())
	require.False(t, active(), "Stop returns once the subsystem is inoperative")

	// Run the inittab command the way init does at boot, through a shell.
	require.NoError(t, os.Remove(filepath.Join(installDir, "env.out")))
	fields := strings.SplitN(strings.TrimSpace(string(inittab)), ":", 4)
	require.Len(t, fields, 4)
	require.NoError(t, exec.Command("/usr/bin/sh", "-c", fields[3]).Run())
	require.Eventually(t, active, 10*time.Second, 200*time.Millisecond)
	require.Eventually(t, func() bool {
		env, err := os.ReadFile(filepath.Join(installDir, "env.out"))
		return err == nil && strings.Contains(string(env), "BINDPLANE_COLLECTOR_HOME="+installDir)
	}, 10*time.Second, 200*time.Millisecond)
	require.NoError(t, a.Stop())
	require.False(t, active(), "Stop returns once the subsystem is inoperative")

	// The updater stops the collector before Update, and rollback runs Update again.
	require.NoError(t, a.Update())
	require.NoError(t, a.Update())

	require.NoError(t, a.Start())
	require.True(t, active(), "Start returns once the subsystem is active")
	require.NoError(t, a.uninstall())
	require.Error(t, exec.Command("lssrc", "-s", "bp1177test").Run())
	require.Error(t, exec.Command("lsitab", "bp1177tg").Run())
}

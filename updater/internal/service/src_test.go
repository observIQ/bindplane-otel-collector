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

package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSRCArgs(t *testing.T) {
	s := srcService{
		subsystem:  "observiq-otel-collector",
		group:      "oiqcollector",
		installDir: "/opt/observiq-otel-collector",
		envFile:    "/etc/observiq-otel-collector.env",
	}

	t.Run("mkssys runs the collector as root with absolute config paths", func(t *testing.T) {
		require.Equal(t, []string{
			"-s", "observiq-otel-collector",
			"-G", "oiqcollector",
			"-p", "/opt/observiq-otel-collector/observiq-otel-collector",
			"-u", "0",
			"-S", "-n", "15", "-f", "9", "-R",
			"-a", "--config /opt/observiq-otel-collector/config.yaml --manager /opt/observiq-otel-collector/manager.yaml --logging /opt/observiq-otel-collector/logging.yaml",
		}, s.mkssysArgs())
	})

	t.Run("startsrc starts the group with the env file's variables", func(t *testing.T) {
		require.Equal(t, []string{"-g", "oiqcollector", "-e", "A=1 B=/x"}, s.startsrcArgs([]byte("A=1 B=/x\n")))
	})

	t.Run("startsrc without an env file passes no environment", func(t *testing.T) {
		require.Equal(t, []string{"-g", "oiqcollector"}, s.startsrcArgs(nil))
	})

	t.Run("inittab reads the env file at boot", func(t *testing.T) {
		require.Equal(t,
			`oiqcollector:23456789:once:startsrc -g oiqcollector -e "$(cat /etc/observiq-otel-collector.env)"`,
			s.inittabEntry())
	})
}

func TestSRCActive(t *testing.T) {
	header := "Subsystem         Group            PID          Status \n"
	require.True(t, srcActive([]byte(header+" observiq-otel-collector oiqcollector     11272516     active\n")))
	require.False(t, srcActive([]byte(header+" observiq-otel-collector oiqcollector                  inoperative\n")))
	require.True(t, srcActive([]byte(header+" observiq-otel-collector oiqcollector     11272516     stopping\n")))
	require.False(t, srcActive(nil))
}

// fakeLssrc returns each output in turn, repeating the last one, and counts calls.
type fakeLssrc struct {
	outs  []string
	err   error
	calls int
}

func (f *fakeLssrc) lssrc() ([]byte, error) {
	out := f.outs[min(f.calls, len(f.outs)-1)]
	f.calls++
	return []byte(out), f.err
}

// fakeClock only moves when sleep is called, so waits finish instantly.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time        { return c.t }
func (c *fakeClock) sleep(d time.Duration) { c.t = c.t.Add(d) }

func TestSRCPollerWaitFor(t *testing.T) {
	header := "Subsystem         Group            PID          Status \n"
	active := header + " observiq-otel-collector oiqcollector     11272516     active\n"
	stopping := header + " observiq-otel-collector oiqcollector     11272516     stopping\n"
	inoperative := header + " observiq-otel-collector oiqcollector                  inoperative\n"

	newPoller := func(l *fakeLssrc) (srcPoller, *fakeClock) {
		c := &fakeClock{t: time.Unix(0, 0)}
		return srcPoller{lssrc: l.lssrc, now: c.now, sleep: c.sleep}, c
	}

	t.Run("returns once the subsystem reaches the status", func(t *testing.T) {
		l := &fakeLssrc{outs: []string{stopping, stopping, inoperative}}
		p, c := newPoller(l)
		require.NoError(t, p.waitFor("inoperative", 120*time.Second))
		require.Equal(t, 3, l.calls)
		require.Equal(t, 2*srcPollInterval, c.t.Sub(time.Unix(0, 0)))
	})

	t.Run("returns at once when the subsystem already has the status", func(t *testing.T) {
		l := &fakeLssrc{outs: []string{active}}
		p, _ := newPoller(l)
		require.NoError(t, p.waitFor("active", 120*time.Second))
		require.Equal(t, 1, l.calls)
	})

	t.Run("times out with the last status seen", func(t *testing.T) {
		l := &fakeLssrc{outs: []string{stopping}}
		p, c := newPoller(l)
		err := p.waitFor("inoperative", 120*time.Second)
		require.ErrorContains(t, err, `did not become inoperative within 2m0s`)
		require.ErrorContains(t, err, `last status "stopping"`)
		require.NotContains(t, err.Error(), "%!")
		require.Equal(t, 120*time.Second, c.t.Sub(time.Unix(0, 0)))
	})

	t.Run("keeps polling through lssrc errors and reports the last one", func(t *testing.T) {
		l := &fakeLssrc{outs: []string{""}, err: errors.New("lssrc failed")}
		p, _ := newPoller(l)
		require.ErrorContains(t, p.waitFor("active", 3*time.Second), "lssrc failed")
		require.Equal(t, 4, l.calls)
	})
}

func TestSRCStatus(t *testing.T) {
	header := "Subsystem         Group            PID          Status \n"
	require.Equal(t, "active", srcStatus([]byte(header+" observiq-otel-collector oiqcollector     11272516     active\n")))
	require.Equal(t, "inoperative", srcStatus([]byte(header+" observiq-otel-collector oiqcollector                  inoperative\n")))
	require.Equal(t, "", srcStatus([]byte(header)))
	require.Equal(t, "", srcStatus(nil))
}

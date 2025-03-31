//go:build windows
// +build windows

package etw

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	AutologgerPath = `HKLM\System\CurrentControlSet\Control\WMI\Autologger`
	regExe         = `C:\Windows\System32\reg.exe`

	regDword  = "REG_DWORD"
	regQword  = "REG_QWORD"
	regSz     = "REG_SZ"
	regBinary = "REG_BINARY"
)

func hexStr(i interface{}) string {
	return fmt.Sprintf("0x%x", i)
}

type AutoLogger struct {
	Name        string
	Guid        string
	LogFileMode uint32
	BufferSize  uint32
	ClockType   uint32
	// MaxFileSize is the maximum file size of the log file, in megabytes.
	// If a real time session with RealtimePersistence this is the maximum file size of the backup file.
	// If not set the default is 100MB, to specify no limit this parameter must be 0 in the registry.
	// But here 0 means that we want don't want to configure MaxFileSize, if we want to set it, this
	// member needs to be explicitely set > 0
	MaxFileSize uint32
}

func (a *AutoLogger) Path() string {
	return fmt.Sprintf(`%s\%s`, strings.TrimRight(AutologgerPath, `\`), strings.TrimLeft(a.Name, `\`))
}

func (a *AutoLogger) Create() (err error) {
	sargs := [][]string{
		// ETWtrace parameters
		{a.Path(), "GUID", regSz, a.Guid},
		{a.Path(), "Start", regDword, "0x1"},
		{a.Path(), "LogFileMode", regDword, hexStr(a.LogFileMode)},
		// ETWevent can be up to 64KB so buffer needs to be at least this size
		{a.Path(), "BufferSize", regDword, hexStr(a.BufferSize)},
		{a.Path(), "ClockType", regDword, hexStr(a.ClockType)},
	}

	if a.MaxFileSize > 0 {
		sargs = append(sargs, []string{a.Path(), "MaxFileSize", regDword, hexStr(a.MaxFileSize)})
	}

	for _, args := range sargs {
		if err = regAddValue(args[0], args[1], args[2], args[3]); err != nil {
			return
		}
	}

	return
}

func binaryFilter(filter []uint16) (f string, err error) {
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.LittleEndian, filter); err != nil {
		return
	}
	return hex.EncodeToString(buf.Bytes()), nil
}

func (a *AutoLogger) EnableProvider(p Provider) (err error) {
	path := fmt.Sprintf(`%s\%s`, a.Path(), p.GUID)

	sargs := [][]string{}

	// ETWtrace parameters
	if p.Name != "" {
		sargs = append(sargs, []string{path, "ProviderName", regSz, p.Name})
	}

	sargs = append(sargs, []string{path, "Enabled", regDword, "0x1"})
	sargs = append(sargs, []string{path, "EnableLevel", regDword, hexStr(p.EnableLevel)})
	sargs = append(sargs, []string{path, "MatchAnyKeyword", regQword, hexStr(p.MatchAnyKeyword)})

	if p.MatchAnyKeyword != 0 {
		sargs = append(sargs, []string{path, "MatchAllKeyword", regQword, hexStr(p.MatchAllKeyword)})
	}

	// enable event filtering
	if len(p.Filter) > 0 {
		var binFilter string
		if binFilter, err = binaryFilter(p.Filter); err != nil {
			return fmt.Errorf("failed to create binary filter: %w", err)
		}
		filtersPath := filepath.Join(path, "Filters")
		sargs = append(sargs, []string{filtersPath, "Enabled", regDword, "0x1"})
		sargs = append(sargs, []string{filtersPath, "EventIdFilterIn", regDword, "0x1"})
		sargs = append(sargs, []string{filtersPath, "EventIds", regBinary, binFilter})
	}

	// executing commands
	for _, args := range sargs {
		if err = regAddValue(args[0], args[1], args[2], args[3]); err != nil {
			return
		}
	}

	return
}

func (a *AutoLogger) Exists() bool {
	return execute(regExe, "QUERY", a.Path()) == nil
}

func (a *AutoLogger) Delete() error {
	return execute(regExe, "DELETE", a.Path(), "/f")
}

func execute(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if out, err := exec.CommandContext(ctx, name, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("err:%s out:%s", err, string(out))
	}
	return nil
}

func regAddValue(path, valueName, valueType, value string) error {
	return execute(regExe, "ADD", path, "/v", valueName, "/t", valueType, "/d", value, "/f")
}

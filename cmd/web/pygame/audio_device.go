package pygame

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// Ebitengine treats an asynchronous audio-device initialization failure as a
// fatal game error. Avoid creating its singleton audio context on headless
// Linux sessions with no device, Pulse server, or explicit ALSA configuration.
// This lets pygame scripts use their normal mixer.get_init() fallback.
func audioAvailable() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	if os.Getenv("PULSE_SERVER") != "" || os.Getenv("ALSA_CONFIG_PATH") != "" {
		return true
	}
	exists := func(path string) bool { _, err := os.Stat(path); return err == nil }
	if exists("/dev/snd") || exists("/etc/asound.conf") {
		return true
	}
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" && exists(filepath.Join(dir, "pulse", "native")) {
		return true
	}
	if exists(filepath.Join("/run/user", strconv.Itoa(os.Getuid()), "pulse/native")) {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil && exists(filepath.Join(home, ".asoundrc")) {
		return true
	}
	return false
}

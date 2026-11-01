// Package lib registers the bundled third-party Python compatibility modules.
// Applications can opt in with a blank import, separately from gpython/stdlib.
package lib

import (
	_ "github.com/kazzmir/web-computer/gpython/lib/numpy"
)

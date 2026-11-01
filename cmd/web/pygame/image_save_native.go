//go:build !js

package pygame

import "os"

func saveImageData(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0666)
}

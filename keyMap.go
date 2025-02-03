package evdev

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// generateKeyMap reads the linux input-event-codes.h file and generates a map of key codes to key names.
// Requires the linux headers to be installed.
// Returns a map of key codes to key names.
func generateKeyMap() map[uint16]string {
	keyMap := make(map[uint16]string)
	file, err := os.Open("/usr/include/linux/input-event-codes.h")
	if err != nil {
		logger.Printf("failed to open keycode file: %s. Do you have linux headers? \n sudo apt-get install linux-headers-$(uname -r)", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#define KEY_") && !strings.HasPrefix(line, "#define BTN_") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		keyName := parts[1] // slice at [:4] to Trim KEY_ and BTN_ prefix
		var code uint16
		if _, err := fmt.Sscanf(parts[2], "%d", &code); err != nil {
			logger.Printf("Failed to parse key code from line: %s", line)
			continue
		}
		keyMap[code] = keyName
	}

	if err := scanner.Err(); err != nil {
		logger.Printf("error reading input-event-codes.h: %s", err)
	}
	return keyMap
}

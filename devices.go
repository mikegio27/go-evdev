package evdev

import (
	"bufio"
	"os"
	"strings"
)

const (
	EVIOCGBIT    = 0x80084500 // Get event bitmask
	EVIOCGBITKEY = 0x80084502 // Get key bitmask
	EV_KEY       = 0x01       // Event type for keys
	KEY_A        = 30         // Key code for "A"
	KEY_ENTER    = 28         // Key code for "Enter"
	KEY_ESC      = 1          // Escape key
	KEY_MAX      = 0x2FF      // Maximum key code
)

type InputDevice struct {
	Bus      string
	Vendor   string
	Product  string
	Version  string
	Name     string
	Phys     string
	Sysfs    string
	Uniq     string
	Handlers string
	Props    map[string]string
}

// returns the event ID of the device parsed from Handlers
func (d InputDevice) EventId() string {
	parts := strings.Fields(d.Handlers)
	for _, part := range parts {
		if strings.HasPrefix(part, "event") {
			return part
		}
	}
	return ""
}

// returns the input path of the device (/dev/input/eventX)
func (d InputDevice) InputPath() string {
	return "/dev/input/" + d.EventId()
}

// IsKeyboard checks if the device is a keyboard by checking if it has keys A and Enter.
func (d InputDevice) IsKeyboard() bool {
	return strings.Contains(d.Handlers, "kbd")
}

// This function is used to detect input devices on the system.
// It reads from /proc/bus/input/devices and returns a list of device paths.
// The function returns an error if the file cannot be read or no suitable devices are found.
func InputDevices() ([]InputDevice, error) {
	file, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		logger.Printf("Failed to open /proc/bus/input/devices: %v", err)
		return nil, err
	}
	defer file.Close()

	var devices []InputDevice
	var device InputDevice
	device.Props = make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			devices = append(devices, device)
			device = InputDevice{}
			device.Props = make(map[string]string)
			continue
		}
		switch line[0] {
		case 'I':
			parsedDeviceInfo(line, &device)
		case 'N':
			device.Name = strings.TrimPrefix(line, "N: Name=")
		case 'P':
			device.Phys = strings.TrimPrefix(line, "P: Phys=")
		case 'S':
			device.Sysfs = strings.TrimPrefix(line, "S: Sysfs=")
		case 'U':
			device.Uniq = strings.TrimPrefix(line, "U: Uniq=")
		case 'H':
			device.Handlers = strings.TrimPrefix(line, "H: Handlers=")
		case 'B':
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				device.Props[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Printf("Error reading /proc/bus/input/devices: %v", err)
		return nil, err
	}

	return devices, nil
}

// parsedDeviceInfo parses the device information from the line and stores it in the device struct.
func parsedDeviceInfo(line string, device *InputDevice) {
	parts := strings.Fields(line)
	for _, part := range parts {
		if strings.HasPrefix(part, "Bus=") {
			device.Bus = strings.TrimPrefix(part, "Bus=")
		} else if strings.HasPrefix(part, "Vendor=") {
			device.Vendor = strings.TrimPrefix(part, "Vendor=")
		} else if strings.HasPrefix(part, "Product=") {
			device.Product = strings.TrimPrefix(part, "Product=")
		} else if strings.HasPrefix(part, "Version=") {
			device.Version = strings.TrimPrefix(part, "Version=")
		}
	}
}

package evdev

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
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

type InputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

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
	fmt.Printf("Opening device %s\n", d.InputPath())

	file, err := os.Open(d.InputPath())
	if err != nil {
		logger.Printf("Failed to open device: %v", err)
		return false
	}
	defer file.Close()

	// Step 1: Check if the device supports EV_KEY
	var evBitmask [((EV_KEY + 7) / 8)]byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), EVIOCGBIT, uintptr(unsafe.Pointer(&evBitmask)))
	if errno != 0 {
		logger.Printf("ioctl error while checking EV_KEY: %v", errno)
		return false
	}

	if evBitmask[EV_KEY/8]&(1<<(EV_KEY%8)) == 0 {
		return false // Device does not support key events
	}

	// Step 2: Check if the device has specific keyboard keys
	var keyBitmask [((KEY_MAX + 7) / 8)]byte
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), EVIOCGBITKEY, uintptr(unsafe.Pointer(&keyBitmask)))
	if errno != 0 {
		logger.Printf("ioctl error while checking key bitmask: %v", errno)
		return false
	}

	// Check for common keyboard keys
	if keyBitmask[KEY_A/8]&(1<<(KEY_A%8)) != 0 ||
		keyBitmask[KEY_ENTER/8]&(1<<(KEY_ENTER%8)) != 0 ||
		keyBitmask[KEY_ESC/8]&(1<<(KEY_ESC%8)) != 0 {
		return true
	}

	return false
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

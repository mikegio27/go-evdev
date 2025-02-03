package evdev

import (
	"bufio"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	EVIOCGBIT = 0x20
	EV_KEY    = 0x01
	KEY_A     = 0x1e
	KEY_ENTER = 0x1c
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

// ioctl is a wrapper around the ioctl syscall.
func ioctl(fd uintptr, request, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, request, arg)
	if errno != 0 {
		return errno
	}
	return nil
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

func (d InputDevice) InputPath() string {
	return "/dev/input/" + d.EventId()
}

// IsKeyboard checks if the device is a keyboard by checking if it has keys A and Enter.
func (d InputDevice) IsKeyboard() bool {
	file, err := os.Open(d.InputPath())
	if err != nil {
		logger.Printf("Failed to open device %s: %v", d.InputPath(), err)
		return false
	}
	defer file.Close()

	var keyBits [256]byte
	request := uintptr((2 << 30) | (EVIOCGBIT << 8) | EV_KEY)
	err = ioctl(file.Fd(), request, uintptr(unsafe.Pointer(&keyBits)))
	if err != nil {
		logger.Printf("Failed to get device capabilities: %v", err)
		return false
	}

	return keyBits[KEY_A/8]&(1<<(KEY_A%8)) != 0 && keyBits[KEY_ENTER/8]&(1<<(KEY_ENTER%8)) != 0
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

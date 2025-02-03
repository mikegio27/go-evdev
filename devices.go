package evdev

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
)

const EV_KEY = 0x01

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

// This function is used to detect input devices on the system.
// It reads from /proc/bus/input/devices and returns a list of device paths.
// The function returns an error if the file cannot be read or no suitable devices are found.
func detectInputDevices() ([]string, error) {
	file, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var devicePaths []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "H: Handlers=") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "event") {
					devicePaths = append(devicePaths, "/dev/input/"+part)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning input devices: %w", err)
	}

	if len(devicePaths) == 0 {
		return nil, errors.New("no suitable devices found")
	}

	return devicePaths, nil
}

// StartDeviceMonitoring sets up the context and wait group, starts the watchDevice function in a goroutine,
// and returns the data channel to the user.
func StartDeviceMonitoring(devicePath string) (chan inputEvent, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	dataChan := make(chan inputEvent, 100)

	go func() {
		defer wg.Done()
		watchDevice(ctx, devicePath, dataChan, &wg)
	}()

	// Return the data channel and the cancel function to the user
	return dataChan, cancel, nil
}

// watchDevice monitors the device for key presses and releases, reads from the device file,
// and sends key events to a channel.
func watchDevice(ctx context.Context, devicePath string, dataChan chan inputEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	logger.Println("Monitoring device at", devicePath)

	f, err := os.Open(devicePath)
	if err != nil {
		logger.Printf("Failed to open device %s: %v", devicePath, err)
		close(dataChan)
		return
	}
	defer f.Close()

	reader := bufio.NewReader(f)

	for {
		select {
		case <-ctx.Done():
			close(dataChan)
			return
		default:
			var event inputEvent
			err := binary.Read(reader, binary.LittleEndian, &event)
			if err != nil {
				logger.Printf("Error reading from device %s: %v", devicePath, err)
				close(dataChan)
				return
			}
			dataChan <- event
		}
	}
}

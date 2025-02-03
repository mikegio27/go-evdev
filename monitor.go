package evdev

import (
	"bufio"
	"context"
	"encoding/binary"
	"os"
	"sync"
)

// MonitorSingleDevice sets up the context and wait group, starts the watchDevice function in a goroutine,
// and returns the data channel to the user. Used for a single device.
func MonitorSingleDevice(devicePath string) (chan InputEvent, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	dataChan := make(chan InputEvent, 100)

	go func() {
		defer wg.Done()
		monitorDevice(ctx, devicePath, dataChan, &wg)
	}()

	// Return the data channel and the cancel function to the user
	return dataChan, cancel, nil
}

// MonitorDevices sets up the context and wait group, starts the watchDevice function in a goroutine for each device,
// and returns a map of data channels to the user. Used for multiple devices.
// Pass nil to monitor all input devices.
func MonitorDevices(devices []InputDevice) (map[string]chan InputEvent, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	if devices == nil {
		// Get all input devices
		var err error
		devices, err = InputDevices()
		if err != nil {
			cancel()
			return nil, nil, err
		}
	}
	dataChanMap := make(map[string]chan InputEvent)

	for _, device := range devices {
		wg.Add(1)
		dataChan := make(chan InputEvent, 100)
		dataChanMap[device.InputPath()] = dataChan

		go func(devicePath string) {
			defer wg.Done()
			monitorDevice(ctx, devicePath, dataChan, &wg)
		}(device.InputPath())
	}

	// Return the data channel map and the cancel function to the user
	return dataChanMap, cancel, nil
}

// watchDevice monitors the device for key presses and releases, reads from the device file,
// and sends key events to a channel.
func monitorDevice(ctx context.Context, devicePath string, dataChan chan InputEvent, wg *sync.WaitGroup) {
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
			var event InputEvent
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

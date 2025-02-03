package evdev

import (
	"bufio"
	"context"
	"encoding/binary"
	"os"
	"sync"
)

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

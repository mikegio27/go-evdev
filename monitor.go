package evdev

import (
	"context"
	"log"
	"sync"
	"time"
)

// MonitorSingleDevice sets up the context and wait group, starts the watchDevice function in a goroutine,
// and returns the data channel to the user. Used for a single device.
func MonitorSingleDevice(devicePath string) (chan inputEvent, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	dataChan := make(chan inputEvent, 100)

	go func() {
		defer wg.Done()
		monitorDevice(ctx, devicePath, dataChan)
	}()

	// Return the data channel and the cancel function to the user
	return dataChan, cancel, nil
}

func MonitorAllDevices() {
	devices, err := InputDevices()
	if err != nil {
		log.Fatalf("Failed to get input devices: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	dataChanMap := make(map[string]chan inputEvent)

	for _, device := range devices {
		wg.Add(1)
		dataChan := make(chan inputEvent, 100)
		dataChanMap[device.InputPath()] = dataChan

		go func(devicePath string) {
			defer wg.Done()
			monitorDevice(ctx, devicePath, dataChan)
		}(device.InputPath())
	}

	// Log the outputs of dataChan
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				for devicePath, dataChan := range dataChanMap {
					select {
					case event := <-dataChan:
						log.Printf("Device: %s, Event: %+v\n", devicePath, event)
					default:
					}
				}
			}
			time.Sleep(100 * time.Millisecond) // Adjust the sleep duration as needed
		}
	}()

	wg.Wait()
}

// watchDevice monitors the device for key presses and releases, reads from the device file,
// and sends key events to a channel.
func monitorDevice(ctx context.Context, devicePath string, dataChan chan inputEvent) {
	defer close(dataChan)
	log.Println("Monitoring device at", devicePath)
	// Implement the logic to read from the device file and send events to dataChan
	// This is a placeholder implementation
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Simulate reading an event from the device
			event := inputEvent{Type: 1, Code: 30, Value: 1}
			dataChan <- event
			time.Sleep(1 * time.Second) // Simulate delay between events
		}
	}
}

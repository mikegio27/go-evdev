package evdev

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
)

// devGlob matches the evdev character device nodes.
const devGlob = "/dev/input/event*"

// ListDevicePaths returns the paths of all evdev device nodes, sorted.
func ListDevicePaths() ([]string, error) {
	paths, err := filepath.Glob(devGlob)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

// ListDevices opens every evdev device node and returns the ones that could be
// opened. Devices that fail with a permission error are skipped silently
// (running without privileges yields a partial list); any other error aborts
// and closes the devices opened so far.
//
// The caller owns the returned devices and must Close them.
func ListDevices() ([]*Device, error) {
	paths, err := ListDevicePaths()
	if err != nil {
		return nil, err
	}
	var devices []*Device
	for _, path := range paths {
		d, err := Open(path)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) {
				continue
			}
			closeAll(devices)
			return nil, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// ListKeyboards returns the subset of devices that look like keyboards. As with
// ListDevices, the caller must Close the returned devices.
func ListKeyboards() ([]*Device, error) {
	all, err := ListDevices()
	if err != nil {
		return nil, err
	}
	var keyboards []*Device
	for _, d := range all {
		ok, err := d.IsKeyboard()
		if err != nil || !ok {
			d.Close()
			continue
		}
		keyboards = append(keyboards, d)
	}
	return keyboards, nil
}

func closeAll(devices []*Device) {
	for _, d := range devices {
		d.Close()
	}
}

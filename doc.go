// Package evdev provides access to the Linux evdev input subsystem.
//
// It opens /dev/input/event* devices directly and uses ioctls to query device
// identity and capabilities, and reads input events off the device file. Event
// codes (EV_*, KEY_*, BTN_*, REL_*, ABS_*, ...) are provided as generated Go
// constants in codes.go, so no kernel headers are needed at build or run time.
//
// Most operations require elevated privileges or membership in the "input"
// group to access /dev/input.
package evdev

//go:generate go run ./internal/gen

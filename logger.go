package evdev

import (
	"log"
	"os"
)

var logger = log.New(os.Stderr, "evdev: ", log.LstdFlags)

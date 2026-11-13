package gps

import (
	"fmt"
	"os"
	"syscall"
)

const i2cSlave = 0x0703 // linux/i2c-dev.h

func OpenI2C(path string, address int) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)

	if err != nil {
		return nil, fmt.Errorf("Could not open i2c device %s: %w", path, err)
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), i2cSlave, uintptr(address))
	if errno != 0 {
		f.Close()
		return nil, fmt.Errorf("Could not open i2c address 0x%02x on %s: %w", address, path, errno)
	}
	return f, nil
}

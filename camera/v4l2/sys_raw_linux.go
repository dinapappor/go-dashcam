//go:build !libcamera

package v4l2

import (
	"syscall"
	"unsafe"
)

func openDevice(path string) (int, error) {
	return syscall.Open(path, syscall.O_RDWR|syscall.O_NONBLOCK, 0)
}

func closeDevice(fd int) error {
	return syscall.Close(fd)
}

func ioctlDevice(fd uintptr, request uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL, fd, request, uintptr(arg),
	)
	if errno != 0 {
		return errno
	}
	return nil
}

func mapDevice(fd int, offset int64, length int) ([]byte, error) {
	return syscall.Mmap(
		fd, offset, length,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
}

func unmapDevice(data []byte) error {
	return syscall.Munmap(data)
}

//go:build libcamera && cgo
package v4l2

/*
#include <fcntl.h>
#include <unistd.h>
#include <stdlib.h>
#include <sys/ioctl.h>
#include <sys/mman.h>

static int v4l2_open(const char *path, int flags)
{
    return open(path, flags);
}

static int v4l2_ioctl(int fd, unsigned long request, void *arg)
{
	return ioctl(fd, request, arg);
}
*/
import "C"

import (
	"fmt"
	"syscall"
	"unsafe"
)

func openDevice(path string) (int, error) {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	fd, err := C.v4l2_open(cPath, C.O_RDWR|C.O_NONBLOCK)

	if fd == -1 {
		return -1, fmt.Errorf("%w", err)
	}

	return int(fd), nil
}

func closeDevice(fd int) error {
	result, err := C.close(C.int(fd))

	if result == -1 {
		return fmt.Errorf("%w", err)
	}
	return nil
}

func ioctlDevice(fd uintptr, request uintptr, arg unsafe.Pointer) error {
	result, err := C.v4l2_ioctl(C.int(fd), C.ulong(request), arg)
	if result == -1 {
		return err
	}
	return nil
}

func mapDevice(fd int, offset int64, length int) ([]byte, error) {
	if length <= 0 {
		return nil, syscall.EINVAL
	}
	ptr, err := C.mmap(
		nil,
		C.size_t(length),
		C.PROT_READ|C.PROT_WRITE,
		C.MAP_SHARED,
		C.int(fd),
		C.off_t(offset),
	)

	if ptr == C.MAP_FAILED {
		return nil, err
	}
	return unsafe.Slice((*byte)(ptr), length), nil
}

func unmapDevice(data []byte) error {
	if len(data) == 0 {
		return syscall.EINVAL
	}

	result, err := C.munmap(
		unsafe.Pointer(&data[0]),
		C.size_t(len(data)),
	)
	if result == -1 {
		return err
	}
	return nil
}

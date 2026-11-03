package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
	"unsafe"
	"github.com/dinapappor/go-dashcam/camera/v4l2"
)

func main() {
	// Flags
	videoDevice := flag.String("video_device", "/dev/video0", "Which video device to use")

	output := flag.String("output", "video.mp4", "Which filename to output to")


	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Look for ffmpeg
	ffmpeg, err :=  exec.LookPath("ffmpeg")
	if err != nil {
		log.Printf("Could not find ffmpeg in path")
		return
	}

	log.Printf("FFmpeg path: %s", ffmpeg)

	cmd := exec.Command(ffmpeg,
		"-hide_banner",
		"-f", "mjpeg",
		"-framerate", "30",
		"-i", "pipe:0",
		"-n",
		"-c:v", "libx264",
		"-pix_fmt", "yuv420p",
		*output,
	)
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}


	pipeReader, ffmpegInput, err := os.Pipe()
	if err != nil {
		log.Printf("create FFmpeg pipe: %v", err)
		return
	}

	defer pipeReader.Close()
	defer ffmpegInput.Close()

	cmd.Stdin = pipeReader


	stopPipeClose := context.AfterFunc(ctx, func() {
		ffmpegInput.Close()
	})
	defer stopPipeClose()


	// open the video device
	f, err := os.OpenFile(
		*videoDevice,
		os.O_RDWR|syscall.O_NONBLOCK,
		0,
	)

	if err != nil {
		log.Fatalf("%s: %s", *videoDevice, err)
	}

	defer f.Close()

	var deviceCapability v4l2.V4L2Capability

	// query the ioctl for v4l2 device now
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocQueryCap, uintptr(unsafe.Pointer(&deviceCapability)))
	if errno != 0 {
		log.Printf("Unable to query for device capabilities for %s, errno:%v", *videoDevice, errno)
		return
	}

	log.Printf("Device Capability struct:%+v", deviceCapability)

	driverName, err := v4l2.ConvertByteToString(deviceCapability.Driver[:])

	if err != nil {
		log.Printf("Could not get driver name, err: %s", err)
		return
	}


	cameraName, err := v4l2.ConvertByteToString(deviceCapability.Card[:])

	if err != nil {
		log.Printf("Could not get camera name, err: %s", err)
		return
	}

	busInfo, err := v4l2.ConvertByteToString(deviceCapability.BusInfo[:])

	if err != nil {
		log.Printf("Could not get bus info, err: %s", err)
		return
	}

	fmt.Printf("Video device: %s (%s), using %s driver on %s\n", cameraName, *videoDevice, driverName, busInfo)

	capabilities := v4l2.SelectCapabilities(deviceCapability)

	// Check for video capture flag
	if v4l2.HasCapability(capabilities, v4l2.VideoCapture) {
		log.Printf("We have video capture capabilities")

	} else {
		log.Printf("We do not have video capture capabilities")
		return
	}

	// Check for streaming capabilities.
	// This means buffers that we queue and requeue later
	if v4l2.HasCapability(capabilities, v4l2.StreamingFlag) {
		log.Printf("We have streaming capabilities")
	} else {
		log.Printf("We do not have streaming capabilities")
		return
	}

	var deviceFormat v4l2.V4L2Format
	deviceFormat.Type = v4l2.BufTypeVideoCapture

	// query the ioctl for v4l2 device now
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocGFMT, uintptr(unsafe.Pointer(&deviceFormat)))
	if errno != 0 {
		log.Printf("Unable to query for device format for %s, errno:%v", *videoDevice, errno)
		return
	}

	log.Printf("%+v", deviceFormat.PixFormat)

	// Get the four CC
	fourCC := v4l2.DecodeFourCC(deviceFormat.PixFormat.PixelFormat)

	log.Printf("Pixformat: %s", fourCC)

	var requestedFormat v4l2.V4L2Format

	requestedFormat.Type = v4l2.BufTypeVideoCapture
	requestedFormat.PixFormat.Width = 1280
	requestedFormat.PixFormat.Height = 720
	requestedFormat.PixFormat.PixelFormat = v4l2.PixFmtMJPEG

	// Saving a copy of what we request for easier comparison
	origRequestedFormat := requestedFormat

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocSFMT, uintptr(unsafe.Pointer(&requestedFormat)))

	if errno != 0 {
		log.Printf("Unable to set requestFormat. Got %v", errno)
		return
	}

	log.Printf("We got format: %s, width:%d, height: %d",
		v4l2.DecodeFourCC(requestedFormat.PixFormat.PixelFormat),
		requestedFormat.PixFormat.Width,
		requestedFormat.PixFormat.Height)

	// Sanity checks

	if !v4l2.ValidateFormat(origRequestedFormat, requestedFormat) {
		log.Printf("Validation of requests failed, bailing")
		return
	}

	var streamParams v4l2.V4L2StreamParm
	streamParams.Type = v4l2.BufTypeVideoCapture

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocGPARM, uintptr(unsafe.Pointer(&streamParams)))

	if errno != 0 {
		log.Printf("Unable to read StreamParm, got %v", errno)
		return
	}

	log.Printf("%v", streamParams.Capture.TimePerFrame)

	if v4l2.HasCapability(streamParams.Capture.Capability, v4l2.CapTimePerFrame) {
		log.Printf("We have TimePerFrame capability")
	} else {
		log.Printf("We do not have TimePerFrame capability")
		return
	}

	var requestedParams v4l2.V4L2StreamParm
	requestedParams.Type = v4l2.BufTypeVideoCapture
	requestedParams.Capture.TimePerFrame.Numerator = 1
	requestedParams.Capture.TimePerFrame.Denominator = 30

	origParams := requestedParams

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocSPARM, uintptr(unsafe.Pointer(&requestedParams)))

	if errno != 0 {
		log.Printf("Unable to request StreamParm, got %v", errno)
		return
	}

	if ! v4l2.ValidateFrameInterval(origParams.Capture.TimePerFrame, requestedParams.Capture.TimePerFrame) {
		log.Printf("Frame intervals did not match")
		return
	}

	log.Printf("%+v", requestedParams.Capture.TimePerFrame)

	var requestBuffers v4l2.V4L2RequestBuffers

	requestBuffers.Count = 4
	requestBuffers.Type = v4l2.BufTypeVideoCapture
	requestBuffers.Memory = v4l2.MemoryMmap

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocReqBufs, uintptr(unsafe.Pointer(&requestBuffers)))

	if errno != 0 {
		log.Printf("Unable to allocate requestBuffers, got %v", errno)
		return
	}

	if requestBuffers.Count == 0 {
		log.Printf("Could not get any request buffers")
		return
	}

	log.Printf("Request buffers: %d", requestBuffers.Count)

	// Store buffers
	buffers := make([][]byte, requestBuffers.Count)
	for index := uint32(0); index < requestBuffers.Count; index++ {
		buffer := v4l2.V4L2Buffer{}
		buffer.Type = v4l2.BufTypeVideoCapture
		buffer.Index = index
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocQueryBuf, uintptr(unsafe.Pointer(&buffer)))

		if errno != 0 {
			log.Printf("Could not query buffer for reading, errno: %d", errno)
			return
		}
		data, err := syscall.Mmap(
			int(f.Fd()),
			int64(buffer.M.Offset),
			int(buffer.Length),
			syscall.PROT_READ|syscall.PROT_WRITE,
			syscall.MAP_SHARED,
		)

		if err != nil {
			log.Printf("Could not map buffer: %v", err)
			return
		}
		defer syscall.Munmap(data)

		// Let's store the data off buffers in buffers
		buffers[index] = data

		buffer.Memory = v4l2.MemoryMmap

		log.Printf("Mapped buffer: %d bytes", len(data))

		buffer.Flags = 0

		_, _, errno = syscall.Syscall(
			syscall.SYS_IOCTL,
			f.Fd(),
			v4l2.VidiocQBuf,
			uintptr(unsafe.Pointer(&buffer)),
		)

		if errno != 0 {
			log.Printf("Could not queue buffer %d: %v", index, errno)
			return
		}
	}

	err = cmd.Start()

	if err != nil {
		log.Printf("Could not start FFmpeg: %v", err)
		return
	}

	pipeReader.Close()

	defer func() {
		ffmpegInput.Close()

		timer := time.AfterFunc(5*time.Second, func() {
			log.Printf("FFmpeg shutdown timed out; stopping it")
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Could not kill FFmpeg: %v", err)
			}
		})
		defer timer.Stop()

		if waitErr := cmd.Wait(); waitErr != nil {
			log.Printf("FFmpeg failed: %v", waitErr)
		}
	}()
	bufferType := uint32(v4l2.BufTypeVideoCapture)

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocStreamOn, uintptr(unsafe.Pointer(&bufferType)))

	if errno != 0 {
		log.Printf("Unable to start stream. Got %+v", errno)
		return
	}
	defer func() {
		_, _, stopErr := syscall.Syscall(
			syscall.SYS_IOCTL,
			f.Fd(),
			v4l2.VidiocStreamOff,
			uintptr(unsafe.Pointer(&bufferType)),
		)
		if stopErr != 0 {
			log.Printf("Unable to stop stream: %v", stopErr)
		}
	}()

	for frameNumber := 0; frameNumber < 300 && ctx.Err() == nil; {
		completedBuffer := v4l2.V4L2Buffer{}
		completedBuffer.Type = v4l2.BufTypeVideoCapture
		completedBuffer.Memory = v4l2.MemoryMmap

		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocDQBuf, uintptr(unsafe.Pointer(&completedBuffer)))

		if errno == syscall.EAGAIN {
			select {
			case <- ctx.Done():
				return
			case <- time.After(10 * time.Millisecond):
			}
			continue
		}

		if errno != 0 {
			log.Printf("Unable to dequeue buffer. Got %+v", errno)
			return
		}

		if completedBuffer.Flags&v4l2.BufFlagError != 0 {
			log.Printf("Frame was marked as damaged")
		} else {

			log.Printf("Buffer: %d, bytes used: %d", completedBuffer.Index, completedBuffer.BytesUsed)

			frame, err := v4l2.FrameBytes(
				buffers,
				completedBuffer.Index,
				completedBuffer.BytesUsed,
			)

			if err != nil {
				log.Printf("Unable to read buffers, got %s", err)
				return
			}

			log.Printf("Length of frame: %d", len(frame))

			err = ffmpegInput.SetWriteDeadline(time.Now().Add(2 * time.Second))

			if err != nil {
				log.Printf("Could not set a writedeadline for ffmpeg pipe, err: %s", err)
				return
			}

			written, err := ffmpegInput.Write(frame)

			if err != nil {
				log.Printf("Could not write frame to FFmpeg: %v", err)
				return
			}

			if written != len(frame) {
				log.Printf("Incomplete frame write: %d of %d bytes", written, len(frame))
				return
			}
		}

		completedBuffer.Flags = 0

		// Requeue after finishing with the frame
		_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocQBuf, uintptr(unsafe.Pointer(&completedBuffer)))

		if errno != 0 {
			log.Printf("Unable to requeue buffer")
			return
		}
		frameNumber++
	}
}

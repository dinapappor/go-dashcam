package main

import (
	"context"
	"errors"
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

	deviceCapability, err := v4l2.QueryCapability(f.Fd())

	if err != nil {
		log.Printf("Unable to query for device Capabilities, err:%s", err)
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

	deviceFormat, err := v4l2.QueryVideoCaptureFormat(f.Fd())

	if err != nil {
		log.Printf("Unable to query video format: %s", err)
		return
	}

	log.Printf("%+v", deviceFormat.PixFormat)

	// Get the four CC
	fourCC := v4l2.DecodeFourCC(deviceFormat.PixFormat.PixelFormat)

	log.Printf("Pixformat: %s", fourCC)

	requestedFormat, err := v4l2.SetVideoCaptureFormat(f.Fd(), 1280, 720, v4l2.PixFmtMJPEG)

	if err != nil {
		log.Printf("Unable to set format: %s", err)
		return
	}

	log.Printf("We got format: %s, width:%d, height: %d",
		v4l2.DecodeFourCC(requestedFormat.PixFormat.PixelFormat),
		requestedFormat.PixFormat.Width,
		requestedFormat.PixFormat.Height)

	streamParams, err := v4l2.QueryVideoCaptureStreamParams(f.Fd())
	if err != nil {
		log.Printf("Unable to query for stream params, got: %s", err)
		return
	}

	log.Printf("%v", streamParams.Capture.TimePerFrame)

	if v4l2.HasCapability(streamParams.Capture.Capability, v4l2.CapTimePerFrame) {
		log.Printf("We have TimePerFrame capability")
	} else {
		log.Printf("We do not have TimePerFrame capability")
		return
	}

	requestedParams, err := v4l2.SetVideoCaptureStreamParams(f.Fd(), 1, 30)

	if err != nil {
		log.Printf("Unable to set stream params, got %s", err)
		return
	}


	log.Printf("%+v", requestedParams.Capture.TimePerFrame)

	requestBuffers, err := v4l2.RequestMmapBuffers(f.Fd(), 4)

	if err != nil {
		log.Printf("Unable to request buffers, got %s", err)
		return
	}

	log.Printf("Request buffers: %d", requestBuffers.Count)

	// Store buffers
	buffers := make([][]byte, requestBuffers.Count)
	for index := uint32(0); index < requestBuffers.Count; index++ {
		buffer, err := v4l2.QueryBuffer(f.Fd(), index)

		if err != nil {
			log.Printf("Could not query buffer, got %s", err)
			return
		}

		data, err := v4l2.MapBuffer(f.Fd(), buffer)

		if err != nil {
			log.Printf("Could not map buffer: %v", err)
			return
		}
		defer syscall.Munmap(data)

		// Let's store the data off buffers in buffers
		buffers[index] = data


		log.Printf("Mapped buffer: %d bytes", len(data))

		err = v4l2.QueueBuffer(f.Fd(), buffer)
		if err != nil {
			log.Printf("Could not queue buffer: %v", err)
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

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), v4l2.VidiocStreamOn, uintptr(unsafe.Pointer(&bufferType)))

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
		// Dequeue bufferss
		completedBuffer, err := v4l2.DequeueBuffer(f.Fd())

		if errors.Is(err, syscall.EAGAIN) {
			select {
			case <- ctx.Done():
				return
			case <- time.After(10 * time.Millisecond):
			}
			continue
		}

		if err != nil {
			log.Printf("Unable to dequeue buffer. Got %+v", err)
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

		err = v4l2.QueueBuffer(f.Fd(), completedBuffer)
		if err != nil {
			log.Printf("Could not requeue the buffer: %v", err)
			return
		}
		frameNumber++
	}
}

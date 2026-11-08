package v4l2

// CapDeviceCaps means DeviceCaps contains the capabilities of this device node.
// When set in Capabilities, check DeviceCaps instead of the device-wide flags.
const CapDeviceCaps = 0x80000000

// VidiocQueryCap asks the driver for its identity and supported capabilities.
// The request encodes: read direction (2), type 'V' (86), struct size (104),
// and command number (0). These ioctl numbers use our current Linux ABI.
const VidiocQueryCap = (2 << 30) | (86 << 8) | (104 << 16) | 0

// VideoCapture means the device supports video capture through the single-planar API.
const VideoCapture = 0x00000001

// BufTypeVideoCapture selects the single-planar capture queue in ioctl arguments.
// This is a buffer type value, not a capability flag.
const BufTypeVideoCapture = 0x0000001

// StreamingFlag means the device supports queued-buffer streaming I/O.
// This describes camera buffer access, not network streaming.
const StreamingFlag = 0x04000000

// VidiocGFMT reads the current image format, including dimensions and pixel format.
const VidiocGFMT = 0xC0D05604

// VidiocSFMT requests an image format and returns the format the driver accepted.
// The driver may adjust the request, so check the returned fields.
const VidiocSFMT = 0xC0D05605

// PixFmtMJPEG is the FourCC code 'MJPG': each frame is a compressed JPEG image.
const PixFmtMJPEG = 0x47504A4D

// PixFmtYUYV is the FourCC code 'YUYV': packed, uncompressed YUV 4:2:2.
// Each pair of pixels stores Y0, U, Y1, V, sharing the U and V colour samples.
const PixFmtYUYV = 0x56595559

// PixFmtYUV420 is 'YU12': planar Y, Cb, Cr with 4:2:0 subsampling
const PixFmtYUV420 =  0x32315559

// VidiocGPARM reads stream parameters, including the frame interval and its capabilities.
const VidiocGPARM = 0xC0CC5615

// VidiocSPARM requests stream parameters and returns the values the driver accepted.
const VidiocSPARM = 0xC0CC5616

// CapTimePerFrame means the driver supports changing the frame interval.
// Check it in the capture parameters Capability field before requesting a rate.
const CapTimePerFrame = 0x1000

// MemoryMmap selects driver-allocated buffers mapped into our process with mmap.
const MemoryMmap = 1

// VidiocReqBufs requests streaming buffers and returns the allocated count.
// With MemoryMmap, the driver allocates the buffers; the count may differ from our request.
const VidiocReqBufs = 0xC0145608

// VidiocQueryBuf reads buffer details by index, including its mmap offset and length.
const VidiocQueryBuf = 0xC0585609

// VidiocQBuf queues a buffer for the driver to fill with a captured frame.
// Do not access its contents until it has been dequeued again.
const VidiocQBuf = 0xC058560F

// VidiocStreamOn starts capture on the selected queue after buffers are queued.
const VidiocStreamOn = 0x40045612

// VidiocStreamOff stops capture and removes buffers from the queue.
// It does not unmap the memory mapped by our process.
const VidiocStreamOff = 0x40045613

// VidiocDQBuf returns a completed buffer for our process to read.
// With O_NONBLOCK, it returns EAGAIN when no completed buffer is ready.
const VidiocDQBuf = 0xc0585611

// BufFlagError marks a dequeued buffer whose frame may be corrupted.
// Skip its data; the buffer can still be requeued to continue capture.
const BufFlagError = 0x00000040

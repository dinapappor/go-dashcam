package v4l2
// Where we can read CapDeviceCaps
const CapDeviceCaps = 0x80000000

const VidiocQueryCap = (2 << 30) | (86 << 8) | (104 << 16) | 0

// videocapture flag
const VideoCapture = 0x00000001

const BufTypeVideoCapture = 0x0000001

// streaming flag
const StreamingFlag = 0x04000000

// Format request number
const VidiocGFMT = 0xC0D05604

// Format request number
const VidiocSFMT = 0xC0D05605

// MJPG
const PixFmtMJPEG = 0x47504A4D

// YUYV
const PixFmtYUYV = 0x56595559

const VidiocGPARM = 0xC0CC5615
const VidiocSPARM = 0xC0CC5616

const CapTimePerFrame = 0x1000

const MemoryMmap = 1

const VidiocReqBufs = 0xC0145608
const VidiocQueryBuf = 0xC0585609

const VidiocQBuf = 0xC058560F
const VidiocStreamOn = 0x40045612
const VidiocStreamOff = 0x40045613
const VidiocDQBuf = 0xc0585611

const BufFlagError = 0x00000040

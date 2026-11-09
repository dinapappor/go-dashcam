package v4l2m2m

const (
	vidiocQueryCap  = 0x80685600
	vidiocSFmt      = 0xC0D05605
	vidiocReqBufs   = 0xC0145608
	vidiocQueryBuf  = 0xC0585609
	vidiocQBuf      = 0xC058560F
	vidiocDQBuf     = 0xC0585611
	vidiocStreamOn  = 0x40045612
	vidiocStreamOff = 0x40045613
	vidiocSCtrl     = 0xC008561C
	vidiocSParm     = 0xC0CC5616
)

const (
	bufTypeCaptureMplane = 9
	bufTypeOutputMplane  = 10
	memoryMmap           = 1
	memoryDmabuf         = 4
	fieldNone            = 1
	bufFlagError         = 0x40
)

const (
	capVideoM2MMplane = 0x00004000
	capStreaming      = 0x04000000
	capDeviceCaps     = 0x80000000
)

const (
	pixFmtYUV420 = 0x32315559
	pixFmtH264   = 0x34363248
)

const (
	cidBitrate         = 0x009909CF
	cidRepeatSeqHeader = 0x009909E2
	cidH264IPeriod     = 0x00990A66
)

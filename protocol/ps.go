package protocol

type PSInfo struct {
	*PSHeader        `json:"PS Header"` // PS头部长度不固定的
	*PSSystemHeader  `json:"PS System Header"`
	*PSM             `json:"PSM"`
	PESCount         int        `json:"此PS包下PES包总数"`
	PESInfoList      []*PESInfo `json:"此PS包下的PES包列表"`
	PESInfoVideoList []*PESInfo `json:"-"`
	PESInfoAudioList []*PESInfo `json:"-"`
	*FrameInfo       `json:"-"` // PS所属的帧

	PSInfoImportMsg string `json:"msg"`
}

type PSHeader struct {
	Len       uint16 `json:"PS Header总长度"`     // `PS Header`总长度
	ExtendLen uint16 `json:"PS Header的扩展长度字段"` // `PS Header`的扩展长度字段
}

// PS System Header
type PSSystemHeader struct {
	Len        uint16   `json:"PS System Header总长度"` // `PSSystemHeader` 总长度
	StreamByte []string `json:"streamByte"`          // 每个长度为3字节，第一个字节是streamID
}

// Program Stream Map（PSM）
type PSM struct {
	// 长度字段之后是这样的:
	// 2字节固定长度、2字节program_stream_info_length、对应长度的字节、2字节element_stream_map_length、对应长度的字节、4字节CRC_32
	PSMLen                  uint16 `json:"PSM总长度"` // `PSM`总长度
	ProgramStreamInfoLength uint16 `json:"-"`      // descriptor 的长度
	ElementStreamMapLength  uint16 `json:"-"`      // 基本映射流 的长度， 基本映射流用来描述原始流信息

	// descriptor + 基本映射流 的 长度就是 长度字段表示的长度
	StreamInfoList []*StreamInfo `json:"流描述信息集"`
}

// StreamInfo 流的描述信息
type StreamInfo struct {
	StreamType    string `json:"stream_type"` // 码流类型 stream_type, 比如H264、H265
	StreamTypeVal string `json:"stream_type[值]"`
	StreamID      string `json:"stream_id"` // 负载流类型，比如是视频、音频
	StreamIDVal   string `json:"stream_id[值]"`
	Len           uint16 `json:"stream描述信息占用的字节"` // stream描述占用的字节

	Msg string `json:"msg"`
}

package protocol

type RTP struct {
	Header  *RTPHeader
	Payload []byte
}

type RTPHeader struct {
	Version        uint8  `json:"version"`        // rtp版本号	2bit
	Padding        bool   `json:"padding"`        // 填充位，为1表示包尾部需要填充字节 1bit
	Extension      bool   `json:"extension"`      // 扩展位x	1bit
	CSRCCount      uint8  `json:"CSRCCount"`      // CSRC计数器 4bit 表示本数据包中含有的CSRC的个数
	Marker         bool   `json:"marker"`         // 标记位 1bit ，为1表示该数据包是一帧数据的最后一个数据包
	PayloadType    uint8  `json:"payloadType"`    // 负载类型 7bit
	SequenceNumber uint16 `json:"sequenceNumber"` // 序列号 2B
	Timestamp      uint32 `json:"timestamp"`      // 时间戳 4B
	SSRC           uint32 `json:"SSRC"`           // 同步源标识符 4B

	CSRCList []uint32 `json:"CSRCList"`

	HeaderBytes string `json:"header_bytes"`
}

// RTPAnalysis RTP分析结果
type RTPAnalysis struct {
	TotalRtp      int          `json:"RTP包总数"`
	LastRtpHeader []*RTPHeader `json:"-"` // 最后三个RTP的头信息
	PreRtpHeader  []*RTPHeader `json:"-"` // 前三个RTP的头信息
	LostSeqNumber []uint16     `json:"丢失的RTP包的Seq"`
	//SameTimestampRtpSeq []uint16              `json:"有相同timestamp的rtp的序号"`
}

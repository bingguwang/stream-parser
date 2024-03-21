package protocol

type PESHeader struct {
	// 长度字段之后，PES的header是这样：
	// 2个字节、一个字节的PESHeaderDataLength， PESHeaderDataLength个字节
	StreamID            string `json:"stream_id"` // 负载流类型
	StreamIDVal         string `json:"stream_id[值]"`
	PESLen              uint16 `json:"PES总长度"`        // PES 总长度, 包括header和body
	PtsDtsFlags         string `json:"pts_dts_flags"` // pts_dts_flags
	PESHeaderDataLength uint8  `json:"PES header 长度"` // PES header 长度
	// 视频和音频需要区分开
	Pts         int64 `json:"pts"`
	LastPts     int64 `json:"上一个pes包的pts"`
	PtsDuration int64 `json:"和上一个pes包的pts之差"`
	Dts         int64 `json:"dts"`
	LastDts     int64 `json:"上一个pes包的dts"`
	DtsDuration int64 `json:"和上一个pes包的dts之差"`
}

type PESInfo struct {
	*PESHeader `json:"PES Header"`
	Body       []byte `json:"-"` // body 是PESHeaderDataLength字段之后跳过PESHeaderDataLength个字节

	// 所属的ps
	PS  *PSInfo `json:"-"`
	Msg string  `json:"msg"`
}

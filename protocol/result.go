package protocol

type Result struct {
	*FrameAnalysis `json:"帧分析结果"`
	*RTPAnalysis   `json:"RTP分析结果"`
}

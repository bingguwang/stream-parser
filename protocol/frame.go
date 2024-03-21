package protocol

type FrameInfoArr []*FrameInfo
type JumpDiffFrameDisplayArr []*JumpDiffFrameDisplay

// FrameAnalysis 帧分析结果
type FrameAnalysis struct {
	TotalFrame             int                      `json:"总帧数"`
	VideoFrameCount        int                      `json:"视频帧总数"`
	AudioFrameCount        int                      `json:"音频帧总数"`
	VideoFrame             *FrameInfoArr            `json:"视频帧"`
	AudioFrame             *FrameInfoArr            `json:"音频帧"`
	KeyFrame               *FrameInfoArr            `json:"关键帧详情"`
	JumpDiffFrameVideoList *JumpDiffFrameDisplayArr `json:"帧间pts跳变[video]"`
	JumpDiffFrameAudioList *JumpDiffFrameDisplayArr `json:"帧间pts跳变[audio]"`
	//JumpDiffFrameVideoList []string                `json:"帧间pts跳变[video]"`
	//JumpDiffFrameAudioList []string     `json:"帧间pts跳变[audio]"`
	OnlyOneFrameForRtp OnlyOneFrameForRtpStruct `json:"只需要一个RTP携带的帧"`
	FrameRateByRtpTs   int                      `json:"帧率(通过RTP时间戳计算而来)"`
	FrameRateByPts     int                      `json:"帧率(通过Pts计算而来)"`
	FramRtptsJump      []*FrameInfo             `json:"-"` // 与上一帧相比，rtp时间戳出现跳变的帧
	FrameNumberString  []string                 `json:"视频帧和音频帧分布"`
	TotalPS            int                      `json:"所有帧的PS包总数"`
	TotalPES           int                      `json:"所有帧的PES包总数"`

	VideoPES []*PESInfo `json:"-"` //全部的video类型的PES
	AudioPES []*PESInfo `json:"-"` //全部的video类型的PES
}

type OnlyOneFrameForRtpStruct struct {
	OnlyOneFrameForRtpInfosA []*OnlyOneFrameForRtpInfo `json:"音频帧"`
	OnlyOneFrameForRtpInfosV []*OnlyOneFrameForRtpInfo `json:"视频帧"`
}

type OnlyOneFrameForRtpInfo struct {
	FrameNumberID int64  `json:"帧号"`
	FrameType     string `json:"帧类型"`
	RtpSeq        uint16 `json:"携带此帧的RTP的序列号"`
	Msg           string `json:"msg"`
}

// 帧间跳变显示信息
type JumpDiffFrameDisplay struct {
	IDNumber    int64  `json:"帧序号"`
	IsKey       bool   `json:"是否关键帧"` // 是否关键帧
	Pts         int64  `json:"本帧的pts"`
	LastPts     int64  `json:"上一帧的pts"`
	PtsDuration int64  `json:"pts之差"`
	RtpSeq      string `json:"携带此帧的rtp序列号"`
	Dts         int64  `json:"dts"`
	LastDts     int64  `json:"上一帧的dts"`
	DtsDuration int64  `json:"dts之差"`

	Msg string `json:"msg"`
}

type FrameInfo struct {
	IDNumber int64 `json:"帧序号"`
	IsKey    bool  `json:"是否关键帧"` // 是否关键帧
	//RtpSeqList    []uint16  `json:"携带此帧数据的RTP包的序号"`
	RtpSeqListStr string    `json:"携带此帧数据的RTP包的序号"`
	PSList        []*PSInfo `json:"此帧的PS包列表"` // 一帧数据可能有多个PS包

	FrameInfoImportMsg string `json:"msg"`
}

func (f *FrameInfoArr) FindBinSearchByKey(id int64) *FrameInfo {
	left, right := 0, len(*f)-1
	for left <= right {
		mid := left + (right-left)/2
		if (*f)[mid].IDNumber == id {
			return (*f)[mid]
		} else if (*f)[mid].IDNumber < id {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return nil
}

func (f *JumpDiffFrameDisplayArr) FindBinSearchByKey(id int64) *JumpDiffFrameDisplay {
	left, right := 0, len(*f)-1
	for left <= right {
		mid := left + (right-left)/2
		if (*f)[mid].IDNumber == id {
			return (*f)[mid]
		} else if (*f)[mid].IDNumber < id {
			left = mid + 1
		} else {
			right = mid - 1
		}
	}
	return nil
}

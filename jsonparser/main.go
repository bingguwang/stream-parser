package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	jsoniter "github.com/json-iterator/go"
	"jsonparser/protocol"
	pcapng "jsonparser/reader"
	"jsonparser/utils"
	"os"
	"strconv"
	"strings"
)

var (
	packCount int
)

const (
	PAYLOADFile = 1
	PCAPNGFile  = 2

	srcAddr = ""
)

func CheckFile(filename string) int {
	if strings.HasSuffix(filename, ".pcapng") {
		return PCAPNGFile
	}
	return PAYLOADFile
}

var (
	files []string
)

type arrayFlagValue []string

func (a *arrayFlagValue) String() string {
	return fmt.Sprintf("%v", *a)
}

func (a *arrayFlagValue) Set(value string) error {
	*a = append(*a, value)
	return nil
}
func main() {
	flag.Var((*arrayFlagValue)(&files), "file", "value:要解析文件的绝对路径,文件格式是tcp raw流,文件可在wireshark 追踪流获取")
	flag.Parse()

	//if len(files) == 0 {
	//	// todo pcapng文件有点问题
	//	files = []string{
	//		//`C:\Users\dell\3D Objects\VSC\test`,
	//		//`C:\Users\dell\3D Objects\VSC\rtptest.pcapng`,
	//		//`C:\Users\dell\3D Objects\VSC\rtptest4.pcapng`,
	//		//`C:\Users\dell\3D Objects\VSC\test4`,
	//		//`C:\Users\dell\3D Objects\VSC\rtptest2.pcapng`,
	//		//`C:\Users\dell\3D Objects\VSC\test2`,
	//		`C:\Users\dell\3D Objects\VSC\test3`,
	//		//`C:\Users\dell\3D Objects\VSC\rtptest3.pcapng`,
	//	}
	//}

	for _, filepath := range files {
		//filename := filepath2.Base(filepath)
		var (
			tcpPayloadStream []byte
			err              error
		)
		switch CheckFile(filepath) {
		case PAYLOADFile:
			tcpPayloadStream, err = ParsePayloadFile(filepath)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
		case PCAPNGFile:
			tcpPayloadStream, err = ParsePcapngFile(filepath)
			if err != nil {
				fmt.Println(err.Error())
				continue
			}
		default:
			fmt.Println("错误的文件类型")
			continue
		}

		var (
			aggregatedData []byte
			lastVideoPts   int64 = 0
			lastAudioPts   int64 = 0
			frameNumber    int64 = 0
			lastVideoPes   *PESInfo
			lastAudioPes   *PESInfo
		)

		var (
			frameInfoList          []*FrameInfo
			keyFrameList           []*FrameInfo
			jumpInnerFrameVideo    []int64                 // 帧内跳变
			jumpInnerFrameAudio    []int64                 // 帧内跳变
			jumpDiffFrameVideoList []*JumpDiffFrameDisplay // 帧间跳变,video
			//jumpDiffFrameVideoList []string                // 帧间跳变,video
			jumpDiffFrameAudioList []*JumpDiffFrameDisplay // 帧间跳变,audio
			totalPS                int                     // 所有帧含有的PS包的总数
			totalPES               int                     // 所有帧含有的PES包的总数
			sameFrameRtpSeq        []uint16                // 携带同一帧数据的RTP的序列号
			frameRate              int                     // 帧率，pfs
			tmpRate                float64                 // 帧率，pfs
			frameRateList          []float64
			framRtptsJump          []*FrameInfo // 与上一帧相比，rtp时间戳出现跳变的帧
			//framRtptsJumpAudio     []*FrameInfo // 与上一帧相比，rtp时间戳出现跳变的音频帧
			tmpRatePts           float64 // 帧率，pfs,由PTS计算得到
			frameRateListPts     []float64
			frameRatepts         int
			withPtsPesCount      int // 有pts的pes包个数
			withPtsPesVideoCount int // 有pts的视频pes包个数
			withPtsPesAudioCount int // 有pts的音频pes包个数

			audioFrameNo []int64      // 音频帧的帧号
			videoFrameNo []int64      // 视频帧的帧号
			videoFrame   []*FrameInfo // 视频帧
			audioFrame   []*FrameInfo // 音频帧
		)

		var (
			lastRtpHeader         *protocol.RTPHeader
			lostRtpSeqNumbers     []uint16                 // 丢失的rtp的序号
			sameTimestampRtpSeq   []uint16                 // timestamp相同的rtp的序号
			onlyOneFrameForRtp    OnlyOneFrameForRtpStruct // 只需要一个Rtp携带的帧
			lastFrameRtpTimestamp uint32                   // 上一帧的RTP时间戳

			pesDuarationInnerFrameVideo []int64
			pesDuarationInnerFrameAudio []int64
			pesDuarationOuterFrameVideo []int64
			pesDuarationOuterFrameAudio []int64

			allVideoPES []*PESInfo
			allAudioPES []*PESInfo
		)

		// 解析RTP
		for i, j := 0, 0; ; {
			if i >= len(tcpPayloadStream) {
				break
			}
			j = i + 2
			dataLength := binary.BigEndian.Uint16(tcpPayloadStream[i:j])
			length := int(dataLength) // rtp包长度
			i = j
			j = i + length
			if j > len(tcpPayloadStream) {
				break
			}

			// 解析RTP头，并将rtp头去掉
			rtp := protocol.RTP{}
			if len(tcpPayloadStream[i:j]) >= RTPHeaderLen {
				parseRTPHeader(tcpPayloadStream[i:j], &rtp)
				// 判断丢失rtp包
				if lastRtpHeader != nil {
					gap := rtp.Header.SequenceNumber - lastRtpHeader.SequenceNumber
					for gap > 1 {
						lostRtpSeqNumbers = append(lostRtpSeqNumbers, rtp.Header.SequenceNumber-gap+1)
						gap--
					}
				}

				if lastRtpHeader != nil && lastRtpHeader.Timestamp == rtp.Header.Timestamp {
					sameTimestampRtpSeq = append(sameTimestampRtpSeq, rtp.Header.SequenceNumber)
				}

				RtpList = append(RtpList, &rtp)
				lastRtpHeader = rtp.Header
			} else {
				continue
			}
			i = j

			sameFrameRtpSeq = append(sameFrameRtpSeq, rtp.Header.SequenceNumber)
			// 解析PS流，一帧一帧解析，得到一帧的完整的数据之后才开始解析
			if !rtp.Header.Marker { // 不是帧的最后一个数据包
				aggregatedData = append(aggregatedData, rtp.Payload...)
				continue
			} else { // 是一帧的最后一个包
				frameNumber++
				aggregatedData = append(aggregatedData, rtp.Payload...)
				isKey := false // 是否是关键帧
				//isVideo := false
				rawData := aggregatedData[:]
				var ps *PSInfo
				var frame = &FrameInfo{}

				if rtp.Header.Timestamp-lastFrameRtpTimestamp > 0 {
					tmpRate = float64(90000.0 / (rtp.Header.Timestamp - lastFrameRtpTimestamp))
					if tmpRate < 1 { // 时间戳差值比时钟频率还大或者为负
						framRtptsJump = append(framRtptsJump, frame)
					} else {
						frameRateList = append(frameRateList, tmpRate)
					}
					lastFrameRtpTimestamp = rtp.Header.Timestamp
				}

				for {
					startCode := binary.BigEndian.Uint32(rawData[:4])
					psDataWithoutPrefix := rawData[4:]
					if !isPsStartCodeValid(startCode) { // 不是ps流的起始码
						fmt.Printf("start code is not vaild, startcode:0x%08x\n", startCode)
						return
					}
					if startCode == 0x000001ba { // PS Header的起始码
						// 创建一个PSH代表一个新的PS包
						psDataWithoutPrefix = psDataWithoutPrefix[9:]
						// 获取扩展数据长度
						extendLen := psDataWithoutPrefix[0] & 0x07

						ps = &PSInfo{
							PSHeader: &PSHeader{ExtendLen: uint16(extendLen), Len: uint16(14 + extendLen)},
						}
						totalPS++
						//frame.PSList = append(frame.PSList, ps) // 先加入ps
						ps.FrameInfo = frame
						ps.FrameInfo.PSList = append(ps.FrameInfo.PSList, ps)
						rawData = psDataWithoutPrefix[1+extendLen:]
					} else if startCode == 0x000001bb { // PS System Header的起始码
						// (ps system header) 当且仅当数据包为第一个数据包时才存在
						// 两字节为长度
						len := binary.BigEndian.Uint16(psDataWithoutPrefix[:2])

						psh := &PSSystemHeader{Len: 4 + 2 + len}
						for i := 0; i < int(len); {
							psh.StreamByte = append(psh.StreamByte,
								hex.EncodeToString(psDataWithoutPrefix[2+6+i:2+6+i+3])) // 6字节需要跳过
							i += 3
						}
						ps.PSSystemHeader = psh
						rawData = psDataWithoutPrefix[len+2:]
					} else if startCode == 0x000001bc { // PSM的起始码
						// (ps map header) 之后接着包含关键帧的pes
						isKey = true

						// 两字节为长度
						len := binary.BigEndian.Uint16(psDataWithoutPrefix[:2])

						psm := &PSM{}
						psm.PSMLen = len + 6
						psm.ProgramStreamInfoLength = binary.BigEndian.Uint16(psDataWithoutPrefix[2+2 : 2+2+2])
						psm.ElementStreamMapLength = binary.BigEndian.Uint16(psDataWithoutPrefix[2+2+2+psm.ProgramStreamInfoLength : 2+2+2+psm.ProgramStreamInfoLength+2])
						start := 2 + 2 + 2 + psm.ProgramStreamInfoLength + 2

						for i := 0; i < int(psm.ElementStreamMapLength); {
							info := &StreamInfo{
								StreamType:    hex.EncodeToString(psDataWithoutPrefix[int(start)+i : int(start)+i+1]),
								StreamTypeVal: StreamTypeValMap[psDataWithoutPrefix[int(start)+i]],
								StreamID:      hex.EncodeToString(psDataWithoutPrefix[int(start)+i+1 : int(start)+i+2]),
								StreamIDVal:   StreamIDValMap[psDataWithoutPrefix[int(start)+i+1]],
								Len:           binary.BigEndian.Uint16(psDataWithoutPrefix[int(start)+i+2 : int(start)+i+4]),
							}
							i += 4 + int(info.Len)
							info.Message()
							psm.StreamInfoList = append(psm.StreamInfoList, info)
						}
						ps.PSM = psm

						rawData = psDataWithoutPrefix[len+2:]
					} else if startCode == 0x000001e0 || startCode == 0x000001c0 { // PES视频流或PES音频流起始码
						// PES
						// 两字节为长度
						l := binary.BigEndian.Uint16(psDataWithoutPrefix[:2])
						// 第三个字节跳过
						// 第四个字节的前两位是PTS_DTS_flag
						//ptsFlag := false

						pes := &PESInfo{}
						pes.PESHeader = &PESHeader{
							StreamIDVal:         StreamIDValMap[uint8(startCode&0xFF)],
							StreamID:            strconv.FormatUint(uint64(startCode&0xFF), 16),
							PESLen:              6 + l,
							PtsDtsFlags:         hex.EncodeToString(psDataWithoutPrefix[3:4]),
							PESHeaderDataLength: psDataWithoutPrefix[4],
						}
						pes.Body = psDataWithoutPrefix[5+psDataWithoutPrefix[4] : 2+l]

						// 第四个字节的前两位是PTS_DTS_flag
						if (psDataWithoutPrefix[3] & 0x80) > 0 { // 只有PTS或者 PTS和DTS都有
							withPtsPesCount++
							//ptsFlag = true
							// 第五个字节是扩展数据长度，有PTS数据时，扩展数据长度不校验，直接按大于5算
							pes.Pts = getPts(psDataWithoutPrefix[5:10])
							pes.PtsHex = hex.EncodeToString(psDataWithoutPrefix[5:10])
							if pes.StreamID == "e0" { // 视频
								withPtsPesVideoCount++
								pes.LastPts = lastVideoPts
								pes.PtsDuration = pes.Pts - lastVideoPts
								frame.Pts = pes.Pts
								frame.PtsHex = pes.PtsHex
								frame.LastPts = lastVideoPts
								frame.PtsDuration = pes.Pts - lastVideoPts
								if lastVideoPts == 0 {
									frame.PtsDuration = 0
								}

								if lastVideoPes != nil {
									if lastVideoPes.PS.FrameInfo.IDNumber != frameNumber {
										// 跳变的pts,阈值可以使用帧率计算，也可以使用方差
										//if len(frameRateList) > 0 && pes.PtsDuration > int64(90000/frameRateList[len(frameRateList)-1]) ||
										if len(frameRateList) > 0 && pes.PtsDuration > 34483*2 ||
											pes.PtsDuration < 0 {

											frame.IsJump = true

											jumpDiffFrameVideoList = append(jumpDiffFrameVideoList, &JumpDiffFrameDisplay{
												IDNumber:    frameNumber,
												IsKey:       frame.IsKey,
												Pts:         pes.Pts,
												LastPts:     pes.LastPts,
												PtsDuration: pes.PtsDuration,
												RtpSeq:      fmt.Sprint(sameFrameRtpSeq),
											})
											jumpDiffFrameVideoList[len(jumpDiffFrameVideoList)-1].Message()
										} else if pes.PtsDuration > 0 { // 正常的pts
											tmpRatePts = float64(90000.0 / pes.PtsDuration)
											frameRateListPts = append(frameRateListPts, tmpRatePts)
										}
										pesDuarationOuterFrameVideo = append(pesDuarationOuterFrameVideo, pes.PtsDuration)
									} else {
										//if len(frameRateList) > 0 && pes.PtsDuration > int64(90000/frameRateList[len(frameRateList)-1]) ||
										if len(frameRateList) > 0 && pes.PtsDuration > 34483*2 ||
											pes.PtsDuration < 0 {
											jumpInnerFrameVideo = append(jumpInnerFrameVideo, frameNumber)
										}
										pesDuarationInnerFrameVideo = append(pesDuarationInnerFrameVideo, pes.PtsDuration)
									}
								}

								lastVideoPts = pes.Pts
								lastVideoPes = pes
							} else {
								withPtsPesAudioCount++
								pes.LastPts = lastAudioPts
								pes.PtsDuration = pes.Pts - lastAudioPts
								frame.Pts = pes.Pts
								frame.PtsHex = pes.PtsHex
								frame.LastPts = lastAudioPts
								frame.PtsDuration = pes.Pts - lastAudioPts
								if lastAudioPts == 0 {
									frame.PtsDuration = 0
								}
								if lastAudioPes != nil {
									if lastAudioPes.PS.FrameInfo.IDNumber != frameNumber {
										// 音频pts跳变阈值这样算？
										if len(frameRateList) > 0 && pes.PtsDuration > int64(90000.0/frameRateList[len(frameRateList)-1]) || pes.PtsDuration < 0 {
											jumpDiffFrameAudioList = append(jumpDiffFrameAudioList, &JumpDiffFrameDisplay{
												IDNumber:    frameNumber,
												IsKey:       frame.IsKey,
												Pts:         pes.Pts,
												LastPts:     lastAudioPts,
												PtsDuration: pes.PtsDuration,
											})
											frame.IsJump = true
										}
										pesDuarationOuterFrameAudio = append(pesDuarationOuterFrameAudio, pes.PtsDuration)

									} else {
										// 音频pts跳变阈值这样算？
										if len(frameRateList) > 0 && pes.PtsDuration > int64(90000.0/frameRateList[len(frameRateList)-1]) || pes.PtsDuration < 0 {
											jumpInnerFrameAudio = append(jumpInnerFrameAudio, frameNumber)
										}
										pesDuarationInnerFrameAudio = append(pesDuarationInnerFrameAudio, pes.PtsDuration)
									}
								}

								lastAudioPts = pes.Pts
								lastAudioPes = pes
							}
						}
						// 一个PES被切割完毕
						pes.PS = ps
						pes.Message()
						ps.PESInfoList = append(ps.PESInfoList, pes)
						ps.PESCount = len(ps.PESInfoList)
						ps.Message()
						if startCode == 0x000001e0 {
							//ps.PESInfoVideoList = append(ps.PESInfoVideoList, pes)
							allVideoPES = append(allVideoPES, pes)
							ps.PESInfoVideoList = append(ps.PESInfoVideoList, pes)
						} else if startCode == 0x000001c0 {
							allAudioPES = append(allAudioPES, pes)
							ps.PESInfoAudioList = append(ps.PESInfoAudioList, pes)
						}
						totalPES++

						rawData = psDataWithoutPrefix[l+2:]
					}
					if len(ps.PESInfoVideoList) > 0 && len(ps.PESInfoAudioList) > 0 {
						panic("") // 检查一下：一个PS里是否会既有视频又有音频的PES
					}

					if len(rawData) > 0 {
						continue
					} else {
						break
					}
				}

				aggregatedData = aggregatedData[:0]

				frame.IsKey = isKey
				frame.IDNumber = frameNumber
				frame.RtpSeqListStr = fmt.Sprintf("%v", sameFrameRtpSeq)
				frame.Message()
				if len(sameFrameRtpSeq) == 1 {
					v := &OnlyOneFrameForRtpInfo{
						FrameNumberID: frameNumber,
						RtpSeq:        sameFrameRtpSeq[0],
						FrameType:     frame.PSList[0].PESInfoList[0].StreamIDVal,
					}
					v.Message()
					if v.FrameType == "audio" {
						onlyOneFrameForRtp.OnlyOneFrameForRtpInfosA = append(onlyOneFrameForRtp.OnlyOneFrameForRtpInfosA, v)
					} else {
						onlyOneFrameForRtp.OnlyOneFrameForRtpInfosV = append(onlyOneFrameForRtp.OnlyOneFrameForRtpInfosV, v)
					}
				}
				sameFrameRtpSeq = sameFrameRtpSeq[:0]

				if frame.PSList[0].PESInfoList[0].PESHeader.StreamIDVal == "video" {
					videoFrameNo = append(videoFrameNo, frameNumber)
					videoFrame = append(videoFrame, frame)
				} else {
					audioFrameNo = append(audioFrameNo, frameNumber)
					audioFrame = append(audioFrame, frame)
				}

				frameInfoList = append(frameInfoList, frame)
				if frame.IsKey {
					frame.Message()
					keyFrameList = append(keyFrameList, frame)
				}
			}
		}

		frameRate = int(utils.CalculateMeanFloat64(frameRateList))
		frameRatepts = int(utils.CalculateMeanFloat64(frameRateListPts))
		// 计算方差
		//cv1 := utils.CalculateStandardDeviationInt64(pesDuarationOuterFrameAudio)
		//cv2 := utils.CalculateStandardDeviationInt64(pesDuarationOuterFrameVideo)
		//cv3 := utils.CalculateStandardDeviationInt64(pesDuarationInnerFrameAudio)
		//cv4 := utils.CalculateStandardDeviationInt64(pesDuarationInnerFrameVideo)

		frameNumberString := utils.GetFrameNumberString(videoFrameNo, audioFrameNo)

		// 结果集封装
		result := &Result{}

		result.FrameAnalysis = &FrameAnalysis{
			TotalFrame:             len(frameInfoList),
			VideoFrameCount:        len(videoFrameNo),
			AudioFrameCount:        len(audioFrameNo),
			VideoFrame:             videoFrame,
			AudioFrame:             audioFrame,
			KeyFrame:               keyFrameList,
			JumpDiffFrameVideoList: jumpDiffFrameVideoList,
			JumpDiffFrameAudioList: jumpDiffFrameAudioList,
			TotalPS:                totalPS,
			TotalPES:               totalPES,
			OnlyOneFrameForRtp:     onlyOneFrameForRtp,
			FrameRateByRtpTs:       frameRate,
			FrameRateByPts:         frameRatepts,
			FramRtptsJump:          framRtptsJump,
			AudioPES:               allAudioPES,
			VideoPES:               allVideoPES,
			FrameNumberString:      frameNumberString,
		}
		result.RTPAnalysis = &RTPAnalysis{
			TotalRtp: len(RtpList),
			//LastRtpHeader: []*protocol.RTPHeader{
			//	RtpList[len(RtpList)-1].Header,
			//	RtpList[len(RtpList)-2].Header,
			//	RtpList[len(RtpList)-3].Header,
			//},
			//PreRtpHeader: []*protocol.RTPHeader{
			//	RtpList[0].Header,
			//	RtpList[1].Header,
			//	RtpList[2].Header,
			//},
			LostSeqNumber: lostRtpSeqNumbers,
			//SameTimestampRtpSeq: sameTimestampRtpSeq,
		}

		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		jsonData, err := json.MarshalIndent(result, "", "")
		if err != nil {
			fmt.Println("Failed to serialize JSON:", err)
			return
		}

		//file, err := os.OpenFile("./res.json", os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
		jsonpath := "./res.json"
		if err := os.WriteFile(jsonpath, jsonData, os.ModePerm); err != nil {
			fmt.Println(err.Error())
			return
		}

	}
}

var (
	RtpList          []*protocol.RTP
	RtpStreamList    [][]byte
	tcpPayloadStream []byte
	count            int
)

const ptsThreshold = 3600

// ps包的开始
func isPsStartCodeValid(startCode uint32) bool {
	switch startCode {
	case 0x000001ba:
		return true
	case 0x000001bb:
		return true
	case 0x000001bc:
		return true
	case 0x000001e0:
		return true
	case 0x000001c0:
		return true
	default:
		return false
	}

	// 0x000001BD私有数据
}

func getPts(ptsBuf []byte) int64 {
	psPts := int64(ptsBuf[0]&0x0E) << 29
	psPts |= int64(ptsBuf[1]) << 22
	psPts |= int64(ptsBuf[2]&0xFE) << 14
	psPts |= int64(ptsBuf[3]) << 7
	psPts |= int64(ptsBuf[4]&0xFE) >> 1

	return psPts
}

func modifyPts(pts int64, ptsBuf []byte) {
	ptsBuf[0] = (uint8)(((pts>>30)&0x07)<<1) | 0x20 | 0x01
	ptsBuf[1] = (uint8)((pts >> 22) & 0xff)
	ptsBuf[2] = (uint8)(((pts>>15)&0xff)<<1) | 0x01
	ptsBuf[3] = (uint8)((pts >> 7) & 0xff)
	ptsBuf[4] = (uint8)((pts&0xff)<<1) | 0x01
}

func ParsePayloadFile(filepath string) ([]byte, error) {
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func getPsHeaderPrefixHexInfo(psData []byte, bytes int) string {
	psDataHeader := psData[:bytes]
	hex := make([]string, len(psDataHeader))
	for i, b := range psDataHeader {
		hex[i] = fmt.Sprintf("%02x", b)
	}

	return strings.Join(hex, " ")
}

func ParsePcapngFile(filepath string) ([]byte, error) {
	pcapFile, err := os.Open(filepath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer pcapFile.Close()

	pcapReader, err := pcapng.NewReader(pcapFile)
	if err != nil {
		fmt.Println("Error creating pcap reader:", err)
		return nil, err
	}
	packetSource := gopacket.NewPacketSource(pcapReader, pcapReader.LinkType())
	var (
		i       = 1
		lastTCP *layers.TCP
	)
	for packet := range packetSource.Packets() {
		if etherLayer := packet.Layer(layers.LayerTypeEthernet); etherLayer != nil {
			ethernet := etherLayer.(*layers.Ethernet)
			if ethernet.EthernetType != layers.EthernetTypeIPv4 {
				i++
				continue
			}
		}

		var ip *layers.IPv4
		if iplayer := packet.Layer(layers.LayerTypeIPv4); iplayer != nil {
			ip, _ = iplayer.(*layers.IPv4)
			if !(ip.SrcIP.String() == srcAddr) {
				i++
				continue
			}
		}

		var tcpLayer *layers.TCP
		if tcplayer := packet.Layer(layers.LayerTypeTCP); tcplayer != nil {
			tcpLayer, _ = tcplayer.(*layers.TCP)
		} else {
			i++
			continue
		}

		// TCP分段重叠部分不要加入最终流
		if lastTCP == nil || lastTCP.Seq < tcpLayer.Seq {
			tcpPayloadStream = append(tcpPayloadStream, tcpLayer.Payload...)
		}

		if len(tcpLayer.Payload) > 0 {
			lastTCP = tcpLayer
		}
		i++
		packCount++
	}
	return tcpPayloadStream, nil
}

const RTPHeaderLen = 12

func parseRTPHeader(data []byte, rtp *protocol.RTP) {
	packet := data[:RTPHeaderLen]
	rtp.Header = new(protocol.RTPHeader)
	rtp.Header.Version = (packet[0] & 0xC0) >> 6
	rtp.Header.Padding = (packet[0] & 0x20) != 0
	rtp.Header.Extension = (packet[0] & 0x10) != 0
	rtp.Header.CSRCCount = packet[0] & 0x0F //这个假设都为0，不管
	rtp.Header.Marker = (packet[1] & 0x80) != 0
	rtp.Header.PayloadType = packet[1] & 0x7F
	rtp.Header.SequenceNumber = binary.BigEndian.Uint16(packet[2:4])
	rtp.Header.Timestamp = binary.BigEndian.Uint32(packet[4:8])
	rtp.Header.SSRC = binary.BigEndian.Uint32(packet[8:12])

	CSRCLen := int(rtp.Header.CSRCCount) * 4 // CSRC每项长度是4字节, CSRCList占的字节数
	if rtp.Header.CSRCCount > 0 && len(data) > CSRCLen {
		rtp.Header.CSRCList = make([]uint32, rtp.Header.CSRCCount)
		packet1 := data[:RTPHeaderLen+CSRCLen]
		for i := 0; i < int(rtp.Header.CSRCCount); i++ {
			CSRC := binary.BigEndian.Uint32(packet1[i*4 : i*4+4])
			rtp.Header.CSRCList = append(rtp.Header.CSRCList, CSRC)
		}
	}

	rtp.Header.HeaderBytes = hex.EncodeToString(data[:RTPHeaderLen+CSRCLen])
	rtp.Payload = data[RTPHeaderLen+CSRCLen:]
}

var StreamIDValMap = map[uint8]string{
	0xe0: "video",
	0xc0: "audio",
}

var StreamTypeValMap = map[uint8]string{
	0x03: "MPEG-1",
	0x10: "MPEG-4",
	0x02: "MPEG-2",
	0x1b: "H.264",
	0x24: "H.265/HEVC",
	0x80: "SVAC_video",
	0x90: "G.711",
	0x92: "G.722.1",
	0x93: "G.723.1",
	0x99: "G.729",
	0x9b: "SVAC_audio",
}

var RTPPayloadTypeClock = map[uint8]int{
	96: 90000,
}

type PESInfo struct {
	*PESHeader `json:"PES Header"`
	Body       []byte `json:"-"` // body 是PESHeaderDataLength字段之后跳过PESHeaderDataLength个字节

	// 所属的ps
	PS  *PSInfo `json:"-"`
	Msg string  `json:"msg"`
}

type PESHeader struct {
	// 长度字段之后，PES的header是这样：
	// 2个字节、一个字节的PESHeaderDataLength， PESHeaderDataLength个字节
	StreamID            string `json:"stream_id"` // 负载流类型
	StreamIDVal         string `json:"stream_id[值]"`
	PESLen              uint16 `json:"PES总长度"`        // PES 总长度, 包括header和body
	PtsDtsFlags         string `json:"pts_dts_flags"` // pts_dts_flags
	PESHeaderDataLength uint8  `json:"PES header 长度"` // PES header 长度
	// 视频和音频需要区分开
	Pts         int64  `json:"Pts"`
	PtsHex      string `json:"Pts[HEX]"`
	LastPts     int64  `json:"上一个pes包的pts"`
	PtsDuration int64  `json:"和上一个pes包的pts之差"`
	Dts         int64  `json:"dts"`
	LastDts     int64  `json:"上一个pes包的dts"`
	DtsDuration int64  `json:"和上一个pes包的dts之差"`
}

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

type FrameInfo struct {
	IDNumber int64 `json:"帧序号"`
	IsKey    bool  `json:"是否关键帧"` // 是否关键帧
	//RtpSeqList    []uint16  `json:"携带此帧数据的RTP包的序号"`
	RtpSeqListStr string    `json:"携带此帧数据的RTP包的序号"`
	PSList        []*PSInfo `json:"此帧的PS包列表"` // 一帧数据可能有多个PS包

	Pts         int64  `json:"本帧的pts"`
	PtsHex      string `json:"本帧的pts[HEX]"`
	LastPts     int64  `json:"上一帧的pts"`
	PtsDuration int64  `json:"pts之差"`
	IsJump      bool   `json:"是否跳变"`

	FrameInfoImportMsg string `json:"msg"`
}

type FrameInfoWithPts struct {
	VideoPts int64
	AudioPts int64
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

// FrameAnalysis 帧分析结果
type FrameAnalysis struct {
	TotalFrame             int                     `json:"总帧数"`
	VideoFrameCount        int                     `json:"视频帧总数"`
	AudioFrameCount        int                     `json:"音频帧总数"`
	VideoFrame             []*FrameInfo            `json:"视频帧"`
	AudioFrame             []*FrameInfo            `json:"音频帧"`
	KeyFrame               []*FrameInfo            `json:"关键帧详情"`
	JumpDiffFrameVideoList []*JumpDiffFrameDisplay `json:"帧间pts跳变[video]"`
	JumpDiffFrameAudioList []*JumpDiffFrameDisplay `json:"帧间pts跳变[audio]"`
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
	Dts         int64  `json:"-"`
	LastDts     int64  `json:"-"`
	DtsDuration int64  `json:"-"`

	Msg string `json:"msg"`
}

type Result struct {
	*FrameAnalysis `json:"帧分析结果"`
	*RTPAnalysis   `json:"RTP分析结果"`
}

// RTPAnalysis RTP分析结果
type RTPAnalysis struct {
	TotalRtp      int                   `json:"RTP包总数"`
	LastRtpHeader []*protocol.RTPHeader `json:"-"` // 最后三个RTP的头信息
	PreRtpHeader  []*protocol.RTPHeader `json:"-"` // 前三个RTP的头信息
	LostSeqNumber []uint16              `json:"丢失的RTP包的Seq"`
	//SameTimestampRtpSeq []uint16              `json:"有相同timestamp的rtp的序号"`
}

type Msg interface {
	Message()
}

func (f *FrameInfo) Message() {
	f.FrameInfoImportMsg = fmt.Sprintf(`帧号:%d,是否关键帧:%v,此帧的pts:%v,此帧的pts[HEX]:%s,上一帧的pts:%v,pts差:%v,是否跳变:%v,
	此帧含有的PS包个数:%d,携带此帧的RTP序列号:%s`,
		f.IDNumber, f.IsKey, f.PSList[0].PESInfoList[0].Pts, f.PSList[0].PESInfoList[0].PtsHex, f.LastPts, f.PtsDuration, f.IsJump,
		len(f.PSList), f.RtpSeqListStr)
}

func (f *JumpDiffFrameDisplay) Message() {
	f.Msg = fmt.Sprintf("帧号:%d, 是否关键帧:%v, 此帧的pts:%v, 上一帧的pts:%d, pts之差:%d, 携带此帧的rtp序列号:%s",
		f.IDNumber, f.IsKey, f.Pts, f.LastPts, f.PtsDuration, f.RtpSeq)
}

func (o *OnlyOneFrameForRtpInfo) Message() {
	o.Msg = fmt.Sprintf("帧号:%d, 携带此帧的RTP的序列号:%v, 帧类型:%v",
		o.FrameNumberID, o.RtpSeq, o.FrameType)
}

func (f *PSInfo) Message() {
	f.PSInfoImportMsg = fmt.Sprintf("此PS包下的PES包总数:%d", f.PESCount)
}

func (in *StreamInfo) Message() {
	in.Msg = fmt.Sprintf("流类型:%v, 码流类型:%v", in.StreamIDVal, in.StreamTypeVal)
}

func (p *PESInfo) Message() {
	p.Msg = fmt.Sprintf("流类型:%v, Pts:%d, 上一个pts:%d", p.StreamIDVal, p.Pts, p.LastPts)
}

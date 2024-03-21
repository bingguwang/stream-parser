package protocol

import "fmt"

type TCPMessage struct {
	SourcePort                      []byte // 源端口号 2B
	DestinationPort                 []byte // 目的端口号 2B
	SequenceNumber                  []byte // 序列号 4B
	AcknowledgeNumber               []byte // 确认序列号 4B
	HeaderLengthAndReservedAndFlags []byte // 头部长度 4位 + 保留位 6位 + 标志位 6位 = 16 位
	Window                          []byte // 窗口大小 2B
	CheckSum                        []byte // 校验和 2B
	UrgentPointer                   []byte // 紧急指针 2B

	SourcePortVal      string
	DestinationPortVal string
	HeaderLength       uint64 // TCP报文头部的长度
	Payload            []byte // 负载
	SequenceNumberVal  uint64 // 序列号

	TCPSegmentList []TCPSegment // tcp分段
}

type TCPSegment struct {
	ID     int // 数据包id
	Length int // 段长度
}

func (m *TCPMessage) ToString() {
	fmt.Printf(
		"SourcePort:%x\n"+
			"DestinationPort:%x\n"+
			"SequenceNumber:%x\n"+
			"AcknowledgeNumber:%x\n"+
			"HeaderLengthAndReservedAndFlags:%x\n"+
			"Window:%x\n"+
			"CheckSum:%x\n"+
			"UrgentPointer:%x\n",
		m.SourcePort,
		m.DestinationPort,
		m.SequenceNumber,
		m.AcknowledgeNumber,
		m.HeaderLengthAndReservedAndFlags,
		m.Window,
		m.CheckSum,
		m.UrgentPointer,
	)
}

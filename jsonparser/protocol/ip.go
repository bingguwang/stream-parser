package protocol

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
)

const (
	ICMP = 1
	IGMP = 2
	TCP  = 6
	UDP  = 7
	IGRP = 88
	OSPF = 89
)

type IPMessage struct {
	IpVersionAndIpHeaderLength []byte // ip版本 4位+ip头长度 4位  1B
	TypeofService              []byte // 服务类型 1B
	IpTotalLength              []byte // ip报文总长度(包括数据和头部在内) 2B
	Identification             []byte // 标识符 2B
	FlagsAndFragmentOffset     []byte // 标记 3位 + 片偏移 13位 2B
	TTL                        []byte // ttl 1B
	Protocol                   []byte // 上层所使用的协议 1B
	HeaderChecksum             []byte // 头部校验 2B
	SourceAddr                 []byte // IP包原地址 2B
	DestinationAddr            []byte // IP包目的地址 2B
	IpHeaderLen                uint64 // ip头长度
	IpTotalLen                 uint64 // ip报文总长度(包括数据和头部在内)

	SourceAddrVal      string
	DestinationAddrVal string
	UnderLayerMsg      []byte
	*TCPMessage
}

func (m *IPMessage) SetTCPMessage(tcpData []byte) error {
	proto, _ := strconv.ParseInt(hex.EncodeToString(m.Protocol), 10, 64)
	if proto != TCP {
		return errors.New("protocol is not TCP")
	}
	if len(tcpData) < 12 {
		return errors.New("length of tcpData is too short")
	}

	if m.TCPMessage == nil {
		m.TCPMessage = new(TCPMessage)
	}

	m.TCPMessage.SourcePort = tcpData[0:2]
	m.TCPMessage.DestinationPort = tcpData[2:4]
	m.TCPMessage.SequenceNumber = tcpData[4:8]
	toString := hex.EncodeToString(m.TCPMessage.SequenceNumber)
	m.TCPMessage.SequenceNumberVal, _ = strconv.ParseUint(toString, 16, 64)

	m.TCPMessage.AcknowledgeNumber = tcpData[8:12]
	m.TCPMessage.Window = tcpData[14:16]
	m.TCPMessage.HeaderLengthAndReservedAndFlags = tcpData[12:14]
	headLength, _ := strconv.ParseUint(hex.EncodeToString(tcpData[12:13])[:1], 16, 64)
	m.TCPMessage.HeaderLength = headLength * 4 // 4字节一个长度单位

	m.TCPMessage.CheckSum = tcpData[16:18]
	m.TCPMessage.UrgentPointer = tcpData[18:20]

	m.TCPMessage.Payload = tcpData[m.TCPMessage.HeaderLength:]

	return nil
}

func (m *IPMessage) ToString() {
	fmt.Printf(
		"IpVersionAndIpHeaderLength:%x\n"+
			"TypeofService:%x\n"+
			"IpTotalLength:%x\n"+
			"Identification:%x\n"+
			"FlagsAndFragmentOffset:%x\n"+
			"TTL:%x\n"+
			"Protocol:%x\n"+
			"HeaderChecksum:%x\n"+
			"SourceAddr:%x\n"+
			"DestinationAddr:%x\n"+
			"IpHeaderLen:%x\n"+
			"IpTotalLen:%x\n",
		m.IpVersionAndIpHeaderLength,
		m.TypeofService,
		m.IpTotalLength,
		m.Identification,
		m.FlagsAndFragmentOffset,
		m.TTL,
		m.Protocol,
		m.HeaderChecksum,
		m.SourceAddr,
		m.DestinationAddr,
		m.IpHeaderLen,
		m.IpTotalLen,
	)
}

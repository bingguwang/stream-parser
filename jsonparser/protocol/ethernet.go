package protocol

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"strconv"
	"strings"
)

const (
	ipv4Type = "0800"
	ipv6Type = "86DD"
	arpType  = "0806"
)

type LayerMsg interface {
	ToString() string
}

type EthernetMessage struct {
	Id             uint64
	SourceMac      []byte // 源MAC地址
	DestinationMac []byte // 目的MAC地址
	EthernetType   []byte // 以太网类型

	*IPMessage
}

func ParseEthernetMsg(ethernetData []byte) (*EthernetMessage, error) {
	if len(ethernetData) != 14 {
		return nil, errors.New("ParseEthernetMsg failed: length of Ethernet content is not 14")
	}
	res := &EthernetMessage{}
	res.SourceMac = ethernetData[0:6]
	res.DestinationMac = ethernetData[6:12]
	res.EthernetType = ethernetData[12:14]

	return res, nil
}

// SetIPv4Message leaveData是除去前14字节之后的数据
func (m *EthernetMessage) SetIPv4Message(leaveData []byte) error {
	if hex.EncodeToString(m.EthernetType) != ipv4Type {
		return errors.New("type is not ipv4 ")
	}
	if m.IPMessage == nil {
		m.IPMessage = new(IPMessage)
	}
	if len(leaveData) < 20 {
		return errors.New("length of ipdata is too short ")
	}

	ipTotalLen := leaveData[2:4]
	m.IpTotalLength = ipTotalLen
	m.IpTotalLen, _ = strconv.ParseUint(hex.EncodeToString(ipTotalLen), 16, 64)

	realIPData := leaveData[:m.IpTotalLen]
	len, _ := strconv.ParseUint(hex.EncodeToString(realIPData[0:1])[1:], 16, 64)
	m.IpHeaderLen = len * 4

	m.TypeofService = realIPData[1:2]
	m.Identification = realIPData[4:6]
	m.FlagsAndFragmentOffset = realIPData[6:8]
	m.TTL = realIPData[8:9]
	m.Protocol = realIPData[9:10]
	m.HeaderChecksum = realIPData[10:12]
	m.SourceAddr = realIPData[12:16]
	m.DestinationAddr = realIPData[16:20]
	var ipStr string
	for _, by := range m.SourceAddr {
		ipStr += fmt.Sprintf("%d.", by)
	}
	m.SourceAddrVal = strings.TrimRight(ipStr, ".")
	ipStr = ""
	for _, by := range m.DestinationAddr {
		ipStr += fmt.Sprintf("%d.", by)
	}
	m.DestinationAddrVal = strings.TrimRight(ipStr, ".")

	m.UnderLayerMsg = realIPData[20:]

	return nil
}

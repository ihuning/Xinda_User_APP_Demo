// 数据交换文件的头部的各个字段和相关方法,使用接收方公钥加密.
//	接收方通过头部可以得知这个数据交换文件的基本信息,从而决定是否解密该分片以及如何从该分片还原出原始文件
package header

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// 头部的组成字段
type Header struct {
	SenderName     [20]byte  // 发送者的代号
	ReceiverName   [20]byte  // 接收者的代号
	FileName       [255]byte // 待发送的原文件（不是数据交换文件）的文件名
	Identification int32     // 本通信过程的标识(发送和接收过程保持一致)
	FileDataLength int32     // 对称加密之前数据交换文件的数据部分长度(加密后会多16字节)
	Timer          int32     // 发送方能接受的最长等待时间
	DivideMethod   int8      // 数据分片方式(分为2/4/8片)
	GroupNum       int8      // 分组的数量
	GroupSN        int8      // 冗余分组序列号
	FragmentSN     int8      // 数据分片序号(如果是冗余分片,则序号为-1)
	GroupContent   [8]int8   // 本冗余分组中所有数据分片的FragmentSN
}

// 生成一个头部结构体,并将头部结构体转为对应的bytes
func GenerateHeaderBytes(senderName, receiverName, fileName string, identification, fileDataLength, timer int32, divideMethod, groupNum, groupSN, fragmentSN int8, groupContent []int8) ([]byte, error) {
	var header *Header = &Header{}
	header.SetSenderName(senderName)
	header.SetReceiverName(receiverName)
	header.SetFileName(fileName)
	header.SetIdentification(identification)
	header.SetFileDataLength(fileDataLength)
	header.SetTimer(timer)
	header.SetDivideMethod(divideMethod)
	header.SetGroupNum(groupNum)
	header.SetGroupSN(groupSN)
	header.SetFragmentSN(fragmentSN)
	header.SetGroupContent(groupContent)
	headerBytes, err := header.HeaderToBytes()
	return headerBytes, err
}

// 将header bytes还原为header结构体
func BytesToHeader(readBytes []byte) (Header, error) {
	var header *Header = &Header{}
	buf := new(bytes.Buffer)
	buf.Write(readBytes)
	err := binary.Read(buf, binary.BigEndian, header)
	if err != nil {
		fmt.Println("无法成功将bytes转为header", err)
		return *header, err
	}
	return *header, err
}

// 将header结构体转为header bytes
func (h Header) HeaderToBytes() ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, h)
	if err != nil {
		fmt.Println("无法成功将header转为bytes", err)
		return nil, err
	}
	headerBytes := buf.Bytes()
	return headerBytes, err
}

// 获得header结构体转为bytes以后所占用的空间
func GetHeaderBytesSize() int {
	var header Header
	headerBytes, _ := header.HeaderToBytes()
	return len(headerBytes)
}

// 获得SenderName
func (h Header) GetSenderName() string {
	var senderNameBytes []byte
	for _, v := range h.SenderName {
		if v == 0 {
			break
		}
		senderNameBytes = append(senderNameBytes, v)
	}
	return string(senderNameBytes)
}

// 获得ReceiverName
func (h Header) GetReceiverName() string {
	var receiverNameBytes []byte
	for _, v := range h.ReceiverName {
		if v == 0 {
			break
		}
		receiverNameBytes = append(receiverNameBytes, v)
	}
	return string(receiverNameBytes)
}

// 获得FileName
func (h Header) GetFileName() string {
	var fileNameBytes []byte
	for _, v := range h.FileName {
		if v == 0 {
			break
		}
		fileNameBytes = append(fileNameBytes, v)
	}
	return string(fileNameBytes)
}

// 获得Identification
func (h Header) GetIdentification() int32 {
	return h.Identification
}

// 获得GetFileDataLength
func (h Header) GetFileDataLength() int32 {
	return h.FileDataLength
}

// 获得Timer
func (h Header) GetTimer() int32 {
	return h.Timer
}

// 获得DivideMethod
func (h Header) GetDivideMethod() int8 {
	return h.DivideMethod
}

// 获得GroupNum
func (h Header) GetGroupNum() int8 {
	return h.GroupNum
}

// 获得GroupSN
func (h Header) GetGroupSN() int8 {
	return h.GroupSN
}

// 获得FragmentSN
func (h Header) GetFragmentSN() int8 {
	return h.FragmentSN
}

// 获得GroupContent
func (h Header) GetGroupContent() []int8 {
	var groupContent []int8
	for _, v := range h.GroupContent {
		if v == -1 {
			break
		}
		groupContent = append(groupContent, v)
	}
	return groupContent
}

// 设定SenderName
func (h *Header) SetSenderName(senderName string) {
	senderNameBytes := []byte(senderName)
	copy((*h).SenderName[:len(senderNameBytes)], senderNameBytes)
}

// 设定ReceiverName
func (h *Header) SetReceiverName(receiverName string) {
	receiverNameBytes := []byte(receiverName)
	copy((*h).ReceiverName[:len(receiverNameBytes)], receiverNameBytes)
}

// 设定FileName
func (h *Header) SetFileName(fileName string) {
	fileNameBytes := []byte(fileName)
	copy((*h).FileName[:len(fileNameBytes)], fileNameBytes)
}

// 设定Identification
func (h *Header) SetIdentification(identification int32) {
	(*h).Identification = identification
}

// 设定FileDataLength
func (h *Header) SetFileDataLength(fileDataLength int32) {
	(*h).FileDataLength = fileDataLength
}

// 设定Timer
func (h *Header) SetTimer(timer int32) {
	(*h).Timer = timer
}

// 设定DivideMethod
func (h *Header) SetDivideMethod(divideMethod int8) {
	(*h).DivideMethod = divideMethod
}

// 设定GroupNum
func (h *Header) SetGroupNum(groupNum int8) {
	(*h).GroupNum = groupNum
}

// 设定GroupSN
func (h *Header) SetGroupSN(groupSN int8) {
	(*h).GroupSN = groupSN
}

// 设定FragmentSN
func (h *Header) SetFragmentSN(fragmentSN int8) {
	(*h).FragmentSN = fragmentSN
}

// 设定GroupConten
func (h *Header) SetGroupContent(groupContent []int8) {
	(*h).GroupContent = [8]int8{-1, -1, -1, -1, -1, -1, -1, -1}
	copy((*h).GroupContent[:len(groupContent)], groupContent)
}

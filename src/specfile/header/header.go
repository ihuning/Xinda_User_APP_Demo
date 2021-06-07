// 用于给二进制数据添加头部
package header

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Header struct {
	SenderName     [20]byte  // 发送者的代号
	ReceiverName   [20]byte  // 接收者的代号
	FileName       [255]byte // 待发送的原文件（不是数据交换文件）的文件名
	Identification int32     // 本通信过程的标识(发送和接收过程保持一致)
	FileDataLength int32     // 对称加密之前数据交换文件的数据部分长度(加密后会多16字节)
	Timer          int32     // 发送方能接受的最长等待时间
	DivideMethod   int8      // 数据分片方式(分为2/4/8片)
	GroupSN        int8      // 冗余分组序列号
	FragmentSN     int8      // 数据分片序号(如果是冗余分片,则序号为-1)
	GroupContent   [8]int8   // 本冗余分组中所有数据分片的FragmentSN
}

// 生成一个Header
func (h *Header) generateHeader(senderName, receiverName, fileName string, identification, fileDataLength, timer int32, divideMethod, groupSN, fragmentSN int8, groupContent []int8) {
	h.SetSenderName(senderName)
	h.SetReceiverName(receiverName)
	h.SetFileName(fileName)
	h.SetIdentification(identification)
	h.SetFileDataLength(fileDataLength)
	h.SetTimer(timer)
	h.SetDivideMethod(divideMethod)
	h.SetGroupSN(groupSN)
	h.SetFragmentSN(fragmentSN)
	h.SetGroupContent(groupContent)
}

// 将Header结构体转为bytes
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

// 得知header转为bytes占用的空间
func GetHeaderBytesSize() int {
	var header Header
	headerBytes, _ := header.HeaderToBytes()
	return len(headerBytes)
}

// 将以bytes形式存储的结构体还原回结构体
func (h *Header) BytesToHeader(readBytes []byte) error {
	var err error
	buf := new(bytes.Buffer)
	buf.Write(readBytes)
	err = binary.Read(buf, binary.BigEndian, h)
	if err != nil {
		fmt.Println("无法成功将bytes转为header", err)
		return err
	}
	return err
}

// 生成一个结构体,并将结构体转为对应的bytes
func GenerateHeaderBytes(senderName, receiverName, fileName string, identification, fileDataLength, timer int32, divideMethod, groupSN, fragmentSN int8, groupContent []int8) ([]byte, error) {
	var header *Header = &Header{}
	header.generateHeader(senderName, receiverName, fileName, identification, fileDataLength, timer, divideMethod, groupSN, fragmentSN, groupContent)
	headerBytes, err := header.HeaderToBytes()
	return headerBytes, err
}

// 将bytes还原为header
func ReadHeaderFromSpecFileBytes(bytes []byte) (Header, error) {
	var header *Header = &Header{}
	err := header.BytesToHeader(bytes)
	return *header, err
}

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

func (h Header) GetIdentification() int32 {
	return h.Identification
}

func (h Header) GetFileDataLength() int32 {
	return h.FileDataLength
}

func (h Header) GetTimer() int32 {
	return h.Timer
}

func (h Header) GetDivideMethod() int8 {
	return h.DivideMethod
}

func (h Header) GetGroupSN() int8 {
	return h.GroupSN
}

func (h Header) GetFragmentSN() int8 {
	return h.FragmentSN
}

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

func (h *Header) SetSenderName(senderName string) {
	senderNameBytes := []byte(senderName)
	copy((*h).SenderName[:len(senderNameBytes)], senderNameBytes)
}

func (h *Header) SetReceiverName(receiverName string) {
	receiverNameBytes := []byte(receiverName)
	copy((*h).ReceiverName[:len(receiverNameBytes)], receiverNameBytes)
}

func (h *Header) SetFileName(fileName string) {
	fileNameBytes := []byte(fileName)
	copy((*h).FileName[:len(fileNameBytes)], fileNameBytes)
}

func (h *Header) SetIdentification(identification int32) {
	(*h).Identification = identification
}

func (h *Header) SetFileDataLength(fileDataLength int32) {
	(*h).FileDataLength = fileDataLength
}

func (h *Header) SetTimer(timer int32) {
	(*h).Timer = timer
}

func (h *Header) SetDivideMethod(divideMethod int8) {
	(*h).DivideMethod = divideMethod
}

func (h *Header) SetGroupSN(groupSN int8) {
	(*h).GroupSN = groupSN
}

func (h *Header) SetFragmentSN(fragmentSN int8) {
	(*h).FragmentSN = fragmentSN
}

func (h *Header) SetGroupContent(groupContent []int8) {
	(*h).GroupContent = [8]int8{-1, -1, -1, -1, -1, -1, -1, -1}
	copy((*h).GroupContent[:len(groupContent)], groupContent)
}

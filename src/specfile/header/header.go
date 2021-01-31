// 用于给二进制数据添加头部
package header

import (
	"bytes"
	"encoding/binary"
	"fmt"
	// "unsafe"
)

const HEADER_BYTES_LENGTH = 328

type Header struct {
	SenderName         [20]byte  // 发送者的代号
	ReceiverName       [20]byte  // 接收者的代号
	FileName           [255]byte // 待传输文件的文件名
	FullDataLength     int32     // 待传输文件的长度
	Identification     int32     // 标识.一个文件的所有分片有相同的标识
	DivideMethod       int32     // 划分方式(2片?4片?8片?)
	FragmentDataLength int32     // 分片的数据部分长度
	FragmentSN         int32     // 分片序号
	Timer              int32     // 分片的TTL
	Nonce              int32     // 防止重放攻击的随机数
	RedundantSN        int32     // 冗余分组序列号
	RedundantTotal     int32     // 本冗余分组的分片总数
	IsRedundant        bool      // 是否为冗余分片：否/是
}

// 生成一个Header
func (h *Header) generateHeader(senderName, receiverName, fileName string, fullDataLength, identification, fragmentDataLength, divideMethod, fragmentSN, timer, nonce, redundantSN, redundantTotal int32, isRedundant bool) {
	senderNameBytes := []byte(senderName)
	copy((*h).SenderName[:len(senderNameBytes)], senderNameBytes)
	receiverNameBytes := []byte(receiverName)
	copy((*h).ReceiverName[:len(receiverNameBytes)], receiverNameBytes)
	fileNameBytes := []byte(fileName)
	copy((*h).FileName[:len(fileNameBytes)], fileNameBytes)
	(*h).FullDataLength = fullDataLength
	(*h).Identification = identification
	(*h).DivideMethod = divideMethod
	(*h).FragmentDataLength = fragmentDataLength
	(*h).FragmentSN = fragmentSN
	(*h).Timer = timer
	(*h).Nonce = nonce
	(*h).RedundantSN = redundantSN
	(*h).RedundantTotal = redundantTotal
	(*h).IsRedundant = isRedundant
}

// 将Header结构体转为bytes
func (h *Header) headerToBytes() ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, *h)
	if err != nil {
		fmt.Println("无法成功将header转为bytes", err)
		return nil, err
	}
	return buf.Bytes(), err
}

// 将以bytes形式存储的结构体还原回结构体
func (h *Header) bytesToHeader(readBytes []byte) error {
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
func GenerateHeaderBytes(senderName, receiverName, fileName string, fullDataLength, identification, divideMethod, fragmentDataLength, fragmentSN, timer, nonce, redundantSN, redundantTotal int32, isRedundant bool) ([]byte, error) {
	var header *Header = &Header{}
	header.generateHeader(senderName, receiverName, fileName, fullDataLength, identification, divideMethod, fragmentDataLength, fragmentSN, timer, nonce, redundantSN, redundantTotal, isRedundant)
	headerBytes, err := header.headerToBytes()
	return headerBytes, err
}

// 将bytes还原为header
func ReadHeaderFromSpecFileBytes(bytes []byte) (Header, error) {
	var header *Header = &Header{}
	err := header.bytesToHeader(bytes)
	return *header, err
}

func (h *Header) GetSenderName() string {
	var senderNameBytes []byte
	for _, v := range (*h).SenderName {
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

func (h Header) GetFullDataLength() int32 {
	return h.FullDataLength
}

func (h Header) GetIdentification() int32 {
	return h.Identification
}

func (h Header) GetDivideMethod() int32 {
	return h.DivideMethod
}

func (h Header) GetFragmentDataLength() int32 {
	return h.FragmentDataLength
}

func (h Header) GetFragmentSN() int32 {
	return h.FragmentSN
}

func (h Header) GetTimer() int32 {
	return h.Timer
}

func (h Header) GetNonce() int32 {
	return h.Nonce
}

func (h Header) GetRedundantSN() int32 {
	return h.RedundantSN
}

func (h Header) GetRedundantTotal() int32 {
	return h.RedundantTotal
}

func (h Header) GetIsRedundant() bool {
	return h.IsRedundant
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

func (h *Header) SetFullDataLength(fullDataLength int32) {
	(*h).FullDataLength = fullDataLength
}

func (h *Header) SetIdentification(identification int32) {
	(*h).Identification = identification
}

func (h *Header) SetDivideMethod(divideMethod int32) {
	(*h).DivideMethod = divideMethod
}

func (h *Header) SetFragmentDataLength(fragmentDataLength int32) {
	(*h).FragmentDataLength = fragmentDataLength
}

func (h *Header) SetFragmentSN(fragmentSN int32) {
	(*h).FragmentSN = fragmentSN
}

func (h *Header) SetTimer(timer int32) {
	(*h).Timer = timer
}

func (h *Header) SetNonce(nonce int32) {
	(*h).Nonce = nonce
}

func (h *Header) SetRedundantSN(redundantSN int32) {
	(*h).RedundantSN = redundantSN
}

func (h *Header) SetRedundantTotal(redundantTotal int32) {
	(*h).RedundantTotal = redundantTotal
}

func (h *Header) SetIsRedundant(isRedundant bool) {
	(*h).IsRedundant = isRedundant
}

package jsontools

import (
	// "bytes"
	"crypto/rand"
	"path/filepath"
	// "math"
	"math/big"
	// "strconv"
	"os"
)

// 前端生成的发送阶段配置json文件,返回生成的json的bytes
func GenerateSendStrategyJsonBytes(divideMethod, groupNum int, senderName, receiverName, srcFilePath string, timer int) []byte {
	jsonParser := GenerateNewJsonParser()
	fileInfo, _ := os.Stat(srcFilePath)
	identification, _ := rand.Int(rand.Reader, big.NewInt(2147483647))
	jsonParser.SetValue("send_strategy", "MsgType")
	jsonParser.SetValue(divideMethod, "DivideMethod")
	jsonParser.SetValue(groupNum, "GroupNum")
	jsonParser.SetValue(senderName, "SenderName")
	jsonParser.SetValue(receiverName, "ReceiverName")
	jsonParser.SetValue(srcFilePath, "SrcFilePath")
	jsonParser.SetValue(identification.Int64(), "Identification")
	jsonParser.SetValue(fileInfo.Size(), "FileDataLength")
	jsonParser.SetValue(timer, "Timer")
	return jsonParser.GenerateJsonBytes()
}

// 反馈给发送进程的发送进度
func GenerateSendProgressChannelJsonBytes(fileName, url, userName string, sendNum int) ([]byte,) {
	jsonParser := GenerateNewJsonParser()
	jsonParser.SetValue(fileName, "FileName")
	jsonParser.SetValue(url, "URL")
	jsonParser.SetValue(userName, "UserName")
	jsonParser.SetValue(sendNum, "SendNum")
	return jsonParser.GenerateJsonBytes()
}

// 反馈给接收进程的接收进度
func GenerateReceiveProgressChannelJsonBytes(fileName, url, userName string) []byte {
	jsonParser := GenerateNewJsonParser()
	jsonParser.SetValue(fileName, "FileName")
	jsonParser.SetValue(url, "URL")
	jsonParser.SetValue(userName, "UserName")
	return jsonParser.GenerateJsonBytes()
}

// 反馈给前端的发送进度
func GenerateSendProgressJsonBytes(srcFilePath string, identification, successSendNum, totalNum int) []byte {
	jsonParser := GenerateNewJsonParser()
	jsonParser.SetValue("sendProgress", "MsgType")
	jsonParser.SetValue(identification, "Identification")
	// jsonParser.SetValue(srcFilePath, "SrcFilePath")
	jsonParser.SetValue(int(100*float64(successSendNum)/float64(totalNum)), "Percentage")
	return jsonParser.GenerateJsonBytes()
}

// 反馈给前端的组合进度
func GenerateRestoreProgressJsonBytes(dstFilePath, senderName, receiverName string, fileDataLength, identification, successReceiveNum, totalNum int) []byte {
	jsonParser := GenerateNewJsonParser()
	jsonParser.SetValue("receiveProgress", "MsgType")
	jsonParser.SetValue(fileDataLength, "FileDataLength")
	_, fileName := filepath.Split(dstFilePath)
	// relativePath := filepath.Join("./还原结果", strconv.Itoa(identification), fileName)
	// jsonParser.SetValue(relativePath, "DstFilePath")
	jsonParser.SetValue(senderName, "SenderName")
	jsonParser.SetValue(receiverName, "ReceiverName")
	jsonParser.SetValue(fileName, "FileName")
	jsonParser.SetValue(identification, "Identification")
	jsonParser.SetValue(int(100*float64(successReceiveNum)/float64(totalNum)), "Percentage")
	return jsonParser.GenerateJsonBytes()
}

// // *控制中心*生成发送阶段2的json文件,返回生成的json的bytes
// func GenerateSendStage2JsonBytes(jsonParser_old *JsonParser, readyToSend bool, receiverPublicKey string, IFSSInfoListJsonBytes []byte) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	divideMethod := int(jsonParser_old.ReadJsonValue("/DivideMethod").(float64))
// 	groupNum := int(jsonParser_old.ReadJsonValue("/GroupNum").(float64))
// 	senderName := jsonParser_old.ReadJsonValue("/SenderName").(string)
// 	receiverName := jsonParser_old.ReadJsonValue("/ReceiverName").(string)
// 	fileName := jsonParser_old.ReadJsonValue("/FileName").(string)
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	fileDataLength := int32(jsonParser_old.ReadJsonValue("/FileDataLength").(float64))
// 	timer := int32(jsonParser_old.ReadJsonValue("/Timer").(float64))
// 	IFSSInfoList, _ := ReadJsonBytes(IFSSInfoListJsonBytes)
// 	jsonParser.SetValue("send_stage_2", "MsgType")
// 	jsonParser.SetValue(readyToSend, "ReadyToSend")
// 	jsonParser.SetValue(divideMethod, "DivideMethod")
// 	jsonParser.SetValue(groupNum, "GroupNum")
// 	jsonParser.SetValue(receiverPublicKey, "ReceiverPublicKey")
// 	jsonParser.SetValue(senderName, "SenderName")
// 	jsonParser.SetValue(receiverName, "ReceiverName")
// 	jsonParser.SetValue(fileName, "FileName")
// 	jsonParser.SetValue(identification, "Identification")
// 	jsonParser.SetValue(fileDataLength, "FileDataLength")
// 	jsonParser.SetValue(timer, "Timer")
// 	jsonParser.Merge(IFSSInfoList)
// 	totalNum := divideMethod + groupNum
// 	maxNumInAGroup := int(math.Ceil(float64(totalNum) / float64(groupNum)))
// 	groupInfoList := GenerateNewJsonParser()
// 	jsonParser.SetArray("GroupInfoList")
// 	for i := 0; i < groupNum; i++ {
// 		specFileInfoList := GenerateNewJsonParser()
// 		groupInfoList.SetArray("SpecFileInfoList")
// 		numInGroup := maxNumInAGroup
// 		if i == groupNum-1 {
// 			numInGroup = totalNum - i*maxNumInAGroup
// 		}
// 		for j := 0; j < numInGroup; j++ {
// 			specFileInfo := GenerateNewJsonParser()
// 			// 随机为数据交换文件分配一个IFSS以供上传
// 			randomIFSSInfo := IFSSInfoList.GetRandomChildren("IFSSInfoList")
// 			specFileInfo.SetValue(randomIFSSInfo.ReadJsonValue("/IFSSName").(string), "IFSSName")
// 			// 随机为数据交换文件分配一个9位的随机字符串文件名
// 			var specFileName string
// 			var seed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
// 			b := bytes.NewBufferString(seed)
// 			for i := 0; i < 9; i++ {
// 				randomInt, _ := rand.Int(rand.Reader, big.NewInt(int64(b.Len())))
// 				specFileName += string(seed[randomInt.Int64()])
// 			}
// 			specFileInfo.SetValue(specFileName, "SpecFileName")
// 			// 为包含数据交换文件分配paddingSize(paddingSize为原文件大小的1/3以内)
// 			paddingSize, _ := rand.Int(rand.Reader, big.NewInt(int64(fileDataLength/3)))
// 			specFileInfo.SetValue(paddingSize, "PaddingSize")
// 			specFileInfoList.AppendArray(specFileInfo.Parser, "SpecFileInfoList")
// 		}
// 		groupInfoList.AppendArray(specFileInfoList.Parser, "GroupInfoList")
// 	}
// 	jsonParser.Merge(groupInfoList)
// 	return jsonParser.GenerateJsonBytes()
// }

// // *发送方*生成发送阶段3的json文件,返回生成的json的bytes
// func GenerateSendStage3JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("send_stage_3", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *发送方*生成发送阶段4的json文件,返回生成的json的bytes
// func GenerateSendStage4JsonBytes(jsonParser_old *JsonParser, sendResult bool) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("send_stage_4", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	jsonParser.SetValue(sendResult, "SendResult")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *控制中心*生成发送阶段5的json文件,返回生成的json的bytes
// func GenerateSendStage5JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("send_stage_5", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *发送方*生成发送阶段6的json文件,返回生成的json的bytes
// func GenerateSendStage6JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("send_stage_6", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *控制中心*生成接收阶段1的json文件,返回生成的json的bytes
// func GenerateReceiveStage1JsonBytes(jsonParser_old *JsonParser, senderPublicKey string) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	divideMethod := int(jsonParser_old.ReadJsonValue("/DivideMethod").(float64))
// 	groupNum := int(jsonParser_old.ReadJsonValue("/GroupNum").(float64))
// 	senderName := jsonParser_old.ReadJsonValue("/SenderName").(string)
// 	receiverName := jsonParser_old.ReadJsonValue("/ReceiverName").(string)
// 	fileName := jsonParser_old.ReadJsonValue("/FileName").(string)
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	fileDataLength := int32(jsonParser_old.ReadJsonValue("/FileDataLength").(float64))
// 	timer := int32(jsonParser_old.ReadJsonValue("/Timer").(float64))
// 	IFSSInfoList := jsonParser_old.ReadJsonValue("/IFSSInfoList")
// 	jsonParser.SetValue("receive_stage_1", "MsgType")
// 	jsonParser.SetValue(divideMethod, "DivideMethod")
// 	jsonParser.SetValue(groupNum, "GroupNum")
// 	jsonParser.SetValue(senderPublicKey, "SenderPublicKey")
// 	jsonParser.SetValue(senderName, "SenderName")
// 	jsonParser.SetValue(receiverName, "ReceiverName")
// 	jsonParser.SetValue(fileName, "FileName")
// 	jsonParser.SetValue(identification, "Identification")
// 	jsonParser.SetValue(fileDataLength, "FileDataLength")
// 	jsonParser.SetValue(timer, "Timer")
// 	jsonParser.SetValue(IFSSInfoList, "IFSSInfoList")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *接收方*生成接收阶段2的json文件,返回生成的json的bytes
// func GenerateReceiveStage2JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("receive_stage_2", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *控制中心*生成接收阶段3的json文件,返回生成的json的bytes
// func GenerateReceiveStage3JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("receive_stage_3", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *接收方*生成接收阶段4的json文件,返回生成的json的bytes
// func GenerateReceiveStage4JsonBytes(jsonParser_old *JsonParser, receiveResult bool) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("receive_stage_4", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	jsonParser.SetValue(receiveResult, "ReceiveResult")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *控制中心*生成接收阶段5的json文件,返回生成的json的bytes
// func GenerateReceiveStage5JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("receive_stage_5", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *接收方*生成接收阶段6的json文件,返回生成的json的bytes
// func GenerateReceiveStage6JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("receive_stage_6", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *控制中心*生成清理阶段1的json文件,返回生成的json的bytes
// func GenerateCleanStage1JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	IFSSInfoList := jsonParser_old.ReadJsonValue("/IFSSInfoList")
// 	jsonParser.SetValue("clean_stage_1", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	jsonParser.SetValue(IFSSInfoList, "IFSSInfoList")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *发送方*生成清理阶段2的json文件,返回生成的json的bytes
// func GenerateCleanStage2JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("clean_stage_2", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

// // *控制中心*生成清理阶段3的json文件,返回生成的json的bytes
// func GenerateCleanStage3JsonBytes(jsonParser_old *JsonParser) []byte {
// 	jsonParser := GenerateNewJsonParser()
// 	identification := int32(jsonParser_old.ReadJsonValue("/Identification").(float64))
// 	jsonParser.SetValue("clean_stage_3", "MsgType")
// 	jsonParser.SetValue(identification, "Identification")
// 	return jsonParser.GenerateJsonBytes()
// }

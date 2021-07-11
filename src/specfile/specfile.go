// 数据交换文件的构成与结构,分为头部/对称密钥/Nonce/数据分片/签名/padding.
package specfile

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"xindauserbackground/src/crypto/aestools"
	"xindauserbackground/src/crypto/rsatools"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/jsontools"
	"xindauserbackground/src/specfile/fragment"
	"xindauserbackground/src/specfile/header"
	"xindauserbackground/src/specfile/padding"
	redundance "xindauserbackground/src/specfile/redudance"
)

// 每个段的信息
type StructureInfo struct {
	Start  int
	Length int
}

// 加密/未加密的的数据交换文件的结构
type FileStructure struct {
	HeaderStructure       StructureInfo
	SymmetricKeyStructure StructureInfo
	NonceStructure        StructureInfo
	FragmentStructure     StructureInfo
	SignStructure         StructureInfo
}

// 数据交换文件的摘要信息
type FileInfo struct {
	FilePath                 string
	UnencryptedFileStructure FileStructure
	EncryptedFileStructure   FileStructure
	Header                   header.Header
}

// 每个组的数据交换文件的摘要信息
type GroupInfo struct {
	DataFileInfoList   []FileInfo
	RedundanceFileInfo FileInfo
}

// 生成加密和未加密过的数据交换文件的结构
func generateFileStructure(h header.Header) (unencryptedFileStructure FileStructure, encryptedFileStructure FileStructure) {
	// Header
	unencryptedHeaderStart := 0
	unencryptedHeaderLength := header.GetHeaderBytesSize()
	unencryptedHeaderStructure := StructureInfo{unencryptedHeaderStart, unencryptedHeaderLength}
	encryptedHeaderStart := 0
	encryptedHeaderLength := rsatools.GetCiphertextLength(unencryptedHeaderLength)
	encryptedHeaderStructure := StructureInfo{encryptedHeaderStart, encryptedHeaderLength}
	// 对称密钥(明文128位,32字节)
	unencryptedSymmetricKeyStart := unencryptedHeaderLength
	unencryptedSymmetricKeyLength := 256 / 8
	unencryptedSymmetricKeyStructure := StructureInfo{unencryptedSymmetricKeyStart, unencryptedSymmetricKeyLength}
	encryptedSymmetricKeyStart := encryptedHeaderLength
	encryptedSymmetricKeyLength := rsatools.GetCiphertextLength(unencryptedSymmetricKeyLength)
	encryptedSymmetricKeyStructure := StructureInfo{encryptedSymmetricKeyStart, encryptedSymmetricKeyLength}
	// Nonce(明文12字节)
	unencryptedNonceStart := unencryptedSymmetricKeyStart + unencryptedSymmetricKeyLength
	unencryptedNonceLength := 12
	unencryptedNonceStructure := StructureInfo{unencryptedNonceStart, unencryptedNonceLength}
	encryptedNonceStart := encryptedSymmetricKeyStart + encryptedSymmetricKeyLength
	encryptedNonceLength := rsatools.GetCiphertextLength(unencryptedNonceLength)
	encryptedNonceStructure := StructureInfo{encryptedNonceStart, encryptedNonceLength}
	// Fragment
	unencryptedFragmentStart := unencryptedNonceStart + unencryptedNonceLength
	unencryptedFragmentLength := int(h.GetFileDataLength())
	unencryptedFragmentStructure := StructureInfo{unencryptedFragmentStart, unencryptedFragmentLength}
	encryptedFragmentStart := encryptedNonceStart + encryptedNonceLength
	encryptedFragmentLength := aestools.GetCiphertextLength(unencryptedFragmentLength)
	encryptedFragmentStructure := StructureInfo{encryptedFragmentStart, encryptedFragmentLength}
	// 签名(明文128字节)
	unencryptedSignStart := unencryptedFragmentStart + unencryptedFragmentLength
	unencryptedSignLength := 128
	unencryptedSignStructure := StructureInfo{unencryptedSignStart, unencryptedSignLength}
	encryptedSignStart := encryptedFragmentStart + encryptedFragmentLength
	encryptedSignLength := rsatools.GetCiphertextLength(unencryptedSignLength)
	encryptedSignStructure := StructureInfo{encryptedSignStart, encryptedSignLength}
	// 生成未加密数据交换文件&加密数据交换文件的FileStructure
	unencryptedFileStructure = FileStructure{unencryptedHeaderStructure, unencryptedSymmetricKeyStructure, unencryptedNonceStructure, unencryptedFragmentStructure, unencryptedSignStructure}
	encryptedFileStructure = FileStructure{encryptedHeaderStructure, encryptedSymmetricKeyStructure, encryptedNonceStructure, encryptedFragmentStructure, encryptedSignStructure}
	return
}

// 多个[]byte数组合并成一个[]byte
func bytesCombine(pBytes ...[]byte) []byte {
	len := len(pBytes)
	var buffer bytes.Buffer
	for i := 0; i < len; i++ {
		buffer.Write(pBytes[i])
	}
	return buffer.Bytes()
}

// 根据分片group生成数据交换文件,并写入指定文件夹
func generateSpecFileFolder(fragmentGroup [][][]byte, senderPrivateKeyFilePath, receiverPublicKeyString string, jsonParser *jsontools.JsonParser, saveDir string) (string, error) {
	var err error
	// 获得发送方私钥字符串
	senderPrivateKey, err := rsatools.ReadPrivateKeyFile(senderPrivateKeyFilePath)
	if err != nil {
		return "", err
	}
	receiverPublicKey, err := rsatools.StringToPublicKey(receiverPublicKeyString)
	if err != nil {
		return "", err
	}
	divideMethod := int8(jsonParser.ReadJsonValue("/DivideMethod").(float64))
	groupNum := int8(jsonParser.ReadJsonValue("/GroupNum").(float64))
	maxNumInAGroup := fragment.CalculateMaxNumInAGroup(int(divideMethod), int(groupNum))
	senderName := jsonParser.ReadJsonValue("/SenderName").(string)
	receiverName := jsonParser.ReadJsonValue("/ReceiverName").(string)
	_, fileName := filepath.Split(jsonParser.ReadJsonValue("/SrcFilePath").(string))
	identification := int32(jsonParser.ReadJsonValue("/Identification").(float64))
	fileDataLength := int32(jsonParser.ReadJsonValue("/FileDataLength").(float64))
	// 以receiverName作为存储数据交换文件的文件夹
	specFileFolderName := receiverName
	timer := int32(jsonParser.ReadJsonValue("/Timer").(float64))
	for i := 0; i < len(fragmentGroup); i++ {
		fragmentNumInGroup := len(fragmentGroup[i])
		for j := 0; j < fragmentNumInGroup; j++ {
			var groupSN = int8(i)
			var groupContent []int8
			for k := 0; k < fragmentNumInGroup-1; k++ {
				groupContent = append(groupContent, int8(i*maxNumInAGroup+k))
			}
			var isRedundant = bool(j+1 == fragmentNumInGroup) // 如果是组中最后一个分片,那就是冗余分片
			var fragmentSN int8
			if isRedundant {
				fragmentSN = -1
			} else {
				fragmentSN = int8(i*maxNumInAGroup + j)
			}
			headerBytes, err := header.GenerateHeaderBytes(senderName, receiverName, fileName, identification, fileDataLength, timer, divideMethod, groupNum, groupSN, fragmentSN, groupContent)
			if err != nil {
				return "", err
			}
			aesKey, nonce, err := aestools.InitAES()
			padding := padding.GeneratePadding(int(fileDataLength / 3))
			sign, err := rsatools.Sign(bytesCombine(headerBytes, aesKey, nonce, fragmentGroup[i][j]), senderPrivateKey)
			if err != nil {
				return "", err
			}
			encryptedHeaderBytes, err := rsatools.EncryptWithPublicKey(headerBytes, receiverPublicKey)
			if err != nil {
				return "", err
			}
			encryptedAesKey, err := rsatools.EncryptWithPublicKey(aesKey, receiverPublicKey)
			if err != nil {
				return "", err
			}
			encryptedNonce, err := rsatools.EncryptWithPublicKey(nonce, receiverPublicKey)
			if err != nil {
				return "", err
			}
			encryptedFragmentBytes, err := aestools.EncryptWithAES(aesKey, nonce, fragmentGroup[i][j])
			if err != nil {
				return "", err
			}
			encryptedSign, err := rsatools.EncryptWithPublicKey(sign, receiverPublicKey)
			if err != nil {
				return "", err
			}
			specFileBytes := bytesCombine(encryptedHeaderBytes, encryptedAesKey, encryptedNonce, encryptedFragmentBytes, encryptedSign, padding)
			// 随机为数据交换文件分配一个9位的随机字符串文件名
			var specFileName string
			var seed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
			b := bytes.NewBufferString(seed)
			for i := 0; i < 9; i++ {
				randomInt, _ := rand.Int(rand.Reader, big.NewInt(int64(b.Len())))
				specFileName += string(seed[randomInt.Int64()])
			}
			filePath := filepath.Join(saveDir, specFileFolderName, specFileName)
			err = filetools.WriteFile(filePath, specFileBytes, 0755)
			if err != nil {
				return "", err
			}
		}
	}
	return filepath.Join(saveDir, specFileFolderName), err
}

// 从加密过的文件中读取出未加密的fragment
func generateUnencryptedFragmentBytes(fileInfo FileInfo, receiverPrivateKeyFilePath string, senderPublicKeyString string) ([]byte, error) {
	var err error
	receiverPrivateKey, err := rsatools.ReadPrivateKeyFile(receiverPrivateKeyFilePath)
	// 获得公钥
	senderPublicKey, err := rsatools.StringToPublicKey(senderPublicKeyString)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(fileInfo.FilePath)
	defer f.Close()
	if err != nil {
		fmt.Println("无法打开数据交换文件", fileInfo.FilePath)
		return nil, err
	}
	unencryptedHeaderBytes, err := fileInfo.Header.HeaderToBytes()
	if err != nil {
		return nil, err
	}
	encryptedAesKey := make([]byte, fileInfo.EncryptedFileStructure.SymmetricKeyStructure.Length)
	f.ReadAt(encryptedAesKey, int64(fileInfo.EncryptedFileStructure.SymmetricKeyStructure.Start))
	unencryptedAesKey, err := rsatools.DecryptWithPrivateKey(encryptedAesKey, receiverPrivateKey)
	if err != nil {
		return nil, err
	}
	encryptedNonce := make([]byte, fileInfo.EncryptedFileStructure.NonceStructure.Length)
	f.ReadAt(encryptedNonce, int64(fileInfo.EncryptedFileStructure.NonceStructure.Start))
	unencryptedNonce, err := rsatools.DecryptWithPrivateKey(encryptedNonce, receiverPrivateKey)
	if err != nil {
		return nil, err
	}
	encryptedFragmentBytes := make([]byte, fileInfo.EncryptedFileStructure.FragmentStructure.Length)
	f.ReadAt(encryptedFragmentBytes, int64(fileInfo.EncryptedFileStructure.FragmentStructure.Start))
	unencryptedFragmentBytes, err := aestools.DecryptWithAES(unencryptedAesKey, unencryptedNonce, encryptedFragmentBytes)
	if err != nil {
		return nil, err
	}
	encryptedSign := make([]byte, fileInfo.EncryptedFileStructure.SignStructure.Length)
	f.ReadAt(encryptedSign, int64(fileInfo.EncryptedFileStructure.SignStructure.Start))
	unencryptedSign, err := rsatools.DecryptWithPrivateKey(encryptedSign, receiverPrivateKey)
	if err != nil {
		return nil, err
	}
	// 数据验签
	if rsatools.Verify(bytesCombine(unencryptedHeaderBytes, unencryptedAesKey, unencryptedNonce, unencryptedFragmentBytes), unencryptedSign, senderPublicKey) != nil {
		fmt.Println("数据验签无法通过")
		return nil, err
	}
	return unencryptedFragmentBytes, err
}

// 生成FragmentSN对应UnencryptedFragmentBytes的Map
func generateFragmentSN_UnencryptedFragmentBytesMap(groupSN_GroupInfoMap map[int]GroupInfo, fragmentSN_DataFileInfoMap *map[int]FileInfo, receiverPrivateKeyFilePath string, senderPublicKeyString string) (map[int][]byte, error) {
	var err error
	groupSN_UnencryptedFragmentBytesMap := make(map[int][]byte)
	if err != nil {
		return nil, err
	}
	for groupSN := range groupSN_GroupInfoMap {
		groupInfo := groupSN_GroupInfoMap[groupSN]
		acturalGroupTotal := len(groupInfo.DataFileInfoList)
		if acturalGroupTotal == 0 {
			fmt.Println("无法获取足够的分片分组信息")
			err = fmt.Errorf("无法获取足够的分片分组信息")
			return nil, err
		}
		expectedGroupTotal := len(groupInfo.DataFileInfoList[0].Header.GetGroupContent())
		if expectedGroupTotal > acturalGroupTotal+1 { // 丢失了多个数据分片
			fmt.Println("无法还原,组内太多分片丢失")
			err = fmt.Errorf("无法还原,组内太多分片丢失")
			return nil, err
		} else if expectedGroupTotal == acturalGroupTotal+1 { // 组里面只丢失了一个数据分片
			if (groupInfo.RedundanceFileInfo == FileInfo{}) {
				fmt.Println("无法还原,组内数据分片丢失且冗余分片丢失")
				err = fmt.Errorf("无法还原,组内数据分片丢失且冗余分片丢失")
				return nil, err
			} else if (groupInfo.RedundanceFileInfo != FileInfo{}) { // 冗余分片还在
				// 找到是哪个分片丢了
				var lostDataFragmentSN int = -1
				expectedFragmentSNList := groupInfo.RedundanceFileInfo.Header.GetGroupContent()
				var acturalFragmentSNList []int8
				for _, dataFileInfo := range groupInfo.DataFileInfoList {
					acturalFragmentSNList = append(acturalFragmentSNList, dataFileInfo.Header.GetFragmentSN())
				}
				for _, expectedFragmentSN := range expectedFragmentSNList {
					flag := false
					for _, acturalFragmentSN := range acturalFragmentSNList {
						if expectedFragmentSN == acturalFragmentSN {
							flag = true
						}
					}
					if flag == false {
						lostDataFragmentSN = int(expectedFragmentSN)
					}
				}
				var restoreGroup [][]byte
				for _, fileInfo := range append(groupInfo.DataFileInfoList, groupInfo.RedundanceFileInfo) {
					unencryptedFragmentBytes, err := generateUnencryptedFragmentBytes(fileInfo, receiverPrivateKeyFilePath, senderPublicKeyString)
					if err != nil {
						return nil, err
					}
					restoreGroup = append(restoreGroup, unencryptedFragmentBytes)
					fragmentSN := int(fileInfo.Header.GetFragmentSN())
					var isRedundant = bool(fragmentSN == -1)
					if isRedundant == false {
						groupSN_UnencryptedFragmentBytesMap[fragmentSN] = unencryptedFragmentBytes
					}
				}
				lostDataFragmentBytes := redundance.RestoreLostFragment(restoreGroup[:len(restoreGroup)-2], restoreGroup[len(restoreGroup)-1])
				groupSN_UnencryptedFragmentBytesMap[lostDataFragmentSN] = lostDataFragmentBytes
			}
		} else if expectedGroupTotal == acturalGroupTotal { // 数据分片已经收齐
			for _, fileInfo := range groupInfo.DataFileInfoList {
				unencryptedFragmentBytes, err := generateUnencryptedFragmentBytes(fileInfo, receiverPrivateKeyFilePath, senderPublicKeyString)
				if err != nil {
					return nil, err
				}
				fragmentSN := int(fileInfo.Header.GetFragmentSN())
				groupSN_UnencryptedFragmentBytesMap[fragmentSN] = unencryptedFragmentBytes
			}
		}
	}
	return groupSN_UnencryptedFragmentBytesMap, err
}

// 根据当前待还原文件夹中的数据交换文件列表还原出来文件,并存在fileSavePath里面
func restoreFromFilePathList(fileSaveDir string, filePathList []string, receiverPrivateKeyFilePath string, userListParser *jsontools.JsonParser, restoreProgressChannel chan []byte) error {
	var err error
	receiverPrivateKey, err := rsatools.ReadPrivateKeyFile(receiverPrivateKeyFilePath)
	if err != nil {
		return err
	}
	groupSN_GroupInfoMap := make(map[int]GroupInfo)
	fragmentSN_DataFileInfoMap := make(map[int]FileInfo)
	// 获得发送方公钥字符串
	var senderPublicKeyString string
	for _, filePath := range filePathList {
		f, err := os.Open(filePath)
		if err != nil {
			fmt.Println("无法打开数据交换文件", filePath)
			return err
		}
		// 读取头部,并加入map中
		encryptedHeaderBytes := make([]byte, rsatools.GetCiphertextLength(header.GetHeaderBytesSize()))
		f.ReadAt(encryptedHeaderBytes, 0) // 将头部读取到headerBytes里面
		f.Close()
		unencryptedHeaderBytes, err := rsatools.DecryptWithPrivateKey(encryptedHeaderBytes, receiverPrivateKey)
		if err != nil {
			return err
		}
		header, err := header.BytesToHeader(unencryptedHeaderBytes)
		if err != nil {
			return err
		}
		senderName := header.GetSenderName()
		if senderPublicKeyString == "" {
			for _, children := range userListParser.GetAllChildren("UserList") {
				if children.ReadJsonValue("/Name").(string) == senderName {
					senderPublicKeyString = children.ReadJsonValue("/PublicKey").(string)
					break
				}
			}
		}
		fragmentSN := int(header.GetFragmentSN())
		groupSN := int(header.GetGroupSN())
		unencryptedFileStructure, encryptedFileStructure := generateFileStructure(header)
		fileInfo := FileInfo{filePath, unencryptedFileStructure, encryptedFileStructure, header}
		if fragmentSN != -1 { // 是数据分片的话
			groupSN_GroupInfoMap[groupSN] = GroupInfo{append(groupSN_GroupInfoMap[groupSN].DataFileInfoList, fileInfo), groupSN_GroupInfoMap[groupSN].RedundanceFileInfo}
			fragmentSN_DataFileInfoMap[fragmentSN] = fileInfo
		} else { // 是冗余分片的话
			groupSN_GroupInfoMap[groupSN] = GroupInfo{groupSN_GroupInfoMap[groupSN].DataFileInfoList, fileInfo}
		}
	}
	if groupSN_GroupInfoMap == nil || groupSN_GroupInfoMap[0].DataFileInfoList == nil {
		fmt.Println("无法得到任何分片的头部信息")
		err = fmt.Errorf("无法得到任何分片的头部信息")
		return err
	}
	firstDataFileHeader := groupSN_GroupInfoMap[0].DataFileInfoList[0].Header
	senderName := firstDataFileHeader.GetSenderName()
	receiverName := firstDataFileHeader.GetReceiverName()
	identification := int(firstDataFileHeader.GetIdentification())
	fileName := firstDataFileHeader.GetFileName()
	fileDataLength := int(firstDataFileHeader.GetFileDataLength())
	fileSavePath := filepath.Join(fileSaveDir, strconv.Itoa(identification), fileName)
	dstAbsFilePath, _ := filepath.Abs(fileSavePath)
	divideMethod := int(firstDataFileHeader.GetDivideMethod())
	groupNum := int(firstDataFileHeader.GetGroupNum())
	successReceiveNum := len(filePathList)
	// 告知前端当前组装进度
	if (successReceiveNum < divideMethod+groupNum) {
		restoreProgressJsonBytes := jsontools.GenerateRestoreProgressJsonBytes(dstAbsFilePath, senderName, receiverName, fileDataLength, identification, successReceiveNum, divideMethod+groupNum)
		restoreProgressChannel <- restoreProgressJsonBytes
	}
	// 对丢失的数据分片进行还原,并读取所有的数据分片
	fragmentSN_UnencryptedFragmentBytesMap, err := generateFragmentSN_UnencryptedFragmentBytesMap(groupSN_GroupInfoMap, &fragmentSN_DataFileInfoMap, receiverPrivateKeyFilePath, senderPublicKeyString)
	if err != nil {
		return err
	}
	var fragmentSNList []int
	for fragmentSN := range fragmentSN_UnencryptedFragmentBytesMap {
		fragmentSNList = append(fragmentSNList, fragmentSN)
	}
	// 判断分片数量够不够header里面的DivideMethod的数量,不是的话没法还原
	if len(fragmentSNList) != divideMethod {
		err = fmt.Errorf("收到的分片数量不够")
		fmt.Println("无法根据目前收到的分片还原出原文件", err)
		return err
	}
	// 给fragmentSN排序，从小到大
	sort.Sort(sort.IntSlice(fragmentSNList))
	// 生成按fragmentSN排序好的FragmentBytesList
	var sortedFragmentBytesList [][]byte
	for _, fragmentSN := range fragmentSNList {
		sortedFragmentBytesList = append(sortedFragmentBytesList, fragmentSN_UnencryptedFragmentBytesMap[fragmentSN])
	}
	// 还原出来的最终文件的存储位置即为fileSavePath
	err = fragment.RestoreByFragmentList(fileSavePath, sortedFragmentBytesList)
	if err != nil {
		return err
	}
	if filetools.IsPathExists(fileSavePath) {
		// 当前还原进度为100%
		fmt.Println("已经还原出文件", fileSavePath)
		restoreProgressJsonBytes := jsontools.GenerateRestoreProgressJsonBytes(dstAbsFilePath, senderName, receiverName, fileDataLength, identification, 100, 100)
		restoreProgressChannel <- restoreProgressJsonBytes
	}
	return err
}

// 对要传输的文件,生成数据交换文件,并写入文件夹
func GenerateSpecFileFolder(userListJsonPath, senderPrivateKeyFilePath string, sendStrategyBytes []byte, saveDir string) (string, error) {
	sendStrategyJsonParser, err := jsontools.ReadJsonBytes(sendStrategyBytes)
	if err != nil {
		return "", err
	}
	divideMethod := fragment.DivideMethod(int(sendStrategyJsonParser.ReadJsonValue("/DivideMethod").(float64)))
	srcFilePath := sendStrategyJsonParser.ReadJsonValue("/SrcFilePath").(string)
	dataFragmentList, err := fragment.GenerateDataFragmentList(srcFilePath, divideMethod)
	if err != nil {
		return "", err
	}
	groupNum := int(sendStrategyJsonParser.ReadJsonValue("/GroupNum").(float64))
	fragmentGroup := fragment.ListToGroup(dataFragmentList, groupNum)
	// 为每一组生成冗余分片
	for i := 0; i < len(fragmentGroup); i++ {
		err = redundance.GenerateRedundanceFragment(&(fragmentGroup[i]))
		if err != nil {
			return "", err
		}
	}
	receiverName := sendStrategyJsonParser.ReadJsonValue("/ReceiverName").(string)
	userListParser, err := jsontools.ReadJsonFile(userListJsonPath)
	if err != nil {
		return "", err
	}
	// 获得接收方公钥字符串
	var receiverPublicKeyString string
	for _, children := range userListParser.GetAllChildren("UserList") {
		name := children.ReadJsonValue("/Name").(string)
		if name == receiverName {
			receiverPublicKeyString = children.ReadJsonValue("/PublicKey").(string)
			break
		}
	}
	// 为所有分片添加签名/对称密钥/头部/无意义填充,使之生成数据交换文件,并写入文件夹
	sendDir, err := generateSpecFileFolder(fragmentGroup, senderPrivateKeyFilePath, receiverPublicKeyString, sendStrategyJsonParser, saveDir)
	return sendDir, err
}

// 从数据交换文件的文件夹中恢复出要传输的文件,并将文件存储在fileSaveDir中
func RestoreFromSpecFileFolder(fileSaveDir, receiverPrivateKeyFilePath, userListJsonPath, specFileFoldeDir string, task_RecordMap *sync.Map, restoreProgressChannel chan []byte) error {
	var err error
	_, identification := filepath.Split(specFileFoldeDir)
	filePathList, _, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(specFileFoldeDir)
	if err != nil {
		return err
	}
	userListParser, err := jsontools.ReadJsonFile(userListJsonPath)
	if err != nil {
		return err
	}
	// sortedFragmentBytesList, fileSavePath, err := restoreFromFilePathList(fileSaveDir, filePathList, receiverPrivateKeyFilePath, userListParser, restoreProgressChannel)
	err = restoreFromFilePathList(fileSaveDir, filePathList, receiverPrivateKeyFilePath, userListParser, restoreProgressChannel)
	if err != nil {
		return err
	}
	// // 还原出来的最终文件的存储位置即为fileSavePath
	// err = fragment.RestoreByFragmentList(fileSavePath, sortedFragmentBytesList)
	// if err != nil {
	// 	return err
	// }
	task_RecordMap.Store(identification, true)
	err = filetools.RmDir(specFileFoldeDir)
	// // 当前还原进度为100%
	// dstAbsFilePath, _ := filepath.Abs(fileSavePath)
	// fileInfo, _ := os.Stat(dstAbsFilePath)
	// identificationNum, _ := strconv.Atoi(identification)
	// restoreProgressJsonBytes := jsontools.GenerateRestoreProgressJsonBytes(dstAbsFilePath, int(fileInfo.Size()), identificationNum, 100, 100)
	// restoreProgressChannel <- restoreProgressJsonBytes
	return err
}

// 根据identification,将从IFSS收到的文件分到"待还原"文件夹的不同文件夹中
func DivideToIdentificationList(specFileFolderDir, receiverPrivateKeyFilePath string, restoreFolderDir string, task_RecordMap *sync.Map) error {
	var err error
	receiverPrivateKey, err := rsatools.ReadPrivateKeyFile(receiverPrivateKeyFilePath)
	if err != nil {
		return err
	}
	filePathList, _, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(specFileFolderDir)
	for _, filePath := range filePathList {
		f, err := os.Open(filePath)
		if err != nil {
			fmt.Println("无法打开数据交换文件", filePath)
			return err
		}
		// 读取头部,并加入map中
		encryptedHeaderBytes := make([]byte, rsatools.GetCiphertextLength(header.GetHeaderBytesSize()))
		f.ReadAt(encryptedHeaderBytes, 0) // 将头部读取到headerBytes里面
		f.Close()
		unencryptedHeaderBytes, err := rsatools.DecryptWithPrivateKey(encryptedHeaderBytes, receiverPrivateKey)
		if err != nil {
			return err
		}
		header, err := header.BytesToHeader(unencryptedHeaderBytes)
		if err != nil {
			return err
		}
		identification := strconv.Itoa(int(header.GetIdentification()))
		isSuccessRestore, isExist := task_RecordMap.Load(identification) // 返回值是value, key是否存在
		// 如果检查到这个数据交换文件是之前已经成功还原了的文件的分片,就删掉它
		if isExist {
			if isSuccessRestore.(bool){
				filetools.RmFile(filePath)
				continue
			}
		}
		if !isExist {
			_, fileName := filepath.Split(filePath)
			newDir := filepath.Join(restoreFolderDir, identification, fileName)
			err = filetools.Mkdir(filepath.Join(restoreFolderDir, identification))
			if err != nil {
				return err
			}
			filetools.Rename(filePath, newDir)
		}		
	}
	return err
}

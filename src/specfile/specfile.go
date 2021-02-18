package specfile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"xindauserbackground/src/crypto/aestools"
	"xindauserbackground/src/crypto/rsatools"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/specfile/fragment"
	"xindauserbackground/src/specfile/header"
	"xindauserbackground/src/specfile/padding"
	"xindauserbackground/src/specfile/redudance"
)

// 每个段的信息
type StructureInfo struct {
	Start  int
	length int
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
	unencryptedFragmentLength := int(h.GetFragmentDataLength())
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
func generateSpecFileFolder(fragmentGroup [][][]byte, zipFilePath string, receiverPublicKeyFilePath string, senderPrivateKeyFilePath string, configFilePath string, folderDir string) error {
	var err error
	receiverPublicKey, _ := rsatools.ReadPublicKeyFile(receiverPublicKeyFilePath)
	senderPrivateKey, _ := rsatools.ReadPrivateKeyFile(senderPrivateKeyFilePath)
	for i := 0; i < len(fragmentGroup); i++ {
		groupNum := len(fragmentGroup[i])
		for j := 0; j < groupNum; j++ {
			_, fileName := filepath.Split(zipFilePath)
			var groupSN = int8(i)
			var groupContent []int8
			for k := 0; k < groupNum-1; k++ {
				groupContent = append(groupContent, int8(i*len(fragmentGroup[0])+k))
			}
			var fragmentDataLength = int32(len(fragmentGroup[i][j]))
			var isRedundant = bool(j+1 == groupNum) // 如果是组中最后一个分片,那就是冗余分片
			var fragmentSN int8
			if isRedundant {
				fragmentSN = -1
			} else {
				fragmentSN = int8(i*len(fragmentGroup[0]) + j)
			}
			headerBytes, err := header.GenerateHeaderBytes("lemon", "cherry", fileName, 999, 999, fragmentDataLength, 999, groupSN, fragmentSN, groupContent) // 999的都是预留的字段
			if err != nil {
				return err
			}
			aesKey, nonce, err := aestools.InitAES()
			padding := padding.GeneratePadding(len(fragmentGroup[i][j])/2, len(fragmentGroup[i][j])*2)
			sign, err := rsatools.Sign(bytesCombine(headerBytes, aesKey, nonce, fragmentGroup[i][j]), senderPrivateKey)
			if err != nil {
				return err
			}
			encryptedHeaderBytes, err := rsatools.EncryptWithPublicKey(headerBytes, receiverPublicKey)
			if err != nil {
				return err
			}
			encryptedAesKey, err := rsatools.EncryptWithPublicKey(aesKey, receiverPublicKey)
			if err != nil {
				return err
			}
			encryptedNonce, err := rsatools.EncryptWithPublicKey(nonce, receiverPublicKey)
			if err != nil {
				return err
			}
			encryptedFragmentBytes, err := aestools.EncryptWithAES(aesKey, nonce, fragmentGroup[i][j])
			if err != nil {
				return err
			}
			encryptedSign, err := rsatools.EncryptWithPublicKey(sign, receiverPublicKey)
			if err != nil {
				return err
			}
			specFileBytes := bytesCombine(encryptedHeaderBytes, encryptedAesKey, encryptedNonce, encryptedFragmentBytes, encryptedSign, padding)
			// 把数据交换文件写入文件夹
			filePath := filepath.Join(folderDir, strconv.Itoa(i*len(fragmentGroup[0])+j))
			err = filetools.WriteFile(filePath, specFileBytes, 0777)
			if err != nil {
				return err
			}
		}
	}
	return err
}

// 从加密过的文件中读取出未加密的fragment
func generateUnencryptedFragmentBytes(fileInfo FileInfo, senderPublicKeyFilePath string, receiverPrivateKeyFilePath string) ([]byte, error) {
	var err error
	receiverPrivateKey, err := rsatools.ReadPrivateKeyFile(receiverPrivateKeyFilePath)
	senderPublicKey, err := rsatools.ReadPublicKeyFile(senderPublicKeyFilePath)
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
	encryptedAesKey := make([]byte, fileInfo.EncryptedFileStructure.SymmetricKeyStructure.length)
	f.ReadAt(encryptedAesKey, int64(fileInfo.EncryptedFileStructure.SymmetricKeyStructure.Start))
	unencryptedAesKey, err := rsatools.DecryptWithPrivateKey(encryptedAesKey, receiverPrivateKey)
	if err != nil {
		return nil, err
	}
	encryptedNonce := make([]byte, fileInfo.EncryptedFileStructure.NonceStructure.length)
	f.ReadAt(encryptedNonce, int64(fileInfo.EncryptedFileStructure.NonceStructure.Start))
	unencryptedNonce, err := rsatools.DecryptWithPrivateKey(encryptedNonce, receiverPrivateKey)
	if err != nil {
		return nil, err
	}
	encryptedFragmentBytes := make([]byte, fileInfo.EncryptedFileStructure.FragmentStructure.length)
	f.ReadAt(encryptedFragmentBytes, int64(fileInfo.EncryptedFileStructure.FragmentStructure.Start))
	unencryptedFragmentBytes, err := aestools.DecryptWithAES(unencryptedAesKey, unencryptedNonce, encryptedFragmentBytes)
	if err != nil {
		return nil, err
	}
	encryptedSign := make([]byte, fileInfo.EncryptedFileStructure.SignStructure.length)
	f.ReadAt(encryptedSign, int64(fileInfo.EncryptedFileStructure.SignStructure.Start))
	unencryptedSign, err := rsatools.DecryptWithPrivateKey(encryptedSign, receiverPrivateKey)
	if err != nil {
		return nil, err
	}
	// 数据验签
	if rsatools.Verify(bytesCombine(unencryptedHeaderBytes, unencryptedAesKey, unencryptedNonce, unencryptedFragmentBytes), unencryptedSign, senderPublicKey) != nil {
		fmt.Println("数据验签无法通过")
		err = fmt.Errorf("数据验签无法通过")
		return nil, err
	}
	return unencryptedFragmentBytes, err
}

// 生成FragmentSN对应UnencryptedFragmentBytes的Map
func generateFragmentSN_UnencryptedFragmentBytesMap(groupSN_GroupInfoMap map[int]GroupInfo, fragmentSN_DataFileInfoMap *map[int]FileInfo, senderPublicKeyFilePath string, receiverPrivateKeyFilePath string) (map[int][]byte, error) {
	var err error
	groupSN_UnencryptedFragmentBytesMap := make(map[int][]byte)
	if err != nil {
		return nil, err
	}
	for groupSN := range groupSN_GroupInfoMap {
		groupInfo := groupSN_GroupInfoMap[groupSN]
		acturalGroupTotal := len(groupInfo.DataFileInfoList)
		expectedGroupTotal := len(groupInfo.DataFileInfoList[0].Header.GetGroupContent())
		if expectedGroupTotal > acturalGroupTotal+1 { // 丢失了多个数据分片
			fmt.Println("丢失了太多分片,无法还原")
			err = fmt.Errorf("丢失了太多分片,无法还原")
			return nil, err
		} else if expectedGroupTotal == acturalGroupTotal+1 { // 组里面只丢失了一个数据分片
			if (groupInfo.RedundanceFileInfo == FileInfo{}) {
				fmt.Println("丢失了冗余分片,无法还原")
				err = fmt.Errorf("丢失了冗余分片,无法还原")
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
					unencryptedFragmentBytes, err := generateUnencryptedFragmentBytes(fileInfo, senderPublicKeyFilePath, receiverPrivateKeyFilePath)
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
				unencryptedFragmentBytes, err := generateUnencryptedFragmentBytes(fileInfo, senderPublicKeyFilePath, receiverPrivateKeyFilePath)
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

// 生成按fragmentSN排序好的fragment的列表,以及要生成的文件的存储位置
func generateSortedFragmentBytesList(fileSaveDir string, filePathList []string, senderPublicKeyFilePath string, receiverPrivateKeyFilePath string) ([][]byte, string, error) {
	var err error
	receiverPrivateKey, err := rsatools.ReadPrivateKeyFile(receiverPrivateKeyFilePath)
	if err != nil {
		return nil, "", err
	}
	groupSN_GroupInfoMap := make(map[int]GroupInfo)
	fragmentSN_DataFileInfoMap := make(map[int]FileInfo)
	for _, filePath := range filePathList {
		f, err := os.Open(filePath)
		defer f.Close()
		if err != nil {
			fmt.Println("无法打开数据交换文件", filePath)
			return nil, "", err
		}
		// 读取头部,并加入map中
		encryptedHeaderBytes := make([]byte, rsatools.GetCiphertextLength(header.GetHeaderBytesSize()))
		f.ReadAt(encryptedHeaderBytes, 0) // 将头部读取到headerBytes里面
		unencryptedHeaderBytes, err := rsatools.DecryptWithPrivateKey(encryptedHeaderBytes, receiverPrivateKey)
		if err != nil {
			return nil, "", err
		}
		header, err := header.ReadHeaderFromSpecFileBytes(unencryptedHeaderBytes)
		if err != nil {
			return nil, "", err
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
	// 对丢失的数据分片进行还原,并读取所有的数据分片
	fragmentSN_UnencryptedFragmentBytesMap, err := generateFragmentSN_UnencryptedFragmentBytesMap(groupSN_GroupInfoMap, &fragmentSN_DataFileInfoMap, senderPublicKeyFilePath, receiverPrivateKeyFilePath)
	if err != nil {
		return nil, "", err
	}
	// 获得原始传输文件的文件名,并生成文件存储路径
	fileName := groupSN_GroupInfoMap[0].DataFileInfoList[0].Header.GetFileName()
	fileSavePath := filepath.Join(fileSaveDir, fileName)
	// 给fragmentSN排序，从小到大
	var fragmentSNList []int
	for fragmentSN := range fragmentSN_UnencryptedFragmentBytesMap {
		fragmentSNList = append(fragmentSNList, fragmentSN)
	}
	sort.Sort(sort.IntSlice(fragmentSNList))
	// 生成按fragmentSN排序好的FragmentBytesList
	var sortedFragmentBytesList [][]byte
	for _, fragmentSN := range fragmentSNList {
		sortedFragmentBytesList = append(sortedFragmentBytesList, fragmentSN_UnencryptedFragmentBytesMap[fragmentSN])
	}
	return sortedFragmentBytesList, fileSavePath, err
}

// 对要传输的文件,生成数据交换文件,并写入文件夹
func GenerateSpecFileFolder(zipFilePath string, divideMethod, groupNum int, receiverPublicKeyFilePath string, senderPrivateKeyFilePath string, configFilePath string, folderDir string) error {
	dataFragmentList, err := fragment.GenerateDataFragmentList(zipFilePath, fragment.DivideMethod(divideMethod))
	if err != nil {
		return err
	}
	fragmentGroup := fragment.ListToGroup(dataFragmentList, groupNum)
	// 为每一组生成冗余分片
	for i := 0; i < len(fragmentGroup); i++ {
		err = redundance.GenerateRedundanceFragment(&(fragmentGroup[i]))
		if err != nil {
			return err
		}
	}
	// 为所有分片添加签名/对称密钥/头部/无意义填充,使之生成数据交换文件,并写入文件夹
	err = generateSpecFileFolder(fragmentGroup, zipFilePath, receiverPublicKeyFilePath, senderPrivateKeyFilePath, configFilePath, folderDir)
	return err
}

// 从数据交换文件的文件夹中恢复出要传输的文件,并将文件存储在fileSaveDir中
func RestoreFromSpecFileFolder(fileSaveDir string, senderPublicKeyFilePath string, receiverPrivateKeyFilePath string, configFilePath string, specFileFoldeDir string) error {
	var err error
	filePathList, _,  err := filetools.GenerateFilePathNameListFromFolder(specFileFoldeDir)
	if err != nil {
		return err
	}
	sortedFragmentBytesList, fileSavePath, err := generateSortedFragmentBytesList(fileSaveDir, filePathList, senderPublicKeyFilePath, receiverPrivateKeyFilePath)
	if err != nil {
		return err
	}
	err = fragment.RestoreByFragmentList(fileSavePath, sortedFragmentBytesList)
	return err
}

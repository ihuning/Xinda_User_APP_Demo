package specfile

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"../crypto/aestools"
	"../crypto/rsatools"
	"../filetools"
	"./fragment"
	"./header"
	"./padding"
	redundance "./redudance"
)

// 每个段的信息
type StructureInfo struct {
	Start  int
	length int
}

// 加密过的数据交换文件的结构
type FileStructure struct {
	HeaderStructure   StructureInfo
	SymmetricKey      StructureInfo
	Nonce             StructureInfo
	FragmentStructure StructureInfo
	SignStructure     StructureInfo
}

type FileInfo struct {
	FilePath                 string
	UnencryptedFileStructure FileStructure
	EncryptedFileStructure   FileStructure
	Header                   header.Header
}

type GroupInfo struct {
	DataFileInfoList   []FileInfo
	RedundanceFileInfo FileInfo
}

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

func GenerateZipFile(text string, filePathList []string) (string, error) {
	// 将文本和附件打包成一个zip文件,方便以后对它分片
	return "", nil
}

func getSpecFilePathListFromFolder(folderDir string) ([]string, error) {
	var err error
	files, err := ioutil.ReadDir(folderDir) //读取目录下文件
	if err != nil {
		fmt.Println("无法获得数据交换文件的信息")
		return nil, err
	}
	var filePathList []string
	for _, file := range files {
		if file.IsDir() || file.Name()[0] == '.' {
			continue
		}
		filePath := filepath.Join(folderDir, file.Name())
		filePathList = append(filePathList, filePath)
	}
	return filePathList, err
}

func generateFragmentSN_UnencryptedFragmentBytesMap(groupSN_GroupInfoMap map[int]GroupInfo, fragmentSN_DataFileInfoMap *map[int]FileInfo, senderPublicKeyFilePath string, receiverPrivateKeyFilePath string) (map[int][]byte, error) {
	var err error
	groupSN_UnencryptedFragmentBytesMap := make(map[int][]byte)
	receiverPrivateKey, err := rsatools.ReadPrivateKeyFile(receiverPrivateKeyFilePath)
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
					acturalFragmentSNList= append(acturalFragmentSNList, dataFileInfo.Header.GetFragmentSN())
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
					f, err := os.Open(fileInfo.FilePath)
					defer f.Close()
					if err != nil {
						fmt.Println("无法打开数据交换文件", fileInfo.FilePath)
						return nil, err
					}
					encryptedAesKey := make([]byte, fileInfo.EncryptedFileStructure.SymmetricKey.length)
					f.ReadAt(encryptedAesKey, int64(fileInfo.EncryptedFileStructure.SymmetricKey.Start))
					aesKey, err := rsatools.DecryptWithPrivateKey(encryptedAesKey, receiverPrivateKey)
					if err != nil {
						return nil, err
					}
					encryptedNonce := make([]byte, fileInfo.EncryptedFileStructure.Nonce.length)
					f.ReadAt(encryptedNonce, int64(fileInfo.EncryptedFileStructure.Nonce.Start))
					nonce, err := rsatools.DecryptWithPrivateKey(encryptedNonce, receiverPrivateKey)
					if err != nil {
						return nil, err
					}
					encryptedFragmentBytes := make([]byte, fileInfo.EncryptedFileStructure.FragmentStructure.length)
					f.ReadAt(encryptedFragmentBytes, int64(fileInfo.EncryptedFileStructure.FragmentStructure.Start))
					unencryptedFragmentBytes, err := aestools.DecryptWithAES(aesKey, nonce, encryptedFragmentBytes)
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
				f, err := os.Open(fileInfo.FilePath)
				defer f.Close()
				if err != nil {
					fmt.Println("无法打开数据交换文件", fileInfo.FilePath)
					return nil, err
				}
				encryptedAesKey := make([]byte, fileInfo.EncryptedFileStructure.SymmetricKey.length)
				f.ReadAt(encryptedAesKey, int64(fileInfo.EncryptedFileStructure.SymmetricKey.Start))
				aesKey, err := rsatools.DecryptWithPrivateKey(encryptedAesKey, receiverPrivateKey)
				if err != nil {
					return nil, err
				}
				encryptedNonce := make([]byte, fileInfo.EncryptedFileStructure.Nonce.length)
				f.ReadAt(encryptedNonce, int64(fileInfo.EncryptedFileStructure.Nonce.Start))
				nonce, err := rsatools.DecryptWithPrivateKey(encryptedNonce, receiverPrivateKey)
				if err != nil {
					return nil, err
				}
				encryptedFragmentBytes := make([]byte, fileInfo.EncryptedFileStructure.FragmentStructure.length)
				f.ReadAt(encryptedFragmentBytes, int64(fileInfo.EncryptedFileStructure.FragmentStructure.Start))
				unencryptedFragmentBytes, err := aestools.DecryptWithAES(aesKey, nonce, encryptedFragmentBytes)
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

func generateSortedFragmentBytesList(filePathList []string, senderPublicKeyFilePath string, receiverPrivateKeyFilePath string) ([][]byte, error) {
	var err error
	receiverPrivateKey, err := rsatools.ReadPrivateKeyFile(receiverPrivateKeyFilePath)
	if err != nil {
		return nil, err
	}
	groupSN_GroupInfoMap := make(map[int]GroupInfo)
	fragmentSN_DataFileInfoMap := make(map[int]FileInfo)
	for _, filePath := range filePathList {
		f, err := os.Open(filePath)
		defer f.Close()
		if err != nil {
			fmt.Println("无法打开数据交换文件", filePath)
			return nil, err
		}
		// 读取头部,并加入map中
		encryptedHeaderBytes := make([]byte, rsatools.GetCiphertextLength(header.GetHeaderBytesSize()))
		f.ReadAt(encryptedHeaderBytes, 0) // 将头部读取到headerBytes里面
		unencryptedHeaderBytes, err := rsatools.DecryptWithPrivateKey(encryptedHeaderBytes, receiverPrivateKey)
		if err != nil {
			return nil, err
		}
		header, err := header.ReadHeaderFromSpecFileBytes(unencryptedHeaderBytes)
		if err != nil {
			return nil, err
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
		return nil, err
	}
	// 给key排序，从小到大
	var fragmentSNList []int
	for fragmentSN := range fragmentSN_UnencryptedFragmentBytesMap {
		fragmentSNList = append(fragmentSNList, fragmentSN)
	}
	sort.Sort(sort.IntSlice(fragmentSNList))
	var sortedFragmentBytesList [][]byte
	for _, fragmentSN := range fragmentSNList {
		sortedFragmentBytesList = append(sortedFragmentBytesList, fragmentSN_UnencryptedFragmentBytesMap[fragmentSN])
	}
	return sortedFragmentBytesList, err
}

func GenerateSpecFileFolder(zipFilePath string, fragmentNum int, receiverPublicKeyFilePath string, senderPrivateKeyFilePath string, configFilePath string, folderDir string) error {
	dataFragmentList, err := fragment.GenerateDataFragmentList(zipFilePath, fragment.DivideMethod(fragmentNum))
	if err != nil {
		return err
	}
	receiverPublicKey, _ := rsatools.ReadPublicKeyFile(receiverPublicKeyFilePath)
	senderPrivateKey, _ := rsatools.ReadPrivateKeyFile(senderPrivateKeyFilePath)
	fragmentGroup := fragment.ListToGroup(dataFragmentList, 3)
	// 为每一组生成冗余分片
	for i := 0; i < len(fragmentGroup); i++ {
		err = redundance.GenerateRedundanceFragment(&(fragmentGroup[i]))
		if err != nil {
			return err
		}
	}
	// 为所有分片添加签名/对称密钥/头部/无意义填充,使之生成数据交换文件
	for i := 0; i < len(fragmentGroup); i++ {
		groupNum := len(fragmentGroup[i])
		for j := 0; j < groupNum; j++ {
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
			headerBytes, err := header.GenerateHeaderBytes("lemon", "cherry", "1.txt", 999, 999, 999, fragmentDataLength, 999, 999, groupSN, fragmentSN, groupContent)
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
			encryptedSpecFileBytes := bytesCombine(encryptedHeaderBytes, encryptedAesKey, encryptedNonce, encryptedFragmentBytes, encryptedSign, padding)
			// 生成数据交换文件
			filePath := filepath.Join(folderDir, strconv.Itoa(i*len(fragmentGroup[0])+j))
			filetools.WriteFile(filePath, encryptedSpecFileBytes, 0777)
		}
	}
	return err
}

func RestoreFromSpecFileFolder(zipFilePath string, senderPublicKeyFilePath string, receiverPrivateKeyFilePath string, configFilePath string, folderDir string) error {
	var err error
	filePathList, err := getSpecFilePathListFromFolder(folderDir)
	if err != nil {
		return err
	}
	sortedFragmentBytesList, err := generateSortedFragmentBytesList(filePathList, senderPublicKeyFilePath, receiverPrivateKeyFilePath)
	if err != nil {
		return err
	}
	err = fragment.RestoreByFragmentList(zipFilePath, sortedFragmentBytesList)
	return err
}

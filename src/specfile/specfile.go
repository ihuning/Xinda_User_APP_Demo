package specfile

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"io/ioutil"
	"os"
	"strconv"
	"../filetools"
	"./fragment"
	"./header"
	redundance "./redudance"
	"./padding"
	"../crypto/aestools"
	"../crypto/rsatools"
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
	// Sign          []byte
	// SymmetricKey  []byte
	Header header.Header
	// Fragment      []byte
	// padding      []byte
}

type GroupInfo struct {
	DataFileInfoList   []FileInfo
	RedundanceFileInfo FileInfo
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

// 生成签名
func GenerateSign(data []byte, privateKeyFilePath string) ([]byte, error) {
	var err error
	privateKey, err := rsatools.ReadPrivateKeyFile(privateKeyFilePath)
	if err != nil {
		return nil, err
	}
	sign, err := rsatools.Sign(data, privateKey)
	if err != nil {
		return nil, err
	}
	return sign, err
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

func restorelostDataFragment(groupSN_GroupInfoMap map[int]GroupInfo, fragmentSN_DataFileInfoMap *map[int]FileInfo) error {
	var err error
	for groupSN := range groupSN_GroupInfoMap {
		groupInfo := groupSN_GroupInfoMap[groupSN]
		expectedGroupTotal := len(groupInfo.RedundanceFileInfo.Header.GetGroupContent())
		acturalGroupTotal := len(groupInfo.DataFileInfoList)
		if expectedGroupTotal == acturalGroupTotal { // 数据分片已经收齐
			continue
		} else if expectedGroupTotal == acturalGroupTotal+1 { // 组里面只丢失了一个数据分片,用其他数据分片和冗余分片还原丢失数据分片
			var lostDataFragmentSN int8 = -1
			for _, fragmentSN := range groupInfo.RedundanceFileInfo.Header.GetGroupContent() {
				for _, dataFileInfo := range groupInfo.DataFileInfoList {
					if fragmentSN != dataFileInfo.Header.GetFragmentSN() {
						lostDataFragmentSN = fragmentSN
						break
					}
				}
			}
			var restoreGroup [][]byte
			for _, fileInfo := range append(groupInfo.DataFileInfoList, groupInfo.RedundanceFileInfo) {
				f, err := os.Open(fileInfo.FilePath)
				if err != nil {
					fmt.Println("无法打开数据交换文件", fileInfo.FilePath)
					return err
				}
				fragmentBytes := make([]byte, fileInfo.UnencryptedFileStructure.FragmentStructure.length)
				f.ReadAt(fragmentBytes, int64(fileInfo.UnencryptedFileStructure.FragmentStructure.Start)) // 将数据读取到fragmentBytes里面
				f.Close()
				restoreGroup = append(restoreGroup, fragmentBytes)
			}
			lostDataFragmentBytes := redundance.RestoreLostFragment(restoreGroup[:len(restoreGroup)-2], restoreGroup[len(restoreGroup)-1])
			lostDataFragmentHeader := groupInfo.DataFileInfoList[0].Header
			(&lostDataFragmentHeader).SetFragmentSN(lostDataFragmentSN)
			lostDataFragmentHeaderBytes, err := lostDataFragmentHeader.HeaderToBytes()
			dir, _ := filepath.Split(groupInfo.RedundanceFileInfo.FilePath)
			fileName := strconv.Itoa(int(lostDataFragmentSN))
			lostDataFilePath := filepath.Join(dir, fileName)
			if err != nil {
				return err
			}
			lostDataFileBytes := bytesCombine(lostDataFragmentHeaderBytes, lostDataFragmentBytes)
			// 生成数据交换文件
			filetools.WriteFile(lostDataFilePath, lostDataFileBytes, 0777)
			lostDataFileStructure := groupInfo.RedundanceFileInfo.UnencryptedFileStructure
			lostDataFileInfo := FileInfo{lostDataFilePath, lostDataFileStructure, lostDataFileStructure, lostDataFragmentHeader}
			(*fragmentSN_DataFileInfoMap)[int(lostDataFragmentSN)] = lostDataFileInfo
		} else { // 丢失了多个数据分片
			fmt.Println("丢失了太多分片,无法还原")
			err = fmt.Errorf("丢失了太多分片,无法还原")
			return err
		}
	}
	return err
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

func generateSortedDataFileInfoList(filePathList []string) ([]FileInfo, error) {
	var err error
	groupSN_GroupInfoMap := make(map[int]GroupInfo)
	fragmentSN_DataFileInfoMap := make(map[int]FileInfo)
	for _, filePath := range filePathList {
		f, err := os.Open(filePath)
		if err != nil {
			fmt.Println("无法打开数据交换文件", filePath)
			return nil, err
		}
		// 读取头部,并加入map中
		headerBytes := make([]byte, header.GetHeaderBytesSize())
		f.ReadAt(headerBytes, 0) // 将头部读取到headerBytes里面
		f.Close()
		header, err := header.ReadHeaderFromSpecFileBytes(headerBytes)
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
	// 检查每个冗余分组是否收齐了所有的数据分片,并尝试对丢失的数据分片进行还原
	restorelostDataFragment(groupSN_GroupInfoMap, &fragmentSN_DataFileInfoMap)
	// 去除冗余分片
	var fragmentSNList []int
	for fragmentSN := range fragmentSN_DataFileInfoMap {
		fragmentSNList = append(fragmentSNList, fragmentSN)
	}
	// 给key排序，从小到大
	sort.Sort(sort.IntSlice(fragmentSNList))
	var sortedDataFileInfoList []FileInfo
	for _, fragmentSN := range fragmentSNList {
		sortedDataFileInfoList = append(sortedDataFileInfoList, fragmentSN_DataFileInfoMap[fragmentSN])
	}
	return sortedDataFileInfoList, err
}

func GenerateSpecFileFolder(zipFilePath string, fragmentNum int, publicKeyFilePath string, privateKeyFilePath string, configFilePath string, folderDir string) error {
	dataFragmentList, err := fragment.GenerateDataFragmentList(zipFilePath, fragment.DivideMethod(fragmentNum))
	if err != nil {
		return err
	}
	fragmentGroup := fragment.ListToGroup(dataFragmentList, 3)
	// 为每一组生成冗余分片
	for i := 0; i < len(fragmentGroup); i++ {
		err = redundance.GenerateRedundanceFragment(&(fragmentGroup[i]))
		if err != nil {
			return err
		}
	}
	// publicKey, _ := rsatools.ReadPublicKeyFile(publicKeyFilePath)
	privateKey, _ := rsatools.ReadPrivateKeyFile(privateKeyFilePath)
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
			sign, err := rsatools.Sign(bytesCombine(headerBytes, aesKey, nonce, fragmentGroup[i][j], padding), privateKey)
			if err != nil {
				return err
			}
			specFileBytes := bytesCombine(headerBytes, aesKey, nonce, fragmentGroup[i][j], sign, padding)
			// 生成数据交换文件
			filePath := filepath.Join(folderDir, strconv.Itoa(i*len(fragmentGroup[0])+j))
			filetools.WriteFile(filePath, specFileBytes, 0777)
		}
	}
	return err
}

func RestoreFromSpecFileFolder(zipFilePath string, publicKeyFilePath string, privateKeyFilePath string, configFilePath string, folderDir string) error {
	var err error
	filePathList, err := getSpecFilePathListFromFolder(folderDir)
	if err != nil {
		return err
	}
	sortedDataFileInfoList, err := generateSortedDataFileInfoList(filePathList)
	if err != nil {
		return err
	}
	var sortedFragmentList [][]byte
	for _, dataFileInfo := range sortedDataFileInfoList {
		f, err := os.Open(dataFileInfo.FilePath)
		if err != nil {
			fmt.Println("无法打开数据交换文件", dataFileInfo.FilePath)
			return err
		}
		fragmentBytes := make([]byte, dataFileInfo.UnencryptedFileStructure.FragmentStructure.length)
		f.ReadAt(fragmentBytes, int64(dataFileInfo.UnencryptedFileStructure.FragmentStructure.Start)) // 将数据读取到fragmentBytes里面
		f.Close()
		sortedFragmentList = append(sortedFragmentList, fragmentBytes)
	}
	err = fragment.RestoreByFragmentList(zipFilePath, sortedFragmentList)
	return err
}

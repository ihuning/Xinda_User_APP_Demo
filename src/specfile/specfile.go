package specfile

import (
	"bytes"
	"fmt"
	"sort"

	// "io"
	"io/ioutil"
	"os"
	"strconv"

	"../filetools"
	"./fragment"
	"./header"
	redundance "./redudance"

	// "./ziptools"
	"../crypto/rsatools"
	// "../crypto/aestools"
)

// 每个段的信息
type StructureInfo struct {
	Start  int
	length int
}

// 未加密或加密过的数据交换文件的结构
type FileStructure struct {
	HeaderStructure StructureInfo
	DataStructure   StructureInfo
}

// type SpecFile struct {
// 	SignLength         int
// 	SymmetricKeyLength int
// 	HeaderLength       int
// 	DataLength         int
// 	PaddingLength      int
// }

type FileInfo struct {
	FilePath      string
	FileStructure FileStructure
	Header        header.Header
}

type GroupInfo struct {
	DataFileInfoList   []FileInfo
	RedundanceFileInfo FileInfo
}

// 生成UnencryptedFileStructure和EncryptedFileStructure
// func GenerateUnencryptedFileStructure(data []byte) (FileStructure, FileStructure) {
// 	var UnencryptedFileStructure, EncryptedFileStructure FileStructure

// 	return UnencryptedFileStructure, EncryptedFileStructure
// }

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

// 验证签名
func VerifySign(data []byte, sign []byte, publicKeyFilePath string) (bool, error) {
	var err error
	publicKey, err := rsatools.ReadPublicKeyFile(publicKeyFilePath)
	if err != nil {
		return false, err
	}
	result := rsatools.Verify(data, sign, publicKey)
	if result == nil {
		return true, nil
	}
	return false, nil
}

func GenerateZipFile(text string, filePathList []string) (string, error) {
	// 将文本和附件打包成一个zip文件,方便以后对它分片
	return "", nil
}

func GenerateSpecFileFolder(zipFilePath string, publicKeyFilePath string, privateKeyFilePath string, configFilePath string, folderDir string) error {
	dataFragmentList, err := fragment.GenerateDataFragmentList(zipFilePath, fragment.FRAGMNETS_8)
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
	// 为所有分片添加头部,使之生成数据交换文件
	for i := 0; i < len(fragmentGroup); i++ {
		groupNum := len(fragmentGroup[i])
		for j := 0; j < groupNum; j++ {
			var groupSN = int32(i)
			var groupContent []int8
			var fragmentDataLength = int32(len(fragmentGroup[i][j]))
			for k := 0; k < groupNum; k++ {
				groupContent = append(groupContent, int8(i*len(fragmentGroup[0])+k))
			}
			var isRedundant = bool(j+1 == groupNum) // 如果是组中最后一个分片,那就是冗余分片
			var fragmentSN int32
			if isRedundant {
				fragmentSN = -1
			} else {
				fragmentSN = int32(i*len(fragmentGroup[0]) + j)
			}
			headerBytes, err := header.GenerateHeaderBytes("lemon", "cherry", "1.txt", 999, 999, 999, fragmentDataLength, fragmentSN, 999, 999, groupSN, groupContent)
			if err != nil {
				return err
			}
			specFileBytes := bytesCombine(headerBytes, fragmentGroup[i][j])
			// 生成数据交换文件
			filePath := fmt.Sprintf("%s%s%s", folderDir, "/", strconv.Itoa(i*len(fragmentGroup[0])+j))
			filetools.WriteFile(filePath, specFileBytes, 0777)
		}
	}
	return err
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
		filePath := fmt.Sprintf("%s%s%s", folderDir, "/", file.Name())
		filePathList = append(filePathList, filePath)
	}
	return filePathList, err
}

func restoreLostDataFragment(groupSN_GroupInfoMap map[int]GroupInfo, fragmentSN_DataFileInfoMap map[int]FileInfo) error {
	var err error
	for groupSN := range groupSN_GroupInfoMap {
		groupInfo := groupSN_GroupInfoMap[groupSN]
		expectedGroupTotal := len(groupInfo.RedundanceFileInfo.Header.GetGroupContent())
		acturalGroupTotal := len(groupSN_GroupInfoMap[groupSN].DataFileInfoList)
		if expectedGroupTotal == acturalGroupTotal { // 数据分片已经收齐
			continue
		} else if expectedGroupTotal == acturalGroupTotal+1 { // 组里面只丢失了一个数据分片,用其他数据分片和冗余分片还原丢失数据分片

		}
	}
	return err
}

func generateFileStructure(h header.Header) FileStructure {
	headerStart := 0
	headerLength := header.GetHeaderBytesSize()
	headerStructure := StructureInfo{headerStart, headerLength}
	dataStart := headerLength
	dataLength := int(h.GetFragmentDataLength())
	dataStructure := StructureInfo{dataStart, dataLength}
	fileStructure := FileStructure{headerStructure, dataStructure}
	return fileStructure
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
		fileStructure := generateFileStructure(header)
		fileInfo := FileInfo{filePath, fileStructure, header}
		if fragmentSN != -1 { // 是数据分片的话
			groupSN_GroupInfoMap[groupSN] = GroupInfo{append(groupSN_GroupInfoMap[groupSN].DataFileInfoList, fileInfo), groupSN_GroupInfoMap[groupSN].RedundanceFileInfo}
			fragmentSN_DataFileInfoMap[fragmentSN] = fileInfo
		} else { // 是冗余分片的话
			groupSN_GroupInfoMap[groupSN] = GroupInfo{groupSN_GroupInfoMap[groupSN].DataFileInfoList, fileInfo}
		}
	}
	// 检查每个冗余分组是否收齐了所有的数据分片,并尝试对丢失的数据分片进行还原

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
		fragmentBytes := make([]byte, dataFileInfo.FileStructure.DataStructure.length)
		f.ReadAt(fragmentBytes, int64(dataFileInfo.FileStructure.DataStructure.Start)) // 将头部读取到headerBytes里面
		f.Close()
		sortedFragmentList = append(sortedFragmentList, fragmentBytes)
	}
	err = fragment.RestoreByFragmentList(zipFilePath, sortedFragmentList)
	return err
}

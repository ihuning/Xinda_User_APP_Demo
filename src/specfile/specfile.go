package specfile

import (
	"bytes"
	"fmt"
	"sort"

	// "io"
	"io/ioutil"
	"os"
	"strconv"

	"./fragment"
	"./header"
	redundance "./redudance"

	// "./ziptools"
	"../crypto/rsatools"
	// "../crypto/aestools"
)

// type SpecFileInfo struct {
// 	FilePath   string
// 	FragmentSN int
// }

// type SpecFile struct {
// 	SignLength         int
// 	SymmetricKeyLength int
// 	HeaderLength       int
// 	DataLength         int
// 	PaddingLength      int
// }

type SpecFileInfo struct {
	filePath string
	Header   header.Header
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
			var fragmentSN = int32(i*len(fragmentGroup[0]) + j)
			var redundantSN = int32(i)
			var redundantTotal = int32(groupNum)
			var isRedundant = bool(j+1 == groupNum) // 如果是组中最后一个分片,那就是冗余分片
			headerBytes, err := header.GenerateHeaderBytes("lemon", "cherry", "1.txt", 999, 999, 999, 999, fragmentSN, 999, 999, redundantSN, redundantTotal, isRedundant)
			if err != nil {
				return err
			}
			specFileBytes := bytesCombine(headerBytes, fragmentGroup[i][j])
			// 生成数据交换文件
			filePath := fmt.Sprintf("%s%s%s", folderDir, "/", strconv.Itoa(int(fragmentSN)))
			ioutil.WriteFile(filePath, specFileBytes, 0777)
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

func generateSortedDataFilePathList(filePathList []string) ([]string, error) {
	var err error
	fragmentSN_FileInfoMap := make(map[int]SpecFileInfo)
	for _, filePath := range filePathList {
		f, err := os.Open(filePath)
		if err != nil {
			fmt.Println("无法打开数据交换文件", filePath)
			return nil, err
		}
		// 读取头部,并加入map中
		headerBytes := make([]byte, 332)
		f.ReadAt(headerBytes, 0) // 将头部读取到headerBytes里面
		f.Close()
		header, err := header.ReadHeaderFromSpecFileBytes(headerBytes)
		fragmentSN := int(header.GetFragmentSN())
		fragmentSN_FileInfoMap[fragmentSN] = SpecFileInfo{filePath, header}
	}
	var fragmentSNList []int
	fragmentSN_DataFileInfoMap := make(map[int]SpecFileInfo)
	// 去除冗余分片
	for fragmentSN := range fragmentSN_FileInfoMap {
		if fragmentSN_FileInfoMap[fragmentSN].Header.GetIsRedundant() == true {
			continue // 冗余分片不参与后续排序
		}
		fragmentSNList = append(fragmentSNList, fragmentSN)
		fragmentSN_DataFileInfoMap[fragmentSN] = fragmentSN_FileInfoMap[fragmentSN]
	}
	// 给key排序，从小到大
	sort.Sort(sort.IntSlice(fragmentSNList))
	var sortedDataFilePathList []string
	for _, fragmentSN := range fragmentSNList {
		// fileInfo := DataFileInfo{int(dataFilePath_HeaderMap[filePath].GetFragmentSN()), filePath}
		sortedDataFilePathList = append(sortedDataFilePathList, fragmentSN_DataFileInfoMap[fragmentSN].filePath)
	}
	return sortedDataFilePathList, err
}

func RestoreFromSpecFileFolder(zipFilePath string, publicKeyFilePath string, privateKeyFilePath string, configFilePath string, folderDir string) error {
	var err error
	filePathList, err := getSpecFilePathListFromFolder(folderDir)
	if err != nil {
		return err
	}
	sortedDataFilePathList, err := generateSortedDataFilePathList(filePathList)
	if err != nil {
		return err
	}
	var sortedFragmentList [][]byte
	for _, dataFilePath := range sortedDataFilePathList {
		f, err := os.Open(dataFilePath)
		if err != nil {
			fmt.Println("无法打开数据交换文件", dataFilePath)
			return err
		}
		fragmentBytes := make([]byte, 267818)
		f.ReadAt(fragmentBytes, 332) // 将头部读取到headerBytes里面
		f.Close()
		sortedFragmentList = append(sortedFragmentList, fragmentBytes)
	}
	err = fragment.RestoreByFragmentList(zipFilePath, sortedFragmentList)
	return err
}

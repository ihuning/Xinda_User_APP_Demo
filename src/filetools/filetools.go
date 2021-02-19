package filetools

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// 如果文件夹不存在,就生成一个文件夹
func createDirIfNotExist(dir string) error {
	var err error
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Println("无法创建文件夹", dir, "错误为", err)
			return err
		}
	}
	return err
}

// 写文件
func WriteFile(filePath string, data []byte, perm os.FileMode) error {
	var err error
	dir, _ := filepath.Split(filePath)
	err = createDirIfNotExist(dir)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, data, perm)
	if err != nil {
		fmt.Println("无法写入文件", dir, "错误为", err)
		return err
	}
	return err
}

// 读文件
func ReadFile(filePath string) ([]byte, error) {
	return ioutil.ReadFile(filePath)
}

// 移动文件
func Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// 判断路径是否存在
func IsPathExists(path string) bool {
	var err error
	_, err = os.Stat(path)
	if err != nil {
		return false
	}
	return true
}

// 创建一个文件夹,
func Mkdir(folderDir string) error {
	var err error
	isFolderExist := IsPathExists(folderDir)
	if err != nil {
		return err
	}
	if isFolderExist == false {
		err := os.Mkdir(folderDir, 0755)
		if err != nil {
			fmt.Println("无法创建文件夹", folderDir, err)
			return err
		}
	}
	return err
}

// 删除一个文件夹
func Rmdir(folderDir string) error {
	var err error
	err = os.RemoveAll(folderDir)
	if err != nil {
		fmt.Println("无法删除文件夹", folderDir, err)
		return err
	}
	return err
}

// 读取文件夹下的所有数据交换文件(排除文件夹和.开头的隐藏文件),并返回路径列表和文件名列表
func GenerateSpecFilePathNameListFromFolder(folderDir string) ([]string, []string, error) {
	var err error
	fileList, err := ioutil.ReadDir(folderDir) //读取目录下文件
	if err != nil {
		fmt.Println("无法读取文件夹", folderDir, err)
		return nil, nil, err
	}
	var filePathList []string
	var fileNameList []string
	for _, file := range fileList {
		if file.IsDir() || file.Name()[0] == '.' {
			continue
		}
		filePath := filepath.Join(folderDir, file.Name())
		filePathList = append(filePathList, filePath)
		fileNameList = append(fileNameList, file.Name())
	}
	return filePathList, fileNameList, err
}

// 读取文件夹下的所有文件和文件夹,并返回路径列表和文件名列表
func GenerateAllFilePathNameListFromFolder(folderDir string) ([]string, []string, error) {
	var err error
	fileList, err := ioutil.ReadDir(folderDir) //读取目录下文件
	if err != nil {
		fmt.Println("无法读取文件夹", folderDir, err)
		return nil, nil, err
	}
	var filePathList []string
	var fileNameList []string
	for _, file := range fileList {
		filePath := filepath.Join(folderDir, file.Name())
		filePathList = append(filePathList, filePath)
		fileNameList = append(fileNameList, file.Name())
	}
	return filePathList, fileNameList, err
}

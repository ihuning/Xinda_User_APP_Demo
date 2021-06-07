package filetools

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
)

// 如果文件夹不存在,就生成一个文件夹
func createFolderIfNotExist(dir string) error {
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
	err = createFolderIfNotExist(dir)
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
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("无法读取文件", err)
		return nil, err
	}
	return bytes, err
}

// 移动或重命名文件
func Rename(oldPath, newPath string) error {
	dir, _ := filepath.Split(newPath)
	err := Mkdir(dir)
	if err != nil {
		return err
	}
	err = os.Rename(oldPath, newPath)
	if err != nil {
		fmt.Println("无法移动或重命名文件", err)
		return err
	}
	return err
}

// 把列表中的文件移动到新的文件夹
func MoveFilesToNewFolder(filePathList []string, newDir string) error {
	var err error
	err = Mkdir(newDir)
	if err != nil {
		return err
	}
	for _, filePath := range filePathList {
		_, fileName := filepath.Split(filePath)
		err = Rename(filePath, filepath.Join(newDir, fileName))
		if err != nil {
			return err
		}
	}
	return err
}

// 移动一个文件夹中的所有文件到新的文件夹
func MoveAllFilesToNewFolder(oldDir, newDir string) error {
	var err error
	err = Mkdir(newDir)
	if err != nil {
		return err
	}
	filePathList, fileNameList, err := GenerateAllFilePathNameListFromFolder(oldDir)
	if err != nil {
		return err
	}
	for i := 0; i < len(filePathList); i++ {
		err = Rename(filePathList[i], filepath.Join(newDir, fileNameList[i]))
		if err != nil {
			return err
		}
	}
	return err
}

// 复制文件
func Copy(oldPath, newPath string) error {
	dir, _ := filepath.Split(newPath)
	err := Mkdir(dir)
	if err != nil {
		return err
	}
	content, err := ioutil.ReadFile(oldPath)
	if err != nil {
		fmt.Println("无法在复制过程中读取原文件", err)
		return err
	}
	err = ioutil.WriteFile(newPath, content, 0777)
	if err != nil {
		fmt.Println("无法在复制过程中写入新文件", err)
		return err
	}
	return err
}

// 复制一个文件夹中的所有文件到新的文件夹
func CopyAllFilesToNewFolder(oldDir, newDir string) error {
	var err error
	err = Mkdir(newDir)
	if err != nil {
		return err
	}
	filePathList, fileNameList, err := GenerateAllFilePathNameListFromFolder(oldDir)
	if err != nil {
		return err
	}
	for i := 0; i < len(filePathList); i++ {
		err = Copy(filePathList[i], filepath.Join(newDir, fileNameList[i]))
		if err != nil {
			return err
		}
	}
	return err
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
	if isFolderExist == false {
		err := os.Mkdir(folderDir, 0755)
		if err != nil {
			fmt.Println("无法创建文件夹", folderDir, err)
			return err
		}
	}
	return err
}

// 删除一个文件
func RmFile(filePath string) error {
	var err error
	err = os.Remove(filePath)
	if err != nil {
		fmt.Println("无法删除文件", filePath, err)
		return err
	}
	return err
}

// 删除一个文件夹
func RmDir(folderDir string) error {
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

// 将多个路径组成的列表拆分为多个列表
func DivideDirListToGroup(dirList []string, groupNum int) [][]string {
	random := func(max int) int {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
		return int(n.Int64())
	}
	var DirGroup [][]string
	for i := 0; i < groupNum; i++ {
		DirGroup = append(DirGroup, []string{})
	}
	for _, dir := range dirList {
		randomNum := random(groupNum)
		DirGroup[randomNum] = append(DirGroup[randomNum], dir)
	}
	return DirGroup
}

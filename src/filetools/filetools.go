package filetools

import (
	"fmt"
	"os"
	"io/ioutil"
	"path/filepath"
)

func CreateDirIfNotExist(dir string) error {
	var err error
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Println("无法创建文件夹", dir)
			return err
		}
	}
	return err
}

func WriteFile(filePath string, data []byte, perm os.FileMode) error {
	var err error
	dir, _ := filepath.Split(filePath)
	err = CreateDirIfNotExist(dir)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filePath, data, perm)
	if err != nil {
		fmt.Println("无法写入文件", dir)
		return err
	}
	return err
}

func ReadFile(filePath string) ([]byte, error) {
	return ioutil.ReadFile(filePath)
}

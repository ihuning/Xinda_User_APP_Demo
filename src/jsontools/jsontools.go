package jsontools

import (
	"encoding/json"
	"fmt"
	"xindauserbackground/src/filetools"
)

//定义配置文件解析后的结构
type JsonRoot struct {
	SenderName   string     `json:"senderName"`
	ReceiverName string     `json:"receiverName"`
	PlatformList []Platform `json:"platformlist"`
	FileInfo     []FileInfo `json:"fileInfo"`
}

type Platform struct {
	PlatformName string `json:"platformName"`
	PlatformType string `json:"platformType"`
	PlatformURL  string `json:"platformURL"`
	UserName     string `json:"userName"`
	UserPassword string `json:"userPassword"`
}

type FileInfo struct {
	FileName     string `json:"fileName"`
	PlatformType string `json:"platformType"`
}

// 读取Json文件,并存储在root结构体中
func ReadJsonFile(filePath string, jsonRoot interface{}) error {
	var err error
	//ReadFile函数会读取文件的全部内容，并将结果以[]byte类型返回
	data, err := filetools.ReadFile(filePath)
	if err != nil {
		fmt.Println("无法读取Json文件")
		return err
	}
	//读取的数据为json格式，需要进行解码
	err = json.Unmarshal(data, jsonRoot)
	if err != nil {
		fmt.Println("无法解析Json文件")
		return err
	}
	return err
}

func WriteJsonFile(filePath string, jsonRoot JsonRoot) error {
	var err error
	jsonBytes, err := json.Marshal(jsonRoot)
	if err != nil {
		fmt.Println("无法生成Json文件")
		return err
	}
	err = filetools.WriteFile(filePath, jsonBytes, 0755)
	if err != nil {
		fmt.Println("无法写入Json文件")
		return err
	}
	return err
}

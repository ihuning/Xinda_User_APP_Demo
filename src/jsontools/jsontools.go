package jsontools

import (
	"fmt"
	"xindauserbackground/src/filetools"
	"github.com/Jeffail/gabs/v2"
)

type JsonParser struct {
	Parser *gabs.Container
}

// 读取Json文件,并生成一个parser
func ReadJsonFile(filePath string) (*JsonParser, error) {
	var err error
	//ReadFile函数会读取文件的全部内容，并将结果以[]byte类型返回
	data, err := filetools.ReadFile(filePath)
	if err != nil {
		fmt.Println("无法读取json文件", err)
		return nil, err
	}
	// 将读取到的json bytes转换为parser
	jsonParser, err := gabs.ParseJSON(data)
	if err != nil {
		fmt.Println("无法生成jsonParser", err)
		return nil, err
	}
	return &JsonParser{jsonParser}, err
}

// 打印出json的所有内容
func (j *JsonParser) PrintJsonContent() {
	fmt.Println(j.Parser.StringIndent("", "  "))
}

// 根据path获取对应的value
func (j *JsonParser) ReadJsonValue(path string) interface{} {
	gObj, err := j.Parser.JSONPointer(path)
	if err != nil {
		fmt.Println("无法从jsonParser找到path对应的value", err)
		return nil
	}
	return gObj.Data()
}

// fetch
func (j *JsonParser) SearchChildren(path string) []*JsonParser{
	var children []*JsonParser
	for _, c := range j.Parser.Search(path).Children() {
		children = append(children, &JsonParser{c})
	}
	return children
}

// 创建一个json文件
// func WriteJsonFile(filePath string, jsonRoot JsonRoot) error {
// 	var err error
// 	jsonBytes, err := json.Marshal(jsonRoot)
// 	if err != nil {
// 		fmt.Println("无法生成Json文件")
// 		return err
// 	}
// 	err = filetools.WriteFile(filePath, jsonBytes, 0755)
// 	if err != nil {
// 		fmt.Println("无法写入Json文件")
// 		return err
// 	}
// 	return err
// }

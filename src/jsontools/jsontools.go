package jsontools

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"xindauserbackground/src/filetools"
	"github.com/Jeffail/gabs/v2"
)

type JsonParser struct {
	Parser *gabs.Container
}

// 存储IFSS信息
type IFSSInfo struct {
	IFSSName         string
	IFSSType         string
	IFSSURL          string
	IFSSUserName     string
	IFSSUserPassword string
}

// 存储数据交换文件信息
type SpecFileInfo struct {
	IFSSName     string
	SpecFileName string
	FragmentSN   string
	PaddingSize  string
}

// 生成一个空的jsonparser
func GenerateNewJsonParser() *JsonParser {
	var jsonParser JsonParser
	jsonParser.Parser = gabs.New()
	return &jsonParser
}

// 读取Json文件,并生成一个parser
func ReadJsonFile(filePath string) (*JsonParser, error) {
	var err error
	jsonParser, err := gabs.ParseJSONFile(filePath)
	if err != nil {
		fmt.Println("无法读取json文件", err)
		return nil, err
	}
	return &JsonParser{jsonParser}, err
}

// 读取json格式的bytes,并生成一个parser
func ReadJsonBytes(jsonBytes []byte) (*JsonParser, error) {
	jsonParser, err := gabs.ParseJSON(jsonBytes)
	if err != nil {
		fmt.Println("无法读取json bytes", err)
		return nil, err
	}
	return &JsonParser{jsonParser}, err
}

// 读取json格式的string,并生成一个parser
func ReadJsonString(jsonString string) (*JsonParser, error) {
	return ReadJsonBytes([]byte(jsonString))
}

// 导出json的文本
func (j *JsonParser) GenerateJsonString() string {
	return j.Parser.StringIndent("", "    ")
}

// 导出json的bytes
func (j *JsonParser) GenerateJsonBytes() []byte {
	return j.Parser.BytesIndent("", "    ")
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

// 获取json数组中的所有成员
func (j *JsonParser) GetAllChildren(path string) []*JsonParser {
	var childrenList []*JsonParser
	for _, c := range j.Parser.Search(path).Children() {
		childrenList = append(childrenList, &JsonParser{c})
	}
	return childrenList
}

// 随机获取json数组中的一个成员
func (j *JsonParser) GetRandomChildren(path string) *JsonParser {
	childrenList := j.GetAllChildren(path)
	randomPosition, _ := rand.Int(rand.Reader, big.NewInt(int64(len(childrenList))))
	return childrenList[randomPosition.Int64()]
}

// 在json中某路径添加或修改数据的key和value
func (j *JsonParser) SetValue(value interface{}, hierarchy ...string) error {
	_, err := j.Parser.Set(value, hierarchy...)
	if err != nil {
		fmt.Println("无法在json中某路径添加或修改数据的key和value", err)
		return nil
	}
	return err
}

// 在json中某路径添加或修改数组的名称
func (j *JsonParser) SetArray(hierarchy ...string) error {
	_, err := j.Parser.Array(hierarchy...)
	if err != nil {
		fmt.Println("无法在json中某路径添加或修改数组的名称", err)
		return nil
	}
	return err
}

// 在json中某路径的数组中添加key和value
func (j *JsonParser) AppendArray(value interface{}, hierarchy ...string) error {
	err := j.Parser.ArrayAppend(value, hierarchy...)
	if err != nil {
		fmt.Println("无法在json中某路径的数组中添加key和value", err)
		return nil
	}
	return err
}

// 合并两个jsonparser
func (j *JsonParser) Merge(source *JsonParser) error {
	err := j.Parser.Merge(source.Parser)
	if err != nil {
		fmt.Println("无法合并两个jsonparser", err)
		return nil
	}
	return err
}

// 把json内容写入一个json文件
func (j *JsonParser) WriteJsonFile(filePath string) error {
	var err error
	err = filetools.WriteFile(filePath, j.GenerateJsonBytes(), 0755)
	if err != nil {
		fmt.Println("无法写入Json文件", err)
		return err
	}
	return err
}


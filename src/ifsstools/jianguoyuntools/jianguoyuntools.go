package jianguoyuntools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/ifsstools/jianguoyuntools/utils"

	"github.com/studio-b12/gowebdav"
)

type JianGuoYun struct {
	user     string
	password string
	url      string
	path     string
	client   *gowebdav.Client
}

// 配置一个坚果云连接
func NewJianGuoYunClient(url, user, password string) *JianGuoYun {
	client := gowebdav.NewClient(url, user, password)
	j := &JianGuoYun{
		url:      url,
		user:     user,
		password: password,
		path:     "/我的坚果云/",
		client:   client,
	}
	return j
}

// 列出坚果云的路径中所有文件的路径
func (j JianGuoYun) list(fileList *[]*utils.FileStat, path string) {
	files, _ := j.client.ReadDir(path)
	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		filePath = filepath.ToSlash(filePath) // 防止windows强制转换斜杠的格式
		if path == filePath {
			continue
		}
		if file.IsDir() {
			f := &utils.FileStat{
				Path:         filePath,
				FileType:     utils.Dir,
				LastModified: file.ModTime().Unix(),
			}
			*fileList = append(*fileList, f)
			j.list(fileList, filePath)
		} else {
			f := &utils.FileStat{
				Path:         filePath,
				FileType:     utils.File,
				LastModified: file.ModTime().Unix(),
			}
			*fileList = append(*fileList, f)
		}
	}
}

// 下载单个文件
func (j JianGuoYun) DownloadFile(jgyPath, localPath string) error {
	var err error
	tmpDir := filepath.Dir(localPath)
	if !utils.FileIsExists(tmpDir) {
		err = os.MkdirAll(tmpDir, 0755)
		if err != nil {
			fmt.Println("无法创建文件夹")
			return err
		}
	}
	data, err := j.client.ReadStream(jgyPath)
	if _, ok := err.(*os.PathError); ok {
		err = fmt.Errorf("坚果云中要找的文件不存在")
		fmt.Println(err)
		return err
	}
	if err != nil {
		fmt.Println("无法读取坚果云文件流")
		return err
	}
	fd, err := os.OpenFile(localPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		fmt.Println("无法打开创建的本地文件")
		return err
	}
	_, err = io.Copy(fd, data)
	if err != nil {
		fmt.Println("无法将下载的坚果云文件流写入本地文件")
		return err
	}
	defer fd.Close()
	return err
}

// 上传单个文件
func (j JianGuoYun) UploadFile(jgyPath, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		fmt.Println("无法打开创建的本地文件")
		return err
	}
	defer file.Close()
	j.client.WriteStream(jgyPath, file, 0755)
	// 检查文件是否上传成功
	_, err = j.client.Read(jgyPath)
	if err != nil {
		fmt.Println("上传到坚果云失败")
		return err
	}
	return err
}

// 测试能不能连上坚果云
func TestJianGuoYunConnection(url, username, password string) error {
	var err error
	j := NewJianGuoYunClient(url, username, password)
	var fileList = make([]*utils.FileStat, 0)
	j.list(&fileList, j.path)
	if fileList == nil { // 正常情况下应该有个"我的坚果云",没有的话就是连接不上
		err = fmt.Errorf("无法连接坚果云")
		fmt.Println(err)
		return err
	}
	return err
}

// 上传一个文件夹中的所有数据交换文件
func UploadAllFilesFromFolder(url, folderDir, username, password string) error {
	var err error
	filePathList, fileNameList, err := filetools.GenerateSpecFilePathNameListFromFolder(folderDir)
	if err != nil {
		return err
	}
	j := NewJianGuoYunClient(url, username, password)
	for i := 0; i < len(filePathList); i++ {
		jgyPath := filepath.Join(j.path, fileNameList[i])
		jgyPath = filepath.ToSlash(jgyPath) // 防止windows强制转换斜杠的格式
		localPath := filePathList[i]
		err = j.UploadFile(jgyPath, localPath)
		if err != nil {
			return err
		} else {
			fmt.Println("数据交换文件", fileNameList[i], "使用WebDav方式成功发送到了", url, "使用的账户为", username)
		}
	}
	return err
}

// 下载一个文件夹里面的所有数据交换文件
func DownloadAllFilesToFolder(url, folderDir, username, password string) error {
	var err error
	var jgyFileStatList = make([]*utils.FileStat, 0)
	j := NewJianGuoYunClient(url, username, password)
	j.list(&jgyFileStatList, j.path)
	for _, jgyFileStat := range jgyFileStatList {
		jgyFilePath := jgyFileStat.Path
		_, jgyFileName := filepath.Split(jgyFilePath)
		localPath := filepath.Join(folderDir, jgyFileName)
		err = j.DownloadFile(jgyFilePath, localPath)
		if err != nil {
			return err
		} else {
			fmt.Println("数据交换文件", jgyFileName, "从", url, "使用WebDav方式成功下载", "使用的账户为", username)
		}
	}
	return err
}

// 清除坚果云中"我的坚果云"文件夹里面的所有文件
func CleanJianguoyun(url, username, password string) error {
	var err error
	var jgyFileStatList = make([]*utils.FileStat, 0)
	j := NewJianGuoYunClient(url, username, password)
	j.list(&jgyFileStatList, j.path)
	for _, jgyFileStat := range jgyFileStatList {
		jgyFilePath := jgyFileStat.Path
		err = j.client.Remove(jgyFilePath)
		if err != nil {
			return err
		}
	}
	return err
}

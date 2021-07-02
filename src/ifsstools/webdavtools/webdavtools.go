package webdavtools

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/jsontools"
	"xindauserbackground/src/ifsstools/webdavtools/utils"

	"github.com/studio-b12/gowebdav"
)

type Webdav struct {
	UserName  string
	Password  string
	Url       string
	WebdavDir string
	LocalDir  string
	Client    *gowebdav.Client
}

// 配置一个Webdav连接
func NewWebdavClient(url, localDir, userName, password string) Webdav {
	client := gowebdav.NewClient(url, userName, password)
	w := Webdav{
		Url:       url,
		UserName:  userName,
		Password:  password,
		WebdavDir: "/tmp_data_transmission/", // 数据传输文件存储在webdav网盘根目录下的这个文件夹中
		LocalDir:  localDir,
		Client:    client,
	}
	filetools.Mkdir(localDir)
	return w
}

// 列出Webdav的路径中所有文件的路径
func (w Webdav) list(fileList *[]*utils.FileStat, path string) {
	files, _ := w.Client.ReadDir(path)
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
			w.list(fileList, filePath)
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
func (w Webdav) DownloadFile(webdavDir, localPath string) error {
	var err error
	tmpDir := filepath.Dir(localPath)
	if !utils.FileIsExists(tmpDir) {
		err = os.MkdirAll(tmpDir, 0755)
		if err != nil {
			fmt.Println("无法创建本地文件夹", err)
			return err
		}
	}
	data, err := w.Client.ReadStream(webdavDir)
	if _, ok := err.(*os.PathError); ok {
		fmt.Println("无法找到Webdav中的文件", err)
		return err
	}
	if err != nil {
		fmt.Println("无法读取Webdav文件流", err)
		return err
	}
	fd, err := os.OpenFile(localPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		fmt.Println("无法打开创建的本地文件", err)
		return err
	}
	_, err = io.Copy(fd, data)
	if err != nil {
		fmt.Println("无法将下载的Webdav文件流写入本地文件", err)
		return err
	}
	defer fd.Close()
	return err
}

// 上传单个文件
func (w Webdav) UploadFile(webdavDir, localPath string) error {
	file, err := os.Open(localPath)
	if err != nil {
		fmt.Println("无法打开创建的本地文件", err)
		return err
	}
	defer file.Close()
	w.Client.WriteStream(webdavDir, file, 0777)
	// 检查文件是否上传成功
	_, err = w.Client.Read(webdavDir)
	if err != nil {
		fmt.Println("无法上传到Webdav", err)
		return err
	}
	return err
}

// 上传一个文件夹中的所有数据交换文件
func (w Webdav) UploadAllFilesFromFolder(sendProgressChannel chan []byte) error {
	var err error
	err = w.Client.Mkdir(w.WebdavDir, 0777) // 如果不存在用来存储数据的临时文件夹,就创建一个
	if err != nil {
		fmt.Println("无法在Webdav中创建新文件夹", err)
		return err
	}
	filePathList, fileNameList, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(w.LocalDir)
	if err != nil {
		return err
	}
	for i := 0; i < len(filePathList); i++ {
		webdavDir := filepath.Join(w.WebdavDir, fileNameList[i])
		webdavDir = filepath.ToSlash(webdavDir) // 防止windows强制转换斜杠的格式
		localPath := filePathList[i]
		err = w.UploadFile(webdavDir, localPath)
		if err != nil {
			fmt.Println("无法上传文件到Webdav", err)
			return err
		} else {
			sendProgressChannelJsonBytes := jsontools.GenerateSendProgressChannelJsonBytes(fileNameList[i], w.Url, w.UserName, 1)
			sendProgressChannel <- sendProgressChannelJsonBytes
		}
	}
	return err
}

// 下载一个文件夹里面的所有数据交换文件
func (w Webdav) DownloadAllFilesToFolder(receiveProgressChannel chan []byte) error {
	var err error
	var webdavFileStatList = make([]*utils.FileStat, 0)
	w.list(&webdavFileStatList, w.WebdavDir)
	for _, webdavFileStat := range webdavFileStatList {
		webdavFilePath := webdavFileStat.Path
		_, webdavFileName := filepath.Split(webdavFilePath)
		localPath := filepath.Join(w.LocalDir, webdavFileName)
		err = w.DownloadFile(webdavFilePath, localPath)
		if err != nil {
			fmt.Println("无法从Webdav下载文件", err)
			return err
		} else {
			receiveProgressChannelJsonBytes := jsontools.GenerateReceiveProgressChannelJsonBytes(webdavFileName, w.Url, w.UserName)
			receiveProgressChannel <- receiveProgressChannelJsonBytes
		}
	}
	return err
}

// 清除之前使用Webdav下载过的文件
func (w Webdav) CleanWebdav() error {
	var err error
	_, fileNameList, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(w.LocalDir)
	if err != nil {
		return err
	}
	if len(fileNameList) == 0 {
		fmt.Println("没有在", w.Url, "中检测到需要下载的内容")
		return err
	}
	for _, fileName := range fileNameList {
		webdavDir := filepath.Join(w.WebdavDir, fileName)
		webdavDir = filepath.ToSlash(webdavDir) // 防止windows强制转换斜杠的格式
		err = w.Client.Remove(webdavDir)
		if err != nil {
			fmt.Println("无法清除Webdav的文件", err)
			return err
		}
	}
	fmt.Println("Webdav", w.Url, "中的内容已被成功清除", "使用的账户为", w.UserName)
	return err
}

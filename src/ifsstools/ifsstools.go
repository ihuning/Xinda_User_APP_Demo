package ifsstools

import (
	"path/filepath"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/ifsstools/gittools"
	"xindauserbackground/src/ifsstools/webdavtools"
	"xindauserbackground/src/jsontools"
)

type IFSSFolderInfo struct {
	IFSSFolderDir    string
	IFSSType         string
	IFSSURL          string
	IFSSUserName     string
	IFSSUserPassword string
}

func generateIFSSFolders(folderDir, configFilePath string) ([]IFSSFolderInfo, error) {
	var err error
	jsonparser, err := jsontools.ReadJsonFile(configFilePath)
	if err != nil {
		return nil, err
	}
	var IFSSFolderInfoList []IFSSFolderInfo
	for _, IFSSInfo := range jsonparser.Search("IFSSInfoList").Children() {
		var IFSSFolderInfo IFSSFolderInfo
		IFSSFolderInfo.IFSSFolderDir = filepath.Join(folderDir, jsontools.ReadJsonValue(IFSSInfo, "/IFSSName").(string))
		IFSSFolderInfo.IFSSType = jsontools.ReadJsonValue(IFSSInfo, "/IFSSType").(string)
		IFSSFolderInfo.IFSSURL = jsontools.ReadJsonValue(IFSSInfo, "/IFSSURL").(string)
		IFSSFolderInfo.IFSSUserName = jsontools.ReadJsonValue(IFSSInfo, "/IFSSUserName").(string)
		IFSSFolderInfo.IFSSUserPassword = jsontools.ReadJsonValue(IFSSInfo, "/IFSSUserPassword").(string)
		IFSSFolderInfoList = append(IFSSFolderInfoList, IFSSFolderInfo)
	}
	return IFSSFolderInfoList, err
}

// 上传文件夹中的数据交换文件到IFSS
func UploadToIFSS(folderDir, configFilePath string) error {
	var err error
	IFSSFolderInfoList, err := generateIFSSFolders(folderDir, configFilePath)
	if err != nil {
		return err
	}
	for _, IFSSFolderInfo := range IFSSFolderInfoList {
		switch IFSSFolderInfo.IFSSType {
		case "git":
			g := gittools.NewGitClient(IFSSFolderInfo.IFSSURL, IFSSFolderInfo.IFSSFolderDir, IFSSFolderInfo.IFSSUserName, IFSSFolderInfo.IFSSUserPassword)
			err = g.CloneRepository()
			if err != nil {
				return err
			}
			err = g.PushToRepository()
			if err != nil {
				return err
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(IFSSFolderInfo.IFSSURL, IFSSFolderInfo.IFSSFolderDir, IFSSFolderInfo.IFSSUserName, IFSSFolderInfo.IFSSUserPassword)
			err = w.UploadAllFilesFromFolder()
			if err != nil {
				return err
			}
		default:
			panic("IFSS类型错误")
		}
	}

	err = filetools.Rmdir(folderDir) // 删除本地文件,销毁上传记录
	return err
}

// 从IFSS下载数据交换文件到文件夹中
func DownloadFromIFSS(folderDir, configFilePath string) error {
	var err error
	IFSSFolderInfoList, err := generateIFSSFolders(folderDir, configFilePath)
	if err != nil {
		return err
	}
	for _, IFSSFolderInfo := range IFSSFolderInfoList {
		switch IFSSFolderInfo.IFSSType {
		case "git":
			g := gittools.NewGitClient(IFSSFolderInfo.IFSSURL, IFSSFolderInfo.IFSSFolderDir, IFSSFolderInfo.IFSSUserName, IFSSFolderInfo.IFSSUserPassword)
			err = g.CloneRepository()
			if err != nil {
				return err
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(IFSSFolderInfo.IFSSURL, IFSSFolderInfo.IFSSFolderDir, IFSSFolderInfo.IFSSUserName, IFSSFolderInfo.IFSSUserPassword)
			err = w.DownloadAllFilesToFolder()
			if err != nil {
				return err
			}
		default:
			panic("IFSS类型错误")
		}
		// 把多通道下载的数据交换文件集中到一个文件夹
		filePathList, _, _ := filetools.GenerateSpecFilePathNameListFromFolder(IFSSFolderInfo.IFSSFolderDir)
		filetools.MoveFilesToNewFolder(filePathList, folderDir)
	}
	return err
}

// 在通信完成时,删除IFSS和本地目录中的所有的数据交换文件,以销毁通信记录
func CleanIFSS(folderDir, configFilePath string) error {
	var err error
	IFSSFolderInfoList, err := generateIFSSFolders(folderDir, configFilePath)
	if err != nil {
		return err
	}
	for _, IFSSFolderInfo := range IFSSFolderInfoList {
		switch IFSSFolderInfo.IFSSType {
		case "git":
			g := gittools.NewGitClient(IFSSFolderInfo.IFSSURL, IFSSFolderInfo.IFSSFolderDir, IFSSFolderInfo.IFSSUserName, IFSSFolderInfo.IFSSUserPassword)
			err = g.CleanRepository()
			if err != nil {
				return err
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(IFSSFolderInfo.IFSSURL, IFSSFolderInfo.IFSSFolderDir, IFSSFolderInfo.IFSSUserName, IFSSFolderInfo.IFSSUserPassword)
			err = w.CleanWebdav()
			if err != nil {
				return err
			}
		default:
			panic("IFSS类型错误")
		}
	}
	err = filetools.Rmdir(folderDir) // 删除本地文件,销毁下载记录
	return err
}

package ifsstools

import (
	"path/filepath"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/ifsstools/gittools"
	"xindauserbackground/src/ifsstools/webdavtools"
	"xindauserbackground/src/jsontools"
)

// 从json中读取IFSSInfoList
func generateIFSSInfoListFromJson(configFilePath string) ([]jsontools.IFSSInfo, error) {
	var err error
	j, err := jsontools.ReadJsonFile(configFilePath)
	if err != nil {
		return nil, err
	}
	var IFSSInfoList []jsontools.IFSSInfo
	for _, children := range j.GetAllChildren("IFSSInfoList") {
		var IFSSInfo jsontools.IFSSInfo
		IFSSInfo.IFSSName = children.ReadJsonValue("/IFSSName").(string)
		IFSSInfo.IFSSType = children.ReadJsonValue("/IFSSType").(string)
		IFSSInfo.IFSSURL = children.ReadJsonValue("/IFSSURL").(string)
		IFSSInfo.IFSSUserName = children.ReadJsonValue("/IFSSUserName").(string)
		IFSSInfo.IFSSUserPassword = children.ReadJsonValue("/IFSSUserPassword").(string)
		IFSSInfoList = append(IFSSInfoList, IFSSInfo)
	}
	return IFSSInfoList, err
}

// 上传文件夹中的数据交换文件到IFSS
func UploadToIFSS(folderDir, configFilePath string) error {
	var err error
	IFSSInfoList, err := generateIFSSInfoListFromJson(configFilePath)
	if err != nil {
		return err
	}
	for _, IFSSInfo := range IFSSInfoList {
		IFSSFolderDir := filepath.Join(folderDir, IFSSInfo.IFSSName)
		switch IFSSInfo.IFSSType {
		case "git":
			g := gittools.NewGitClient(IFSSInfo.IFSSURL, IFSSFolderDir, IFSSInfo.IFSSUserName, IFSSInfo.IFSSUserPassword)
			err = g.CloneRepository()
			if err != nil {
				return err
			}
			err = g.PushToRepository()
			if err != nil {
				return err
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(IFSSInfo.IFSSURL, IFSSFolderDir, IFSSInfo.IFSSUserName, IFSSInfo.IFSSUserPassword)
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
	IFSSInfoList, err := generateIFSSInfoListFromJson(configFilePath)
	if err != nil {
		return err
	}
	for _, IFSSInfo := range IFSSInfoList {
		IFSSFolderDir := filepath.Join(folderDir, IFSSInfo.IFSSName)
		switch IFSSInfo.IFSSType {
		case "git":
			g := gittools.NewGitClient(IFSSInfo.IFSSURL, IFSSFolderDir, IFSSInfo.IFSSUserName, IFSSInfo.IFSSUserPassword)
			err = g.CloneRepository()
			if err != nil {
				return err
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(IFSSInfo.IFSSURL, IFSSFolderDir, IFSSInfo.IFSSUserName, IFSSInfo.IFSSUserPassword)
			err = w.DownloadAllFilesToFolder()
			if err != nil {
				return err
			}
		default:
			panic("IFSS类型错误")
		}
		// 把多通道下载的数据交换文件集中到一个文件夹
		filePathList, _, _ := filetools.GenerateSpecFilePathNameListFromFolder(IFSSFolderDir)
		filetools.MoveFilesToNewFolder(filePathList, folderDir)
	}
	return err
}

// 在通信完成时,删除IFSS和本地目录中的所有的数据交换文件,以销毁通信记录
func CleanIFSS(folderDir, configFilePath string) error {
	var err error
	IFSSInfoList, err := generateIFSSInfoListFromJson(configFilePath)
	if err != nil {
		return err
	}
	for _, IFSSInfo := range IFSSInfoList {
		IFSSFolderDir := filepath.Join(folderDir, IFSSInfo.IFSSName)
		switch IFSSInfo.IFSSType {
		case "git":
			g := gittools.NewGitClient(IFSSInfo.IFSSURL, IFSSFolderDir, IFSSInfo.IFSSUserName, IFSSInfo.IFSSUserPassword)
			err = g.CleanRepository()
			if err != nil {
				return err
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(IFSSInfo.IFSSURL, IFSSFolderDir, IFSSInfo.IFSSUserName, IFSSInfo.IFSSUserPassword)
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

package ifsstools

import (
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/ifsstools/gittools"
	"xindauserbackground/src/ifsstools/webdavtools"
)


// 上传文件夹中的数据交换文件到IFSS
func UploadToIFSS(ifssType, url, folderDir, username, password string) error {
	var err error
	switch ifssType {
	case "git":
		g := gittools.NewGitClient(url, folderDir, username, password)
		err = g.CloneRepository()
		if err != nil {
			return err
		}
		err = g.PushToRepository()
		if err != nil {
			return err
		}
	case "jianguoyun":
		w := webdavtools.NewWebdavClient(url, folderDir, username, password)
		err = w.UploadAllFilesFromFolder()
		if err != nil {
			return err
		}
	default:
		panic("IFSS类型错误")
	}
	err = filetools.Rmdir(folderDir) // 删除本地文件,销毁上传记录
	return err
}

// 从IFSS下载数据交换文件到文件夹中
func DownloadFromIFSS(ifssType, url, folderDir, username, password string) error {
	var err error
	switch ifssType {
	case "git":
		g := gittools.NewGitClient(url, folderDir, username, password)
		err = g.CloneRepository()
		if err != nil {
			return err
		}
	case "jianguoyun":
		w := webdavtools.NewWebdavClient(url, folderDir, username, password)
		err = w.DownloadAllFilesToFolder()
		if err != nil {
			return err
		}
	default:
		panic("IFSS类型错误")
	}
	return err
}

// 在通信完成时,删除IFSS和本地目录中的所有的数据交换文件,以销毁通信记录
func CleanIFSS(ifssType, url, folderDir, username, password string) error {
	var err error
	switch ifssType {
	case "git":
		g := gittools.NewGitClient(url, folderDir, username, password)
		err = g.CleanRepository()
		if err != nil {
			return err
		}
	case "jianguoyun":
		w := webdavtools.NewWebdavClient(url, folderDir, username, password)
		err = w.CleanWebdav()
		if err != nil {
			return err
		}
	default:
		panic("IFSS类型错误")
	}
	err = filetools.Rmdir(folderDir) // 删除本地文件,销毁下载记录
	return err
}

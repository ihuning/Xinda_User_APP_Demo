package ifsstools

import (
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/ifsstools/gittools"
	"xindauserbackground/src/ifsstools/jianguoyuntools"
)

// 初始化IFSS连接,指定通信用到的文件夹(在通信开始前,IFSS应该被clean过)
func TestIFSSConnection(ifssType, url, username, password string) error {
	var err error
	switch ifssType {
	case "git":
		err = gittools.TestGitConnection(url, username, password)
		if err != nil {
			return err
		}
	case "jianguoyun":
		err = jianguoyuntools.TestJianGuoYunConnection(url, username, password)
		if err != nil {
			return err
		}
	default:
		panic("IFSS类型错误")
	}
	return err
}

// 上传文件夹中的数据交换文件到IFSS
func UploadToIFSS(ifssType, url, folderDir, username, password string) error {
	var err error
	switch ifssType {
	case "git":
		err = gittools.CloneRepository(url, folderDir, username, password)
		if err != nil {
			return err
		}
		err = gittools.PushToRepository(url, folderDir, username, password)
		if err != nil {
			return err
		}
	case "jianguoyun":
		err = jianguoyuntools.UploadAllFilesFromFolder(url, folderDir, username, password)
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
		err = gittools.CloneRepository(url, folderDir, username, password)
		if err != nil {
			return err
		}
	case "jianguoyun":
		err = jianguoyuntools.DownloadAllFilesToFolder(url, folderDir, username, password)
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
		err = gittools.CleanRepository(folderDir, username, password)
		if err != nil {
			return err
		}
	case "jianguoyun":
		err = jianguoyuntools.CleanJianguoyun(url, username, password)
		if err != nil {
			return err
		}
	default:
		panic("IFSS类型错误")
	}
	err = filetools.Rmdir(folderDir) // 删除本地文件,销毁下载记录
	return err
}

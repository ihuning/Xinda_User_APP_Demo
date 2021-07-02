package ifsstools

import (
	"fmt"
	"path/filepath"
	"sync"
	"xindauserbackground/src/crypto/rsatools"
	"xindauserbackground/src/filetools"
	"xindauserbackground/src/ifsstools/gittools"
	"xindauserbackground/src/ifsstools/webdavtools"
	"xindauserbackground/src/jsontools"
	"xindauserbackground/src/ziptools"
)

// 上传文件夹中的数据交换文件到IFSS
func UploadToIFSS(sendFolderDir string, neighborJsonParser *jsontools.JsonParser, sendProgressChannel chan []byte) error {
	var err error
	if !filetools.IsPathExists(sendFolderDir) || filetools.IsFolderEmpty(sendFolderDir) {
		err = fmt.Errorf("无法在文件夹中找到需要上传到IFSS的文件")
		return err
	}
	fmt.Println("在", sendFolderDir, "中找到了需要上传到IFSS的文件")
	_, receiverName := filepath.Split(sendFolderDir)
	neighborPublicKeyString := neighborJsonParser.ReadJsonValue("/PublicKey").(string)
	neighborPublicKey, err := rsatools.StringToPublicKey(neighborPublicKeyString)
	receiverAccountParserList := neighborJsonParser.GetAllChildren("OwnAccountList")
	filePathList, _, _ := filetools.GenerateUnhiddenFilePathNameListFromFolder(sendFolderDir)
	filePathGroup := filetools.DivideDirListToGroup(filePathList, len(receiverAccountParserList))
	uploadGoroutine := func(wg *sync.WaitGroup, children *jsontools.JsonParser, filePathList []string) {
		defer wg.Done()
		ifssName := children.ReadJsonValue("/IFSSName").(string)
		ifssFolderDir := filepath.Join(sendFolderDir, ifssName)
		// 生成一个新的文件夹,使用该IFSS账号的所有文件都储存在这个文件夹中
		err = filetools.Mkdir(ifssFolderDir)
		if err != nil {
			return
		}
		for _, filePath := range filePathList {
			_, fileName := filepath.Split(filePath)
			// 将数据交换文件移动到新的文件夹
			tempFolderDir := filepath.Join(ifssFolderDir, fileName+"_ready_to_zip")
			specFilePath := filepath.Join(tempFolderDir, fileName)
			filetools.Rename(filePath, specFilePath)
			encryptedReceiverName, err := rsatools.EncryptWithPublicKey([]byte(receiverName), neighborPublicKey)
			if err != nil {
				return
			}
			infoFilePath := filepath.Join(tempFolderDir, fileName+"_")
			err = filetools.WriteFile(infoFilePath, encryptedReceiverName, 0777)
			if err != nil {
				return
			}
			tempFilePathList := append([]string{}, specFilePath, infoFilePath)
			zipFilePath := filepath.Join(ifssFolderDir, fileName)
			err = ziptools.ZipFiles(tempFilePathList, zipFilePath)
			if err != nil {
				return
			}
			err = filetools.RmDir(tempFolderDir)
			if err != nil {
				return
			}
		}
		// 使用IFSS账号将本地文件上传到IFSS平台
		ifssType := children.ReadJsonValue("/IFSSType").(string)
		ifssURL := children.ReadJsonValue("/IFSSURL").(string)
		ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
		ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
		switch ifssType {
		case "git":
			g := gittools.NewGitClient(ifssURL, ifssFolderDir, ifssUserName, ifssPassword)
			err = g.CloneRepository()
			if err != nil {
				return
			}
			err = g.PushToRepository(sendProgressChannel)
			if err != nil {
				return
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(ifssURL, ifssFolderDir, ifssUserName, ifssPassword)
			err = w.UploadAllFilesFromFolder(sendProgressChannel)
			if err != nil {
				return
			}
		default:
			panic("IFSS类型错误")
		}
		filetools.RmDir(ifssFolderDir) // 删除IFSS上传使用的文件夹
	}
	var wg sync.WaitGroup // 信号量
	wg.Add(len(filePathGroup))
	for i, filePathList := range filePathGroup {
		children := receiverAccountParserList[i]        // 被选中的IFSS平台
		go uploadGoroutine(&wg, children, filePathList) // 将list中的所有文件打包后上传到这个平台
	}
	// for _, filePath := range filePathList {
	// 	filetools.RmFile(filePath)
	// }
	wg.Wait()
	// err = filetools.RmDir(sendFolderDir) // 删除本地文件,销毁上传记录
	return err
}

// 从IFSS下载数据交换文件到receiveDir的以最终接收者命名的文件夹中
func DownloadFromIFSS(userPrivateKeyPath string, ownAccountListJsonParser *jsontools.JsonParser, receiveDir string, receiveProgressChannel chan []byte) ([]string, error) {
	var err error
	var saveDir string
	type void struct{}
	var voidMember void
	saveDirListSet := make(map[string]void) // 为了去重
	var saveDirList []string
	userPrivateKey, err := rsatools.ReadPrivateKeyFile(userPrivateKeyPath)
	if err != nil {
		return nil, err
	}
	downloadGoroutine := func(wg *sync.WaitGroup, children *jsontools.JsonParser) {
		defer wg.Done()
		ifssName := children.ReadJsonValue("/IFSSName").(string)
		ifssDownloadDir := filepath.Join(receiveDir, ifssName)
		ifssType := children.ReadJsonValue("/IFSSType").(string)
		ifssURL := children.ReadJsonValue("/IFSSURL").(string)
		ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
		ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
		switch ifssType {
		case "git":
			g := gittools.NewGitClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			// if err.Error() == "没有需要下载的内容" {
			// 	return
			// }
			err = g.DownloadFromRepository(receiveProgressChannel)
			// if err == nil {
			// err = g.CleanRepository()
			// }
			// if err != nil {
			// 	return
			// }
		case "webdav":
			w := webdavtools.NewWebdavClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			err = w.DownloadAllFilesToFolder(receiveProgressChannel)
			// err = w.CleanWebdav()
			// if err != nil {
			// 	return
			// }
		default:
			panic("IFSS类型错误")
		}
		filePathList, _, err := filetools.GenerateUnhiddenFilePathNameListFromFolder(ifssDownloadDir)
		if err != nil || filePathList == nil {
			return
		}
		for _, filePath := range filePathList {
			_, fileName := filepath.Split(filePath)
			unzipFolderDir := filepath.Join(ifssDownloadDir, fileName+"_ziptemp")
			err = filetools.Mkdir(unzipFolderDir)
			if err != nil {
				return
			}
			_, err = ziptools.UnzipFile(filePath, unzipFolderDir)
			if err != nil {
				return
			}
			// 找到配置文件,并解出接收方是谁
			specFilePath := filepath.Join(unzipFolderDir, fileName)
			infoFilePath := filepath.Join(unzipFolderDir, fileName+"_")
			encryptedReceiverNameBytes, err := filetools.ReadFile(infoFilePath)
			if err != nil {
				return
			}
			receiverNameBytes, err := rsatools.DecryptWithPrivateKey(encryptedReceiverNameBytes, userPrivateKey)
			if err != nil {
				return
			}
			// 数据交换文件最终存储的文件夹位置
			saveDir = filepath.Join(receiveDir, string(receiverNameBytes))
			if !filetools.IsPathExists(filepath.Join(saveDir, fileName)) {
				saveDirListSet[saveDir] = voidMember
				err = filetools.Rename(specFilePath, filepath.Join(saveDir, fileName))
				if err != nil {
					return
				}
			}
			err = filetools.RmDir(unzipFolderDir)
			if err != nil {
				return
			}
		}
	}
	ownAccountParserList := ownAccountListJsonParser.GetAllChildren("OwnAccountList")
	var wg sync.WaitGroup
	wg.Add(len(ownAccountParserList))
	for _, children := range ownAccountParserList {
		go downloadGoroutine(&wg, children)
	}
	wg.Wait()
	for key := range saveDirListSet {
		saveDirList = append(saveDirList, key) // 利用set去重
	}
	return saveDirList, err
}

// 在通信完成时,删除IFSS中的所有的数据交换文件,以销毁通信记录
func CleanIFSS(ownAccountListJsonParser *jsontools.JsonParser, receiveDir string) error {
	var err error
	for _, children := range ownAccountListJsonParser.GetAllChildren("OwnAccountList") {
		ifssName := children.ReadJsonValue("/IFSSName").(string)
		ifssDownloadDir := filepath.Join(receiveDir, ifssName)
		ifssType := children.ReadJsonValue("/IFSSType").(string)
		ifssURL := children.ReadJsonValue("/IFSSURL").(string)
		if !filetools.IsPathExists(ifssDownloadDir) { // 如果没有检测到下载下来了新内容
			continue
		}
		// 删除在线记录
		switch ifssType {
		case "git":
			ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
			ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
			g := gittools.NewGitClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			err = g.CleanRepository()
			if err != nil {
				return err
			}
		case "webdav":
			ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
			ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
			w := webdavtools.NewWebdavClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			err = w.CleanWebdav()
			if err != nil {
				return err
			}
		default:
			panic("IFSS类型错误")
		}
		// fmt.Println("已经成功清除在线仓库", ifssURL, "中的内容")
		// 删除本地记录
		err = filetools.RmDir(ifssDownloadDir)
	}
	return err
}

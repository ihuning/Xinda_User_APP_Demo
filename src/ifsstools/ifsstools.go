package ifsstools

import (
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
func UploadToIFSS(sendFolderDir string, ownAccountListJsonParser *jsontools.JsonParser, neighborJsonParser *jsontools.JsonParser) error {
	var err error
	_, receiverName := filepath.Split(sendFolderDir)
	neighborPublicKeyString := neighborJsonParser.ReadJsonValue("/PublicKey").(string)
	neighborPublicKey, err := rsatools.StringToPublicKey(neighborPublicKeyString)
	var ifssNameList []string
	for _, children := range neighborJsonParser.Parser.Search("IFSSNameList").Children() {
		ifssName := children.Data().(string)
		ifssNameList = append(ifssNameList, ifssName)
	}
	filePathList, _, _ := filetools.GenerateSpecFilePathNameListFromFolder(sendFolderDir)
	filePathGroup := filetools.DivideDirListToGroup(filePathList, len(ifssNameList))
	var wg sync.WaitGroup // 信号量
	uploadGoroutine := func(wg *sync.WaitGroup, ifssName string, filePathList []string) {
		ifssFolderDir := filepath.Join(sendFolderDir, ifssName)
		// 生成一个新的文件夹,使用该IFSS账号的所有文件都储存在这个文件夹中
		err = filetools.Mkdir(ifssFolderDir)
		if err != nil {
			return
		}
		zipGoroutine := func(wgZip *sync.WaitGroup, filePath string) {
			_, fileName := filepath.Split(filePath)
			// 将数据交换文件移动到新的文件夹
			tempFolderDir := filepath.Join(ifssFolderDir, fileName + "_ready_to_zip")
			specFilePath := filepath.Join(tempFolderDir, fileName)
			filetools.Rename(filePath, specFilePath)
			encryptedReceiverName, err := rsatools.EncryptWithPublicKey([]byte(receiverName), neighborPublicKey)
			if err != nil {
				return
			}
			infoFilePath := filepath.Join(tempFolderDir, fileName + "_")
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
			wgZip.Done()
		}
		var wgZip sync.WaitGroup
		wgZip.Add(len(filePathList))
		for _, filePath := range filePathList {
			zipGoroutine(&wgZip, filePath)
		}
		wgZip.Wait()
		// 使用IFSS账号将本地文件上传到IFSS平台
		for _, children := range ownAccountListJsonParser.GetAllChildren("OwnAccountList") {
			if children.ReadJsonValue("/IFSSName").(string) == ifssName {
				ifssType := children.ReadJsonValue("/IFSSType").(string)
				switch ifssType {
				case "git":
					ifssURL := children.ReadJsonValue("/IFSSURL").(string)
					ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
					ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
					g := gittools.NewGitClient(ifssURL, ifssFolderDir, ifssUserName, ifssPassword)
					err = g.CloneRepository()
					if err != nil {
						return
					}
					err = g.PushToRepository()
					if err != nil {
						return
					}
				case "webdav":
					ifssURL := children.ReadJsonValue("/IFSSURL").(string)
					ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
					ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
					w := webdavtools.NewWebdavClient(ifssURL, ifssFolderDir, ifssUserName, ifssPassword)
					err = w.UploadAllFilesFromFolder()
					if err != nil {
						return
					}
				default:
					panic("IFSS类型错误")
				}
			}
		}
		wg.Done()
	}
	wg.Add(len(filePathGroup))
	for i, filePathList := range filePathGroup {	
		ifssName := ifssNameList[i]	
		go uploadGoroutine(&wg, ifssName, filePathList)
	}
	wg.Wait()
	err = filetools.RmDir(sendFolderDir) // 删除本地文件,销毁上传记录
	return err
}

// 从IFSS下载数据交换文件到receiveDir的以最终接收者命名的文件夹中
func DownloadFromIFSS(userPrivateKeyPath string, ownAccountListJsonParser *jsontools.JsonParser, receiveDir string) ([]string, error) {
	var err error
	// identity := ownAccountListJsonParser.ReadJsonValue("/Identity").(string)
	var saveDir string // 如果身份是ap,则将下载的文件存储在以[接收者]命名的文件夹中;如果身份是接收者,则存储在以[Identification]命名的文件夹中
	type void struct{}
	var member void
	saveDirListSet := make(map[string]void) // 为了去重
	var saveDirList []string
	for _, children := range ownAccountListJsonParser.GetAllChildren("OwnAccountList") {
		ifssName := children.ReadJsonValue("/IFSSName").(string)
		ifssDownloadDir := filepath.Join(receiveDir, ifssName)
		ifssType := children.ReadJsonValue("/IFSSType").(string)
		switch ifssType {
		case "git":
			ifssURL := children.ReadJsonValue("/IFSSURL").(string)
			ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
			ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
			g := gittools.NewGitClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			err = g.CloneRepository()
			if err != nil {
				return nil, err
			}
		case "webdav":
			ifssURL := children.ReadJsonValue("/IFSSURL").(string)
			ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
			ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
			w := webdavtools.NewWebdavClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			err = w.DownloadAllFilesToFolder()
			if err != nil {
				return nil, err
			}
		default:
			panic("IFSS类型错误")
		}
		filePathList, fileNameList, _ := filetools.GenerateSpecFilePathNameListFromFolder(ifssDownloadDir)
		for i, filePath := range filePathList {
			unzipFolderDir := filepath.Join(ifssDownloadDir, fileNameList[i] + "_ziptemp")
			err = filetools.Mkdir(unzipFolderDir)
			if err != nil {
				return nil, err
			}
			_, err = ziptools.UnzipFile(filePath, unzipFolderDir)
			if err != nil {
				return nil, err
			}
			// 找到配置文件,并解出接收方是谁
			specFilePath := filepath.Join(unzipFolderDir, fileNameList[i])
			infoFilePath := filepath.Join(unzipFolderDir, fileNameList[i]+"_")
			encryptedReceiverNameBytes, err := filetools.ReadFile(infoFilePath)
			if err != nil {
				return nil, err
			}
			userPrivateKey, err := rsatools.ReadPrivateKeyFile(userPrivateKeyPath)
			if err != nil {
				return nil, err
			}
			receiverNameBytes, err := rsatools.DecryptWithPrivateKey(encryptedReceiverNameBytes, userPrivateKey)
			if err != nil {
				return nil, err
			}
			// 数据交换文件最终存储的文件夹位置
			saveDir = filepath.Join(receiveDir, string(receiverNameBytes)) // 如果是接入节点,使用用户名命名数据交换文件文件夹
			saveDirListSet[saveDir] = member	
			err = filetools.Rename(specFilePath, filepath.Join(saveDir, fileNameList[i]))
			if err != nil {
				return saveDirList, err
			}			
			err = filetools.RmDir(unzipFolderDir)
			if err != nil {
				return saveDirList, err
			}
		}
	}
	for key := range saveDirListSet {
		saveDirList = append(saveDirList, key) // 利用set去重
	}
	return saveDirList, err
}

// 在通信完成时,删除IFSS和本地目录中的所有的数据交换文件,以销毁通信记录
func CleanIFSS(ownAccountListJsonParser *jsontools.JsonParser, receiveDir string) error {
	var err error
	for _, children := range ownAccountListJsonParser.GetAllChildren("OwnAccountList") {
		ifssName := children.ReadJsonValue("/IFSSName").(string)
		ifssDownloadDir := filepath.Join(receiveDir, ifssName)
		ifssType := children.ReadJsonValue("/IFSSType").(string)
		// 删除在线记录
		switch ifssType {
		case "git":
			ifssURL := children.ReadJsonValue("/IFSSURL").(string)
			ifssUserName := children.ReadJsonValue("/IFSSUserName").(string)
			ifssPassword := children.ReadJsonValue("/IFSSUserPassword").(string)
			g := gittools.NewGitClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			err = g.CleanRepository()
			if err != nil {
				return err
			}
		case "webdav":
			ifssURL := children.ReadJsonValue("/IFSSURL").(string)
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
		// 删除本地记录
		err = filetools.RmDir(ifssDownloadDir)
	}
	return err
}

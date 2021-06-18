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
func UploadToIFSS(sendFolderDir string, neighborJsonParser *jsontools.JsonParser) error {
	var err error
	_, receiverName := filepath.Split(sendFolderDir)
	neighborPublicKeyString := neighborJsonParser.ReadJsonValue("/PublicKey").(string)
	neighborPublicKey, err := rsatools.StringToPublicKey(neighborPublicKeyString)
	receiverAccountParserList := neighborJsonParser.GetAllChildren("OwnAccountList")
	filePathList, _, _ := filetools.GenerateSpecFilePathNameListFromFolder(sendFolderDir)
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
			err = g.PushToRepository()
			if err != nil {
				return
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(ifssURL, ifssFolderDir, ifssUserName, ifssPassword)
			err = w.UploadAllFilesFromFolder()
			if err != nil {
				return
			}
		default:
			panic("IFSS类型错误")
		}
	}
	var wg sync.WaitGroup // 信号量
	wg.Add(len(filePathGroup))
	for i, filePathList := range filePathGroup {	
		children := receiverAccountParserList[i] // 被选中的IFSS平台
		go uploadGoroutine(&wg, children, filePathList) // 将list中的所有文件打包后上传到这个平台
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
	var voidMember void
	saveDirListSet := make(map[string]void) // 为了去重
	var saveDirList []string
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
			err = g.CloneRepository()
			if err != nil {
				return
			}
		case "webdav":
			w := webdavtools.NewWebdavClient(ifssURL, ifssDownloadDir, ifssUserName, ifssPassword)
			err = w.DownloadAllFilesToFolder()
			if err != nil {
				return
			}
		default:
			panic("IFSS类型错误")
		}
		if !filetools.IsPathExists(ifssDownloadDir) { // 如果没有检测到下载下来了新内容
			fmt.Println("没有在", ifssURL, "中检测到需要下载的内容")
			return
		}
		filePathList, _, _ := filetools.GenerateSpecFilePathNameListFromFolder(ifssDownloadDir)
		for _, filePath := range filePathList {
			_, fileName := filepath.Split(filePath)
			unzipFolderDir := filepath.Join(ifssDownloadDir, fileName + "_ziptemp")
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
			userPrivateKey, err := rsatools.ReadPrivateKeyFile(userPrivateKeyPath)
			if err != nil {
				return
			}
			receiverNameBytes, err := rsatools.DecryptWithPrivateKey(encryptedReceiverNameBytes, userPrivateKey)
			if err != nil {
				return
			}
			// 数据交换文件最终存储的文件夹位置
			saveDir = filepath.Join(receiveDir, string(receiverNameBytes)) // 如果是接入节点,使用用户名命名数据交换文件文件夹
			saveDirListSet[saveDir] = voidMember	
			err = filetools.Rename(specFilePath, filepath.Join(saveDir, fileName))
			if err != nil {
				return
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

// 在通信完成时,删除IFSS和本地目录中的所有的数据交换文件,以销毁通信记录
func CleanIFSS(ownAccountListJsonParser *jsontools.JsonParser, receiveDir string) error {
	var err error
	for _, children := range ownAccountListJsonParser.GetAllChildren("OwnAccountList") {
		ifssName := children.ReadJsonValue("/IFSSName").(string)
		ifssDownloadDir := filepath.Join(receiveDir, ifssName)
		ifssType := children.ReadJsonValue("/IFSSType").(string)
		ifssURL := children.ReadJsonValue("/IFSSURL").(string)
		if !filetools.IsPathExists(ifssDownloadDir) { // 如果没有检测到下载下来了新内容
			return nil
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
		fmt.Println("已经成功清除在线仓库", ifssURL, "中的内容")
		// 删除本地记录
		err = filetools.RmDir(ifssDownloadDir)
	}
	return err
}

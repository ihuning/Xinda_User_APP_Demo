package ifsstools

type IFSSType int

// 不同的IFSS通道
const (
	jianguoyun IFSSType = 0
	git        IFSSType = 1
)

// 测试IFSS连接
func IFSSConnectionTest(ifssType IFSSType) error {
	var err error
	return err
}

// 上传文件夹中的数据交换文件到IFSS
func UploadFileToIFSS(ifssType IFSSType, uploadDir string) error {
	var err error
	return err
}

// 从IFSS下载数据交换文件到文件夹中
func DownloadFileFromIFSS(ifssType IFSSType, saveDir string) error {
	var err error
	return err
}

// 在通信完成时,删除IFSS中的所有的数据交换文件,销毁通信记录
func CleanIFSS(ifssType IFSSType) error {
	var err error
	return err
}

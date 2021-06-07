package emailtools

import (
	"fmt"
	"mime"
	"net/smtp"
	"path/filepath"
	"xindauserbackground/src/crypto/rsatools"
	"runtime/debug"
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	// "github.com/emersion/go-message/textproto"
	// "log"
	"xindauserbackground/src/filetools"
	// "encoding/base64"
	"github.com/axgle/mahonia"
	"github.com/emersion/go-imap"
	ImapClient "github.com/emersion/go-imap/client"

	// "github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/jordan-wright/email"
)

type SMTPClient struct {
	EmailAddr  string
	SMTPServer string
	SMTPAuth   smtp.Auth
}

type IMAPClient struct {
	IMAPClient *ImapClient.Client
}

// 建立和SMTP的连接
func ConnectToSMTPServer(smtpServer, emailAddr, password string) (*SMTPClient, error) {
	// PlainAuth 身份认证机制 第一个参数通常为空，第二个是发送方邮箱，第三个是发送方密码/密钥，第四个是发送发邮件服务器地址 此处不包括端口号
	smtpAuth := smtp.PlainAuth("", emailAddr, password, strings.Split(smtpServer, ":")[0])
	smtpClient := &SMTPClient{emailAddr, smtpServer, smtpAuth}
	return smtpClient, nil
}

// 建立和IMAP的连接
func ConnectToIMAPServer(imapServer, emailAddr, password string) (*IMAPClient, error) {
	imapClient, err := ImapClient.DialTLS(imapServer, nil)
	if err != nil {
		fmt.Println("无法与邮箱IMAP服务器建立TLS连接")
		return nil, err
	}
	err = imapClient.Login(emailAddr, password)
	if err != nil {
		fmt.Println("无法登录邮箱IMAP服务器")
		return nil, err
	}
	return &IMAPClient{imapClient}, nil
}

// 断开和IMAP的连接
func (c *IMAPClient) Close() error {
	return c.IMAPClient.Logout()
}

// 发送一封带主题和附件的邮件
func (c *SMTPClient) SendEmail(receiverAddr, text, attachmentPath string) error {
	var err error
	//新建一封邮件
	e := email.NewEmail()
	e.From = c.EmailAddr
	e.To = []string{receiverAddr}
	_, subject := filepath.Split(attachmentPath)
	e.Subject = subject
	e.Text = []byte(text)
	// e.HTML = []byte("<h1>Fancy HTML is supported, too!</h1>")
	_, err = e.AttachFile(attachmentPath)
	if err != nil {
		fmt.Println("无法为邮件添加附件", err)
		return err
	}
	//PlainAuth 身份认证机制 第一个参数通常为空，第二个是发送方邮箱，第三个是发送方密码/密钥，第四个是发送发邮件服务器地址 此处不包括端口号
	err = e.Send(c.SMTPServer, c.SMTPAuth)
	if err != nil {
		fmt.Println("无法发送邮件", err)
		return err
	}
	fmt.Println(c.EmailAddr, "成功发送邮件", subject, "给收件人", receiverAddr)
	return nil
}

// 输出当前邮箱中所有的boxs
func (c *IMAPClient) PrintBoxList() {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.IMAPClient.List("", "*", mailboxes)
	}()
	fmt.Println("Mailboxes:")
	for m := range mailboxes {
		fmt.Println("* " + m.Name)
	}
}

// 获取当前邮箱收件箱中完整的邮件列表
func (c *IMAPClient) GetEmailList(boxType string) (*imap.SeqSet, error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("got error: ", err)
			debug.PrintStack()
		}
	}()
	var err error
	// 选择收件箱
	mbox, err := c.IMAPClient.Select(boxType, false)
	if err != nil {
		fmt.Println("无法选择该信箱", err)
		return nil, err
	}
	emailList := new(imap.SeqSet)
	emailList.AddRange(1, mbox.Messages)
	return emailList, err
}

// 接收邮件列表中的所有邮件,并保存附件
func (c *IMAPClient) ReceiveEmail(userPrivateKeyPath string, emailList *imap.SeqSet, saveDir string) error {
	var err error
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("got error: ", err)
			debug.PrintStack()
		}
	}()
	// 私钥用来解密该邮件的最终接收方是谁
	userPrivateKey, err := rsatools.ReadPrivateKeyFile(userPrivateKeyPath)
	if err != nil {
		return err
	}
	// 获取邮件的message body
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}
	messages := make(chan *imap.Message, emailList.Set[0].Stop)
	done := make(chan error, emailList.Set[0].Stop)
	go func() {
		done <- c.IMAPClient.Fetch(emailList, items, messages)
	}()
	for msg := range messages {
		r := msg.GetBody(&section)
		// 创建一个mail reader
		mr, err := mail.CreateReader(r)
		if err != nil {
			fmt.Println("无法创建mail reader", err)
			return err
		}
		var receiverName, fileName string
		var fileContent []byte
		// 输出邮件头的信息
		// header := mr.Header
		// if date, err := header.Date(); err == nil {
		// 	fmt.Println("Date:", date)
		// }
		// if from, err := header.AddressList("From"); err == nil {
		// 	fmt.Println("From:", from)
		// }
		// if to, err := header.AddressList("To"); err == nil {
		// 	fmt.Println("To:", to)
		// }
		// if subject, err := header.Subject(); err == nil {
		// 	fmt.Println("Subject:", subject)
		// }
		// 遍历MIME结构
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("无法遍历MIME", err)
			}
			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				// 邮件正文(plain-text or HTML)
				body, _ := ioutil.ReadAll(p.Body)
				bodyHexBytes, err := rsatools.HexStringToBytes(string(body))
				if err != nil {
					return err
				}
				decryptedReceiverNameBytes, err := rsatools.DecryptWithPrivateKey(bodyHexBytes, userPrivateKey)
				if err != nil {
					return err
				}
				receiverName = string(decryptedReceiverNameBytes)
			case *mail.AttachmentHeader:
				// 附件
				fileName, err = h.Filename()
				if err != nil {
					fmt.Println("无法读取附件的文件名", err)
					return err
				}
				fileContent, err = ioutil.ReadAll(p.Body)
				if err != nil {
					fmt.Println("无法读取附件的内容", err)
					return err
				}
			}
		}
		// 保存文件
		filePath := filepath.Join(saveDir, receiverName, fileName)
		err = filetools.WriteFile(filePath, fileContent, 0777)
		if err != nil {
			return err
		}
	}
	return err
}

// 删除邮件列表中的所有邮件
func (c *IMAPClient) DeleteEmail(emailList *imap.SeqSet) error {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("got error: ", err)
			debug.PrintStack()
		}
	}()
	// defer c.Close()
	var err error
	// 如果本来就为空
	if emailList.Set[0].Stop == 0 {
		return err
	}
	// 先给邮件置删除标志位
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	err = c.IMAPClient.Store(emailList, item, flags, nil)
	if err != nil {
		fmt.Println("无法为邮件添加删除标志", err)
		return err
	}
	// 应用删除操作
	err = c.IMAPClient.Expunge(nil)
	if err != nil {
		fmt.Println("无法执行删除操作", err)
		return err
	}
	return err
}

// 解码邮件头
func decoder() (dec *mime.WordDecoder) {
	dec = new(mime.WordDecoder)
	dec.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "gb2312":
			content, err := ioutil.ReadAll(input)
			if err != nil {
				return nil, err
			}
			utf8str := convertToString(string(content), "gbk", "utf-8")
			t := bytes.NewReader([]byte(utf8str))
			return t, nil
		case "gb18030":
			content, err := ioutil.ReadAll(input)
			if err != nil {
				return nil, err
			}

			utf8str := convertToString(string(content), "gbk", "utf-8")
			t := bytes.NewReader([]byte(utf8str))

			return t, nil

		case "gbk":
			content, err := ioutil.ReadAll(input)
			if err != nil {
				return nil, err
			}

			utf8str := convertToString(string(content), "gbk", "utf-8")
			t := bytes.NewReader([]byte(utf8str))

			return t, nil
		default:
			return nil, fmt.Errorf("unhandle charset:%s", charset)

		}
	}
	return dec
}

// 将字符串转为utf-8编码
func convertToString(src string, srcCode string, tagCode string) string {
	srcCoder := mahonia.NewDecoder(srcCode)
	srcResult := srcCoder.ConvertString(src)
	tagCoder := mahonia.NewDecoder(tagCode)
	_, cdata, _ := tagCoder.Translate([]byte(srcResult), true)
	result := string(cdata)
	return result
}

// 判断byte是否为gbk 编码
func isGBK(data []byte) bool {
	length := len(data)
	var i int = 0
	for i < length {
		if data[i] <= 0xff { //编码小于等于127,只有一个字节的编码，兼容ASCII码
			i++
			continue
		} else { //大于127的使用双字节编码
			if data[i] >= 0x81 &&
				data[i] <= 0xfe &&
				data[i+1] >= 0x40 &&
				data[i+1] <= 0xfe &&
				data[i+1] != 0xf7 {
				i += 2
				continue
			} else {
				return false
			}
		}
	}
	return true
}

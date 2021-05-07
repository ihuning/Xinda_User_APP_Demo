package emailtools

import (
	"fmt"
	// "time"
	"io"
	"io/ioutil"
	// "github.com/emersion/go-message/textproto"
	// "log"
	// "xindauserbackground/src/filetools"
	// "encoding/base64"
	"github.com/emersion/go-imap"
	IMAPClient "github.com/emersion/go-imap/client"
	// "github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

type Client struct {
	Server     string
	UserName   string
	Password   string
	SMTPClient string
	IMAPClient *IMAPClient.Client
}

// 登录SMTP Server和IMAP Server
func ConnectToIMAPServer(server, userName, password string) (*Client, error) {
	imapClient, err := IMAPClient.DialTLS(server, nil)
	if err != nil {
		fmt.Println("无法与邮箱IMAP服务器建立TLS连接")
		return nil, err
	}
	err = imapClient.Login(userName, password)
	if err != nil {
		fmt.Println("无法登录邮箱IMAP服务器")
		return nil, err
	}
	client := &Client{
		Server:     server,
		UserName:   userName,
		Password:   password,
		SMTPClient: "测试",
		IMAPClient: imapClient,
	}
	return client, nil
}

// 断开连接
func (c *Client) Close() error {
	return c.IMAPClient.Close()
}

// 获取当前邮箱的邮件列表
func (c *Client) GetEmailList(receiverAddr, attachmentPath string) ([]string, error) {
	var err error
	return nil, err
}

// 发送一封带附件的邮件
func (c *Client) SendEmail(receiverAddr, attachmentPath string) error {
	var err error
	return err
}

// 接收最近一封邮件,并保存附件
func (c *Client) ReceiveEmail(attachmentPath string) error {
	mbox, err := c.IMAPClient.Select("INBOX", false)
	if err != nil {
		fmt.Println(err)
	}
	// 获取最后一条邮件
	if mbox.Messages == 0 {
		fmt.Println("No message in mailbox")
	}
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(mbox.Messages)
	// 获取邮件的message body
	var section imap.BodySectionName
	items := []imap.FetchItem{section.FetchItem()}
	messages := make(chan *imap.Message, 1)
	go func() {
		if err := c.IMAPClient.Fetch(seqSet, items, messages); err != nil {
			fmt.Println(err)
		}
	}()
	msg := <-messages
	if msg == nil {
		fmt.Println("Server didn't returned message")
	}
	r := msg.GetBody(&section)
	if r == nil {
		fmt.Println("Server didn't returned message body")
	}
	// 创建一个mail reader
	mr, err := mail.CreateReader(r)
	if err != nil {
		fmt.Println(err)
	}
	// 输出邮件头的信息
	header := mr.Header
	if date, err := header.Date(); err == nil {
		fmt.Println("Date:", date)
	}
	if from, err := header.AddressList("From"); err == nil {
		fmt.Println("From:", from)
	}
	if to, err := header.AddressList("To"); err == nil {
		fmt.Println("To:", to)
	}
	if subject, err := header.Subject(); err == nil {
		fmt.Println("Subject:", subject)
	}
	// 遍历MIME结构
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println(err)
		}
		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// 邮件正文(plain-text or HTML)
			b, _ := ioutil.ReadAll(p.Body)
			fmt.Println("Got text:", string(b))
		case *mail.AttachmentHeader:
			fmt.Println("Got attachment==========")
			// 附件
			filename, _ := h.Filename()
			fmt.Println("Got attachment:", filename)
			b, errp := ioutil.ReadAll(p.Body)
			fmt.Println("errp ===== :", errp)
			err := ioutil.WriteFile(attachmentPath, b, 0777)
			if err != nil {
				fmt.Println("attachment err:", err)
			}
		}
	}
	return err
}

// 删除一封邮件
func (c *Client) DeleteEmail(attachmentPath string) error {
	var err error
	return err
}

// 对称加解密方法.
//  对称密钥和防重放攻击的12位Nonce写在数据交换文件的头部之后,使用接收方公钥加密.
package aestools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// 生成256位的AES key和12字节的nonce
func InitAES() ([]byte, []byte, error) {
	aesKey := make([]byte, 256/8)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		fmt.Println("无法生成AES Key", err)
		return nil, nil, err
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		fmt.Println("无法生成nonce", err)
		return nil, nil, err
	}
	return aesKey, nonce, nil
}

// aes-256-gcm 加密
func EncryptWithAES(aesKey []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		fmt.Println("无法生成AES block", err)
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println("无法生成AEAD对象", err)
		return nil, err
	}
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, err

}

// aes-256-gcm 解密
func DecryptWithAES(aesKey []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		fmt.Println("无法生成AES block", err)
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		fmt.Println("无法生成AEAD对象", err)
		return nil, err
	}
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		fmt.Println("无法使用AES解密", err)
		return nil, err
	}
	return plaintext, err
}

// 获得使用aes-256-gcm加密后的密文长度
func GetCiphertextLength(plaintextLength int) int {
	return plaintextLength + 16
}

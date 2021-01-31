package aestools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// 生成256位的AES key和12位的nonce
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

package rsatools

import (
	"bytes"
	"math"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"../../filetools"
)

var FilePermMode = os.FileMode(0777) // Default file permission

// 生成RSA密钥对
func GenerateKeyPair(bits int) (*rsa.PublicKey, *rsa.PrivateKey, error) {
	var err error
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		fmt.Println("无法生成RSA密钥对", err)
		return nil, nil, err
	}
	return &priv.PublicKey, priv, err
}

// 将生成的私钥转为bytes
func privateKeyToBytes(priv *rsa.PrivateKey) ([]byte, error) {
	privBytes := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)
	return privBytes, nil
}

// 将生成的公钥转为bytes
func publicKeyToBytes(pub *rsa.PublicKey) ([]byte, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		fmt.Println("无法将生成的公钥转为bytes", err)
		return nil, err
	}
	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubASN1,
	})
	return pubBytes, err
}

// 将bytes转为私钥
func bytesToPrivateKey(priv []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(priv)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		fmt.Println("is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			fmt.Println("无法解析RSA私钥", err)
			return nil, err
		}
	}
	key, err := x509.ParsePKCS1PrivateKey(b)
	if err != nil {
		fmt.Println("无法解析RSA私钥", err)
		return nil, err
	}
	return key, nil
}

// 将bytes转为公钥
func bytesToPublicKey(pub []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pub)
	enc := x509.IsEncryptedPEMBlock(block)
	b := block.Bytes
	var err error
	if enc {
		fmt.Println("is encrypted pem block")
		b, err = x509.DecryptPEMBlock(block, nil)
		if err != nil {
			fmt.Println("无法解析RSA公钥", err)
			return nil, err
		}
	}
	ifc, err := x509.ParsePKIXPublicKey(b)
	if err != nil {
		fmt.Println("无法解析RSA公钥", err)
		return nil, err
	}
	key, ok := ifc.(*rsa.PublicKey)
	if !ok {
		fmt.Println("not ok")
	}
	return key, nil
}

// 生成密钥对文件
func GenerateKeyPairFiles(bits int, publicKeyFilePath string, privateKeyFilePath string) error {
	var pub *rsa.PublicKey = &rsa.PublicKey{}
	var priv *rsa.PrivateKey = &rsa.PrivateKey{}
	// 生成密钥对
	pub, priv, err := GenerateKeyPair(bits)
	if err != nil {
		fmt.Println("无法生成密钥对", err)
		return err
	}
	publicKeyBytes, err := publicKeyToBytes(pub)
	if err != nil {
		fmt.Println("无法将公钥转为字节流", err)
		return err
	}
	privateKeyBytes, err := privateKeyToBytes(priv)
	if err != nil {
		fmt.Println("无法将私钥转为字节流", err)
		return err
	}
	filetools.WriteFile(publicKeyFilePath, publicKeyBytes, FilePermMode)
	if err != nil {
		fmt.Println("无法将公钥写入文件", err)
		return err
	}
	filetools.WriteFile(privateKeyFilePath, privateKeyBytes, FilePermMode)
	if err != nil {
		fmt.Println("无法将私钥写入文件", err)
		fmt.Println(err)
		return err
	}
	return nil
}

// 读取公钥文件
func ReadPublicKeyFile(publicKeyFilePath string) (*rsa.PublicKey, error) {
	var pub *rsa.PublicKey = &rsa.PublicKey{}
	publicKeyBytes, err := filetools.ReadFile(publicKeyFilePath)
	if err != nil {
		fmt.Println("无法读取公钥文件", err)
		return nil, err
	}
	pub, err = bytesToPublicKey(publicKeyBytes)
	if err != nil {
		fmt.Println("无法将公钥文件转为字节流", err)
		return nil, err
	}
	return pub, nil
}

// 读取私钥文件
func ReadPrivateKeyFile(privateKeyFilePath string) (*rsa.PrivateKey, error) {
	var priv *rsa.PrivateKey = &rsa.PrivateKey{}
	privateKeyBytes, err := filetools.ReadFile(privateKeyFilePath)
	if err != nil {
		fmt.Println("无法读取私钥文件", err)
		return nil, err
	}
	priv, err = bytesToPrivateKey(privateKeyBytes)
	if err != nil {
		fmt.Println("无法将私钥文件转为字节流", err)
		return nil, err
	}
	return priv, nil
}

// 将过长的数据分隔成片,避免RSA无法对过长数据进行加解密
func split(buf []byte, lim int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}

// 将数据用公钥加密
func EncryptWithPublicKey(plaintext []byte, pub *rsa.PublicKey) ([]byte, error) {
	partLen := pub.N.BitLen()/8 - 11
	chunks := split([]byte(plaintext), partLen)
	buffer := bytes.NewBufferString("")
	for _, chunk := range chunks {
		bytes, err := rsa.EncryptPKCS1v15(rand.Reader, pub, chunk)
		if err != nil {
			fmt.Println("无法将数据用RSA公钥加密", err)
			return nil, err
		}
		buffer.Write(bytes)
	}
	ciphertext := buffer.Bytes()
	return ciphertext, nil
}

// 将数据用私钥解密
func DecryptWithPrivateKey(ciphertext []byte, priv *rsa.PrivateKey) ([]byte, error) {
	partLen := priv.N.BitLen() / 8
	chunks := split([]byte(ciphertext), partLen)
	buffer := bytes.NewBufferString("")
	for _, chunk := range chunks {
		decrypted, err := rsa.DecryptPKCS1v15(rand.Reader, priv, chunk)
		if err != nil {
			fmt.Println("无法将数据用RSA私钥解密", err)
			return nil, err
		}
		buffer.Write(decrypted)
	}
	plaintext := buffer.Bytes()
	return plaintext, nil
}

// 数据加签
func Sign(data []byte, priv *rsa.PrivateKey) ([]byte, error) {
	h := crypto.SHA256.New()
	h.Write(data)
	hashed := h.Sum(nil)
	sign, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed)
	if err != nil {
		fmt.Println("无法将数据用RSA私钥签名", err)
		return nil, err
	}
	// return base64.RawURLEncoding.EncodeToString(sign), err
	return sign, err
}

// 数据验签
func Verify(data []byte, sign []byte, pub *rsa.PublicKey) error {
	h := crypto.SHA256.New()
	_, err := h.Write(data)
	if err != nil {
		fmt.Println("无法将数据用RSA公钥验签", err)
		return err
	}
	hashed := h.Sum(nil)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed, sign)
}

// 根据明文长度计算出密文长度
func GetCiphertextLength(plaintextLength int) int {
	var ciphertextLength = math.Ceil(float64(plaintextLength) / 117) * 128
	return int(ciphertextLength)
}

package padding

import (
	"crypto/rand"
	"io"
	"math/big"
)

// 生成随机长度的填充字节流
func GeneratePadding(min, max int) []byte {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min)))
	paddingLength := int(n.Int64()) + min
	padding := make([]byte, paddingLength)
	io.ReadFull(rand.Reader, padding)
	return padding
}

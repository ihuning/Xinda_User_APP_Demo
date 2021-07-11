// 生成一段定长的随机bytes.
package padding

import (
	"crypto/rand"
	"io"
	"math/big"
)

// 生成随机长度的填充字节流
func GeneratePadding(max int) []byte {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	paddingLength := int(n.Int64())
	padding := make([]byte, paddingLength)
	io.ReadFull(rand.Reader, padding)
	return padding
}

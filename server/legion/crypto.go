package legion

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"math"
	"os"
)

// GetCryptoMd5Hash ...
func GetCryptoMd5Hash(fqp string) (string, error) {
	const chunk = 8192 // 8KB

	f, err := os.Open(fqp)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fInfo, _ := f.Stat()
	size := fInfo.Size()

	blocks := uint64(math.Ceil(float64(size) / float64(chunk)))

	hash := md5.New()

	for i := uint64(0); i < blocks; i++ {
		blocksize := int(math.Min(chunk, float64(size-int64(i*chunk))))
		buf := make([]byte, blocksize)

		f.Read(buf)
		io.WriteString(hash, string(buf)) // append into the hash
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

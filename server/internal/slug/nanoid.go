package slug

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "_-0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const size = 10

func generateID() (string, error) {
	b := make([]byte, size)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}

func GenerateFileSlug() (string, error) {
	id, err := generateID()
	if err != nil {
		return "", err
	}
	return "f~" + id, nil
}

func GenerateShareSlug() (string, error) {
	id, err := generateID()
	if err != nil {
		return "", err
	}
	return "s~" + id, nil
}
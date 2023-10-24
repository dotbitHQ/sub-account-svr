package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
)

func AesEncrypt(data, key string) (string, error) {
	res, err := aesEncrypt([]byte(data), []byte(key))
	if err != nil {
		return "", fmt.Errorf("aesEncrypt:%s", err.Error())
	}
	return base64.StdEncoding.EncodeToString(res), nil
}

func AesDecrypt(data, key string) (string, error) {
	encBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("base64.StdEncoding.DecodeString:%s", err.Error())
	}
	if len(encBytes)%aes.BlockSize != 0 {
		return "", fmt.Errorf("encBytes len err")
	}
	res, err := aesDecrypt(encBytes, []byte(key))
	if err != nil {
		return "", fmt.Errorf("aesDecrypt:%s", err.Error())
	}
	return string(res), nil
}

func aesEncrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	data = pKCS5Padding(data, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(data))
	blockMode.CryptBlocks(crypted, data)
	return crypted, nil
}

func aesDecrypt(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(data))
	blockMode.CryptBlocks(origData, data)
	origData, err = pKCS5UnPadding(origData)
	if err != nil {
		return origData, err
	}
	return origData, nil
}

func pKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func pKCS5UnPadding(origData []byte) ([]byte, error) {
	length := len(origData)
	if length < 1 {
		return []byte{}, errors.New("pKCS5UnPadding len(origData) err")
	}
	unpadding := int(origData[length-1])
	tarLen := length - unpadding
	if length < tarLen || tarLen < 0 {
		return []byte{}, errors.New("pKCS5UnPadding tarLen err")
	}
	return origData[:tarLen], nil
}

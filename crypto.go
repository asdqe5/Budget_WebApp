// 프로젝트 결산 프로그램
//
// Description : 암호화 관련 스크립트

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"io/ioutil"
	"net/mail"
	"os"
	"os/user"
	"path"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// keyFilePathFunc 함수는 key 파일 경로를 반환하는 함수이다.
func keyFilePathFunc() (string, error) {
	// .key 파일 경로
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}

	keyFilePath := user.HomeDir + "/.budget/" + hostname + "_private.key"

	return keyFilePath, nil
}

// encryptFunc 함수는 문자를 입력받아 해쉬문자로 변환하는 함수이다.
func encryptFunc(s string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// checkAESKeyFileFunc 함수는 key 파일이 존재하는지 확인하는 함수이다.
func checkAESKeyFileFunc() (bool, error) {
	keyFilePath, err := keyFilePathFunc()
	if err != nil {
		return false, err
	}

	existed, err := checkFileExistsFunc(keyFilePath)
	if err != nil {
		return false, err
	}

	if existed {
		return true, nil
	} else {
		return false, nil
	}
}

// genAESKEYFileFunc 함수는 key를 생성하여 .key 파일에 저장하는 함수이다.
func genAESKEYFileFunc() error {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return err
	}

	block := &pem.Block{
		Type:  "AES KEY",
		Bytes: key,
	}

	// .key 파일 경로
	keyFilePath, err := keyFilePathFunc()
	if err != nil {
		return err
	}

	// .key 파일 경로에 있는 파일들 삭제
	err = delAllFilesFunc(path.Dir(keyFilePath))
	if err != nil {
		if os.IsNotExist(err) {
			err = createFolderFunc(path.Dir(keyFilePath))
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// .key 파일에 key 저장
	err = ioutil.WriteFile(keyFilePath, pem.EncodeToMemory(block), 0644)
	if err != nil {
		return err
	}
	return nil
}

// readKEYFileFunc 함수는 .key 파일에서 키를 가져오는 함수이다.
func readKEYFileFunc() ([]byte, error) {
	// .key 파일 경로
	keyFilePath, err := keyFilePathFunc()
	if err != nil {
		return nil, err
	}

	key, err := ioutil.ReadFile(keyFilePath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(key)
	return block.Bytes, nil
}

// encryptAES256Func 함수는 문자열을 입력받아 AES 256 암호화 기법으로 암호화하는 함수이다.
func encryptAES256Func(s string) (string, error) {
	key, err := readKEYFileFunc()
	if err != nil {
		return "", err
	}

	// mac 주소 가져오기
	// mac, err := serviceMACAddrFunc() // mac := "52:54:00:df:6a:e9"
	// if err != nil {
	// 	return "", err
	// }
	mac := "52:54:00:df:6a:e9" // 10.20.30.192 MAC address 고정(MAIN)
	// mac := "b4:2e:99:6e:a1:07"        // 10.20.31.160 MAC address 애림(TEST)
	iv := []byte(mac[:aes.BlockSize]) // iv의 크기는 AES의 block 크기와 같아야한다(aes.BlockSize = 128 bit = 16 bytes)

	bplainText := PKCS5Padding([]byte(s), aes.BlockSize)

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}

	cipherText := make([]byte, len(bplainText))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText, bplainText)

	return hex.EncodeToString(cipherText), nil
}

// decryptAES256Func 함수는 AES 256 암호화 기법으로 암호화된 문자열을 복호화하는 함수이다.
func decryptAES256Func(cipherText string) (string, error) {
	if cipherText == "" {
		return "", nil
	}

	key, err := readKEYFileFunc()
	if err != nil {
		return "", err
	}

	// mac 주소 가져오기
	// mac, err := serviceMACAddrFunc() // mac := "52:54:00:df:6a:e9"
	// if err != nil {
	// 	return "", err
	// }
	mac := "52:54:00:df:6a:e9" // 10.20.30.192 MAC address 고정(MAIN)
	// mac := "b4:2e:99:6e:a1:07"        // 10.20.31.160 MAC address 애림(TEST)
	iv := []byte(mac[:aes.BlockSize]) // iv의 크기는 AES의 block 크기와 같아야한다(aes.BlockSize = 128 bit = 16 bytes)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	bcipherText, _ := hex.DecodeString(cipherText)

	mode := cipher.NewCBCDecrypter(block, iv)
	orig := make([]byte, len(bcipherText))
	mode.CryptBlocks(orig, bcipherText)

	orig = PKCS5UnPadding(orig)

	return string(orig), nil
}

func PKCS5Padding(cipherText []byte, blockSize int) []byte {
	padding := (blockSize - len(cipherText)%blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(cipherText, padText...)
}

func PKCS5UnPadding(orig []byte) []byte {
	length := len(orig)
	unpadding := int(orig[length-1])
	return orig[:(length - unpadding)]
}

// encodeRFC2047Func 함수는 메일 제목 변경하는 함수이다.
func encodeRFC2047Func(s string) string {
	addr := mail.Address{s, ""}
	return strings.Trim(addr.String(), "<@>")
}

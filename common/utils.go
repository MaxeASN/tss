package common

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"
)

func Encrypt(passphrase, channelId, moniker, id string) ([]byte, error) {
	text := []byte(fmt.Sprintf("%s@%s@%s", channelId, moniker, id))
	key := sha256.Sum256([]byte(passphrase))

	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key[:])
	// if there are any errors, handle them
	if err != nil {
		return nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		return nil, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// here we encrypt our text using the Seal function
	// Seal encrypts and authenticates plaintext, authenticates the
	// additional data and appends the result to dst, returning the updated
	// slice. The nonce must be NonceSize() bytes long and unique for all
	// time, for a given key.
	return gcm.Seal(nonce, nonce, text, nil), nil
}

func Decrypt(ciphertext []byte, channelId, passphrase string) (moniker, id string, error error) {
	key := sha256.Sum256([]byte(passphrase))
	c, err := aes.NewCipher(key[:])
	if err != nil {
		error = err
		return
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		error = err
		return
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		error = fmt.Errorf("ciphertext is not as long as expected")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		error = err
		return
	}

	res := strings.SplitN(string(plaintext), "@", 3)
	if len(res) != 3 {
		error = fmt.Errorf("wrong format of decrypted plaintext")
		return
	}
	if res[0] != channelId {
		error = fmt.Errorf("wrong channel id of message")
	}
	epochSeconds := ConvertHexToTimestamp(channelId[3:])
	if time.Now().Unix() > int64(epochSeconds) {
		error = fmt.Errorf("password has been expired")
		return
	}
	return res[1], res[2], nil
}

// conversion between hex and int32 epoch seconds
// refer: https://www.epochconverter.com/hex
func ConvertTimestampToHex(timestamp int64) string {
	buf := bytes.Buffer{}
	if err := binary.Write(&buf, binary.BigEndian, int32(timestamp)); err != nil {
		return ""
	}
	return fmt.Sprintf("%X", buf.Bytes())
}

// conversion between hex and int32 epoch seconds
// refer: https://www.epochconverter.com/hex
func ConvertHexToTimestamp(hexTimestamp string) int {
	dst := make([]byte, 8)
	hex.Decode(dst, []byte(hexTimestamp))
	var epochSeconds int32
	if err := binary.Read(bytes.NewReader(dst), binary.BigEndian, &epochSeconds); err != nil {
		return math.MaxInt64
	}
	return int(epochSeconds)
}

func ReplaceIpInAddr(addr, realIp string) string {
	re := regexp.MustCompile(`((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))`)
	return re.ReplaceAllString(addr, realIp)
}

func ConvertMultiAddrStrToNormalAddr(listenAddr string) (string, error) {
	re := regexp.MustCompile(`((([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5]))\/tcp\/([0-9]+)`)
	all := re.FindStringSubmatch(listenAddr)
	if len(all) != 6 {
		return "", fmt.Errorf("failed to convert multiaddr to listen addr")
	}
	return fmt.Sprintf("%s:%s", all[1], all[5]), nil
}

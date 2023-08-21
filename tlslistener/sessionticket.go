package tlslistener

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"io/ioutil"
	"os"
	"time"

	"github.com/getlantern/errors"
)

const (
	keySize = 32

	rotateInterval = 24 * time.Hour
)

// Using on-disk session ticket keys is deprecated, but this is left here for backwards compatibility
// with old infrastructure proxies. Use maintainSessionTicketKeysInMemory instead.
func maintainSessionTicketKeyFile(
	cfg *tls.Config, sessionTicketKeyFile, firstSessionTicketKey string, keyListener func(keys [][keySize]byte)) error {

	log.Debugf("Will rotate session ticket key every %v and store in %v", rotateInterval, sessionTicketKeyFile)

	var firstKey *[keySize]byte
	if firstSessionTicketKey != "" {
		b, err := base64.StdEncoding.DecodeString(firstSessionTicketKey)
		if err != nil {
			return errors.New("failed to parse first session ticket key: %v", err)
		}
		if len(b) != keySize {
			return errors.New("first session ticket key should be keySize bytes")
		}
		firstKey = new([keySize]byte)
		copy(firstKey[:], b)
	}

	// read cached session ticket keys
	keyBytes, err := ioutil.ReadFile(sessionTicketKeyFile)
	if err != nil {
		if !os.IsNotExist(err) {
			panic(errors.New("Unable to read session ticket key file %v: %v", sessionTicketKeyFile, err))
		}
		keyBytes = make([]byte, 0)
	}

	if firstKey != nil {
		keyBytes = ensureFirstKey(*firstKey, keyBytes)
	}

	// Create a new key right away
	keyBytes = prependToSessionTicketKeys(cfg, sessionTicketKeyFile, keyBytes, keyListener)

	go func() {
		for {
			time.Sleep(rotateInterval)
			keyBytes = prependToSessionTicketKeys(cfg, sessionTicketKeyFile, keyBytes, keyListener)
		}
	}()

	return nil
}

// ensureFirstKey ensures that firstKey is the oldest key in keyBytes, where keyBytes represents a
// string of session ticket keys in ascending order by age.
//
// In other words, the last keySize bytes of the result will be equal to firstKey.
func ensureFirstKey(firstKey [keySize]byte, keyBytes []byte) []byte {
	if len(keyBytes) < keySize {
		return append(keyBytes, firstKey[:]...)
	}

	currentFirst := keyBytes[len(keyBytes)-keySize:]
	if bytes.Equal(currentFirst, firstKey[:]) {
		return keyBytes
	}
	return append(keyBytes, firstKey[:]...)
}

func prependToSessionTicketKeys(cfg *tls.Config, sessionTicketKeyFile string, keyBytes []byte, keyListener func(keys [][keySize]byte)) []byte {
	newKey := makeSessionTicketKey()
	keyBytes = append(newKey, keyBytes...)
	saveSessionTicketKeys(sessionTicketKeyFile, keyBytes)

	keys := buildKeysArray(keyBytes)
	cfg.SetSessionTicketKeys(keys)
	keyListener(keys)
	return keyBytes
}

func saveSessionTicketKeys(sessionTicketKeyFile string, keyBytes []byte) {
	err := ioutil.WriteFile(sessionTicketKeyFile, keyBytes, 0644)
	if err != nil {
		panic(errors.New("Unable to save session ticket key bytes to %v: %v", sessionTicketKeyFile, err))
	}
}

func makeSessionTicketKey() []byte {
	b := make([]byte, keySize)
	rand.Read(b)
	return b
}

func maintainSessionTicketKeysInMemory(
	cfg *tls.Config, sessionTicketKeys string, keyListener func(keys [][keySize]byte)) error {

	keyBytes, err := base64.StdEncoding.DecodeString(sessionTicketKeys)
	if err != nil {
		return errors.New("failed to parse session ticket keys: %v", err)
	}
	if len(keyBytes)%keySize != 0 {
		return errors.New("session ticket keys should be multiple of keySize bytes")
	}

	// Initialize key
	keyListener(buildKeysArray(keyBytes))

	if len(keyBytes) == keySize {
		log.Debug("session ticket keys contains only one key, we'll use that and not bother rotating")
		return nil
	}

	log.Debugf("Will rotate %d session ticket keys in memory every %v hours", len(keyBytes)/keySize, rotateInterval)

	go func() {
		for {
			time.Sleep(rotateInterval)
			keyBytes = rotateSessionTicketKeysInMemory(keyBytes)
			keyListener(buildKeysArray(keyBytes))
		}
	}()

	return nil
}

func rotateSessionTicketKeysInMemory(keyBytes []byte) []byte {
	shiftedKeyBytes := make([]byte, len(keyBytes))
	copy(shiftedKeyBytes, keyBytes[keySize:])
	copy(shiftedKeyBytes[len(keyBytes)-keySize:], keyBytes[:keySize])
	keyBytes = shiftedKeyBytes
	return keyBytes
}

func buildKeysArray(keyBytes []byte) [][keySize]byte {
	numKeys := len(keyBytes) / keySize
	keys := make([][keySize]byte, 0, numKeys)
	for i := 0; i < numKeys; i++ {
		currentKeyBytes := keyBytes[i*keySize:]
		var key [keySize]byte
		copy(key[:], currentKeyBytes)
		keys = append(keys, key)
	}
	return keys
}

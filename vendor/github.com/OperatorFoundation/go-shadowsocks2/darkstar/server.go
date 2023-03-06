package darkstar

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/OperatorFoundation/go-shadowsocks2/internal"
	"github.com/aead/ecdh"
)

type DarkStarServer struct {
	serverPersistentPublicKey  crypto.PublicKey
	serverPersistentPrivateKey crypto.PrivateKey
	serverEphemeralPublicKey   crypto.PublicKey
	serverEphemeralPrivateKey  crypto.PrivateKey
	serverIdentifier           []byte
	clientEphemeralPublicKey   crypto.PublicKey
}

func NewDarkStarServer(serverPersistentPrivateKey string, host string, port int) *DarkStarServer {
	privateKey, decodeError := base64.StdEncoding.DecodeString(serverPersistentPrivateKey)
	if decodeError != nil {
		return nil
	}

	keyExchange := ecdh.Generic(elliptic.P256())
	serverIdentifier := getServerIdentifier(host, port)

	serverEphemeralPrivateKey, serverEphemeralPublicKey, keyError := keyExchange.GenerateKey(rand.Reader)
	if keyError != nil {
		return nil
	}

	return &DarkStarServer{
		serverPersistentPublicKey:  keyExchange.PublicKey(privateKey),
		serverPersistentPrivateKey: privateKey,
		serverEphemeralPublicKey:   serverEphemeralPublicKey,
		serverEphemeralPrivateKey:  serverEphemeralPrivateKey,
		serverIdentifier:           serverIdentifier,
	}
}

func (a *DarkStarServer) StreamConn(conn net.Conn) (net.Conn, error) {
	clientEphemeralPublicKeyBuffer := make([]byte, keySize)
	_, keyReadError := conn.Read(clientEphemeralPublicKeyBuffer)
	if keyReadError != nil {
		print("DarkStarServer: Error creating a DarkStar connection: ")
		println(keyReadError)
		return nil, keyReadError // ERROR, this means they never send us anything, probably the connection is closed
	}

	if internal.CheckSalt(clientEphemeralPublicKeyBuffer) {
		return NewBlackHoleConn(), nil
	} else {
		internal.AddSalt(clientEphemeralPublicKeyBuffer)
	}

	a.clientEphemeralPublicKey = BytesToPublicKey(clientEphemeralPublicKeyBuffer)

	clientConfirmationCode := make([]byte, confirmationCodeSize)
	_, confirmationReadError := conn.Read(clientConfirmationCode)
	if confirmationReadError != nil {
		fmt.Println("DarkStarServer: Error creating a DarkStar connection: ", confirmationReadError)
		return nil, confirmationReadError // ERROR, probably the connection is closed
	}

	serverCopyClientConfirmationCode, confirmationError := a.generateClientConfirmationCode()
	if confirmationError != nil {
		fmt.Println("DarkStarServer: BlackholeConnection: ", confirmationError)
		return NewBlackHoleConn(), nil // BLACKHOLE, this means we could not generate the code potentially because we did not receive a valid key
	}

	if !bytes.Equal(clientConfirmationCode, serverCopyClientConfirmationCode) {
		fmt.Println("DarkStarServer : BlackholeConnection: The client confirmation code and the server copy of the client confirmation code are not equal")
		return NewBlackHoleConn(), nil // BLACKHOLE
	}

	serverEphemeralPublicKeyData, pubKeyToBytesError := PublicKeyToBytes(a.serverEphemeralPublicKey)
	if pubKeyToBytesError != nil {
		fmt.Println("DarkStarServer: BlackholeConnection: ", pubKeyToBytesError)
		return NewBlackHoleConn(), nil // BLACKHOLE, this means the bytes they sent us were not a public key, probably a probe
	}

	serverConfirmationCode, _ := a.generateServerConfirmationCode()

	_, keyWriteError := conn.Write(serverEphemeralPublicKeyData)
	if keyWriteError != nil {
		fmt.Println("DarkStarServer: Error creating a DarkStar connection: ", keyWriteError)
		return nil, keyWriteError // ERROR, the client closed the connection
	}

	_, confirmationWriteError := conn.Write(serverConfirmationCode)
	if confirmationWriteError != nil {
		fmt.Println("DarkStarServer: Error creating a DarkStar connection: ", confirmationWriteError)
		return nil, confirmationWriteError // ERROR, the client closed the connection
	}

	sharedKeyServerToClient, sharedKeyServerError := a.createServerToClientSharedKey()
	if sharedKeyServerError != nil {
		fmt.Println("DarkStarServer: BlackholeConnection: ", sharedKeyServerError)
		return NewBlackHoleConn(), nil // BLACKHOLE, not sure why this would happen
	}

	sharedKeyClientToServer, sharedKeyClientError := a.createClientToServerSharedKey()
	if sharedKeyClientError != nil {
		fmt.Println("DarkStarServer: BlackholeConnection: ", sharedKeyClientError)
		return NewBlackHoleConn(), nil // BLACKHOLE, not sure why this would happen
	}

	encryptCipher, encryptKeyError := a.Encrypter(sharedKeyServerToClient)
	if encryptKeyError != nil {
		fmt.Println("DarkStarServer: Error creating a DarkStar connection: ", encryptKeyError)
		return NewBlackHoleConn(), nil // BLACKHOLE, not sure why this would happen
	}

	decryptCipher, decryptKeyError := a.Decrypter(sharedKeyClientToServer)
	if decryptKeyError != nil {
		fmt.Println("DarkStarServer: Error creating a DarkStar connection: ", decryptKeyError)
		return NewBlackHoleConn(), nil // BLACKHOLE, not sure why this would happen
	}

	return NewDarkStarConn(conn, encryptCipher, decryptCipher), nil
}

func (a *DarkStarServer) PacketConn(conn net.PacketConn) net.PacketConn {
	return NewPacketConn(conn, a)
}

func (a *DarkStarServer) KeySize() int {
	return 32
}

func (a *DarkStarServer) SaltSize() int {
	return 96
}

func (a *DarkStarServer) Encrypter(sharedKey []byte) (cipher.AEAD, error) {
	return a.aesGCM(sharedKey)
}

func (a *DarkStarServer) Decrypter(sharedKey []byte) (cipher.AEAD, error) {
	return a.aesGCM(sharedKey)
}

func (a *DarkStarServer) aesGCM(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return cipher.NewGCM(blk)
}

func (a *DarkStarServer) generateSharedKey(personalizationString string) ([]byte, error) {
	p256 := ecdh.Generic(elliptic.P256())
	ephemeralECDHBytes := p256.ComputeSecret(a.serverEphemeralPrivateKey, a.clientEphemeralPublicKey)
	persistentECDHBytes := p256.ComputeSecret(a.serverPersistentPrivateKey, a.clientEphemeralPublicKey)

	clientEphemeralPublicKeyBytes, clientKeyToBytesError := PublicKeyToBytes(a.clientEphemeralPublicKey)
	if clientKeyToBytesError != nil {
		return nil, clientKeyToBytesError
	}

	serverEphemeralPublicKeyBytes, serverKeyToBytesError := PublicKeyToBytes(a.serverEphemeralPublicKey)
	if serverKeyToBytesError != nil {
		return nil, serverKeyToBytesError
	}

	hash := sha256.New()
	hash.Write(ephemeralECDHBytes)
	hash.Write(persistentECDHBytes)
	hash.Write(a.serverIdentifier)
	hash.Write(clientEphemeralPublicKeyBytes)
	hash.Write(serverEphemeralPublicKeyBytes)
	hash.Write([]byte("DarkStar"))
	hash.Write([]byte(personalizationString)) // Destination

	return hash.Sum(nil), nil
}

func (a *DarkStarServer) createServerToClientSharedKey() ([]byte, error) {
	return a.generateSharedKey("client")
}

func (a *DarkStarServer) createClientToServerSharedKey() ([]byte, error) {
	return a.generateSharedKey("server")
}

func (a *DarkStarServer) generateServerConfirmationCode() ([]byte, error) {
	p256 := ecdh.Generic(elliptic.P256())
	ecdhSecret := p256.ComputeSecret(a.serverPersistentPrivateKey, a.clientEphemeralPublicKey)
	serverPersistentPublicKeyData, serverKeyError := PublicKeyToBytes(a.serverPersistentPublicKey)
	if serverKeyError != nil {
		return nil, serverKeyError
	}

	clientEphemeralPublicKeyData, clientKeyError := PublicKeyToBytes(a.clientEphemeralPublicKey)
	if clientKeyError != nil {
		return nil, clientKeyError
	}

	hash := sha256.New()
	hash.Write(ecdhSecret)
	hash.Write(a.serverIdentifier)
	hash.Write(serverPersistentPublicKeyData)
	hash.Write(clientEphemeralPublicKeyData)
	hash.Write([]byte("DarkStar"))
	hash.Write([]byte("server"))

	return hash.Sum(nil), nil
}

func (a *DarkStarServer) generateClientConfirmationCode() (code []byte, codeError error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("Failed to create a ClientConfirmationCode:", err)
			code = nil
			codeError = errors.New("failed to create a ClientConfirmationCode")
		}
	}()

	p256 := ecdh.Generic(elliptic.P256())
	if a.serverPersistentPrivateKey == nil {
		return nil, errors.New("(generateClientConfirmationCode) serverPersistentPrivateKey is nil")
	}

	if a.clientEphemeralPublicKey == nil {
		return nil, errors.New("(generateClientConfirmationCode) clientEphemeralPublicKey is nil")
	}

	ecdhSecret := p256.ComputeSecret(a.serverPersistentPrivateKey, a.clientEphemeralPublicKey)

	serverPersistentPublicKeyData, serverKeyError := PublicKeyToBytes(a.serverPersistentPublicKey)
	if serverKeyError != nil {
		return nil, serverKeyError
	}

	clientEphemeralPublicKeyData, clientKeyError := PublicKeyToBytes(a.clientEphemeralPublicKey)
	if clientKeyError != nil {
		return nil, clientKeyError
	}

	hash := sha256.New()
	hash.Write(ecdhSecret)
	hash.Write(a.serverIdentifier)
	hash.Write(serverPersistentPublicKeyData)
	hash.Write(clientEphemeralPublicKeyData)
	hash.Write([]byte("DarkStar"))
	hash.Write([]byte("client"))

	return hash.Sum(nil), nil
}

func (a *DarkStarServer) getServerIdentifier(host string, port int) []byte {
	hostIP := net.ParseIP(host)
	// we do the below part because host IP in bytes is 16 bytes with padding at the beginning
	hostBytes := []byte(hostIP)[12:16]
	portUint := uint16(port)
	portBuffer := []byte{0, 0}
	binary.BigEndian.PutUint16(portBuffer, portUint)

	buffer := make([]byte, 0)
	buffer = append(buffer, hostBytes...)
	buffer = append(buffer, portBuffer...)

	return buffer
}

func (a *DarkStarServer) makeServerIdentifier(host string, port int) []byte {
	hostIP := net.ParseIP(host)
	hostBytes := []byte(hostIP.String())
	portUint := uint16(port)
	portBuffer := []byte{0, 0}
	binary.BigEndian.PutUint16(portBuffer, portUint)

	hash := sha256.New()
	hash.Write(hostBytes)
	hash.Write(portBuffer)

	return hash.Sum(nil)
}

package polish

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/OperatorFoundation/go-shadowsocks2/darkstar"
	"github.com/aead/ecdh"
)

type DarkStarPolishServerConfig struct {
	ServerAddress    string `json:"serverAddress"`
	ServerPrivateKey string `json:"serverPrivateKey"`
}

type DarkStarPolishClientConfig struct {
	ServerAddress   string `json:"serverAddress"`
	ServerPublicKey string `json:"serverPublicKey"`
}

type DarkStarPolishServer struct {
	darkStarServer darkstar.DarkStarServer
}

type DarkStarPolishServerConnection struct {
	darkStarServer darkstar.DarkStarServer
	conn           net.Conn
}

type DarkStarPolishClientConnection struct {
	darkStarClient darkstar.DarkStarClient
}

func (serverConfig DarkStarPolishServerConfig) Construct() (Server, error) {
	return NewDarkStarServer(serverConfig), nil
}

func (clientConfig DarkStarPolishClientConfig) Construct() (Connection, error) {
	return NewDarkStarClient(clientConfig), nil
}

func (server DarkStarPolishServer) NewConnection(conn net.Conn) Connection {
	// this does the DarkStar handshake
	// serverStreamConn, connError := server.darkStarServer.StreamConn(conn)
	// fmt.Printf("streamConn type: %T\n", serverStreamConn)
	// if connError != nil {
	// 	return nil
	// }

	return &DarkStarPolishServerConnection{
		darkStarServer: server.darkStarServer,
		conn:           conn,
	}
}

// TODO: handshake behavior is already performed in DarkStarPolishServer.darkStarServer.StreamConn()
func (serverConn *DarkStarPolishServerConnection) Handshake(conn net.Conn) (net.Conn, error) {
	streamConn, connError := serverConn.darkStarServer.StreamConn(conn)
	if connError != nil {
		return nil, connError
	}

	if streamConn == nil {
		return nil, errors.New("streamConn in server handshake returned nil")
	}

	return streamConn, nil
}

func (clientConn *DarkStarPolishClientConnection) Handshake(conn net.Conn) (net.Conn, error) {
	streamConn, connError := clientConn.darkStarClient.StreamConn(conn)
	if connError != nil {
		return nil, connError
	}
	if streamConn == nil {
		return nil, errors.New("streamConn in client handshake returned nil")
	}
	return streamConn, nil
}

func NewDarkStarPolishClientConfigFromPrivate(serverPrivateKey crypto.PrivateKey) (*DarkStarPolishClientConfig, error) {
	keyExchange := ecdh.Generic(elliptic.P256())
	serverPublicKey := keyExchange.PublicKey(serverPrivateKey)
	fmt.Print("server publicKey: ")
	fmt.Println(serverPublicKey)
	publicKeyBytes, keyError := darkstar.PublicKeyToBytes(serverPublicKey)
	if keyError != nil {
		return nil, keyError
	}

	return &DarkStarPolishClientConfig{
		ServerPublicKey: base64.StdEncoding.EncodeToString(publicKeyBytes),
	}, nil
}

func NewDarkStarPolishClientConfig(serverPublicKey []byte) (*DarkStarPolishClientConfig, error) {
	return &DarkStarPolishClientConfig{
		ServerPublicKey: base64.StdEncoding.EncodeToString(serverPublicKey),
	}, nil
}

func NewDarkStarPolishServerConfig() (*DarkStarPolishServerConfig, error) {
	keyExchange := ecdh.Generic(elliptic.P256())
	serverEphemeralPrivateKey, _, keyError := keyExchange.GenerateKey(rand.Reader)
	if keyError != nil {
		return nil, keyError
	}

	privateKeyBytes, ok := serverEphemeralPrivateKey.([]byte)
	if !ok {
		return nil, errors.New("could not convert private key to bytes")
	}

	return &DarkStarPolishServerConfig{
		ServerPrivateKey: base64.StdEncoding.EncodeToString(privateKeyBytes),
	}, nil
}

func NewDarkStarClient(config DarkStarPolishClientConfig) Connection {
	// Get a host and port from the provided address string
	addressArray := strings.Split(config.ServerAddress, ":")
	host := addressArray[0]
	port, stringErr := strconv.Atoi(addressArray[1])
	if stringErr != nil {
		fmt.Println("Error: failed to make the port string into an int")
	}

	darkStarClient := darkstar.NewDarkStarClient(config.ServerPublicKey, host, port)
	darkStarClientConnection := DarkStarPolishClientConnection{
		darkStarClient: *darkStarClient,
	}
	return &darkStarClientConnection
}

func NewDarkStarServer(config DarkStarPolishServerConfig) DarkStarPolishServer {
	// Get a host and port from the provided address string
	addressArray := strings.Split(config.ServerAddress, ":")
	host := addressArray[0]
	port, stringErr := strconv.Atoi(addressArray[1])
	if stringErr != nil {
		fmt.Println("Error: failed to make the port string into an int")
	}

	darkStarServer := darkstar.NewDarkStarServer(config.ServerPrivateKey, host, port)
	darkStarPolishServer := DarkStarPolishServer{
		darkStarServer: *darkStarServer,
	}
	return darkStarPolishServer
}

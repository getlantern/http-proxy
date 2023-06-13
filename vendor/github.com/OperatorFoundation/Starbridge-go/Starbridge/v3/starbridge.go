package Starbridge

import (
	"crypto"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"

	replicant "github.com/OperatorFoundation/Replicant-go/Replicant/v3"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/polish"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/toneburst"
	"github.com/OperatorFoundation/go-shadowsocks2/darkstar"
	"github.com/aead/ecdh"
	"golang.org/x/net/proxy"
)

type TransportClient struct {
	Config  ClientConfig
	Address string
	// TODO: Dialer can be removed later (both here and in dispatcher)
	Dialer proxy.Dialer
}

type TransportServer struct {
	Config  ServerConfig
	Address string
	// TODO: Dialer can be removed later (both here and in dispatcher)
	Dialer proxy.Dialer
}

type ClientConfig struct {
	ServerAddress   string `json:"serverAddress"`
	ServerPublicKey string `json:"serverPublicKey"`
	Transport       string `json:"transport"`
}

type ServerConfig struct {
	ServerAddress    string `json:"serverAddress"`
	ServerPrivateKey string `json:"serverPrivateKey"`
	Transport        string `json:"transport"`
}

type starbridgeTransportListener struct {
	address  string
	listener *net.TCPListener
	config   ServerConfig
}

func newStarbridgeTransportListener(address string, listener *net.TCPListener, config ServerConfig) *starbridgeTransportListener {
	return &starbridgeTransportListener{address: address, listener: listener, config: config}
}

func (listener *starbridgeTransportListener) Addr() net.Addr {
	interfaces, _ := net.Interfaces()
	addrs, _ := interfaces[0].Addrs()
	return addrs[0]
}

// Accept waits for and returns the next connection to the listener.
func (listener *starbridgeTransportListener) Accept() (net.Conn, error) {
	keyBytes, keyError := base64.StdEncoding.DecodeString(listener.config.ServerPrivateKey)
	if keyError != nil {
		return nil, keyError
	}

	keyCheckSuccess := CheckPrivateKey(keyBytes)
	if !keyCheckSuccess {
		return nil, errors.New("bad private key")
	}

	replicantConfig := getServerConfig(listener.address, keyBytes)

	conn, err := listener.listener.Accept()
	if err != nil {
		return nil, err
	}

	serverConn, serverError := NewServerConnection(replicantConfig, conn)
	if serverError != nil {
		conn.Close()
		return nil, serverError
	}

	return serverConn, nil
}

// Close closes the transport listener.
// Any blocked Accept operations will be unblocked and return errors.
func (listener *starbridgeTransportListener) Close() error {
	return listener.listener.Close()
}

// Listen checks for a working connection
func (config ServerConfig) Listen() (net.Listener, error) {
	if strings.ToLower(config.Transport) != "starbridge" {
		return nil, errors.New("incorrect transport name")
	}

	addr, resolveErr := ResolveAddr(config.ServerAddress)
	if resolveErr != nil {
		return nil, resolveErr
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return newStarbridgeTransportListener(config.ServerAddress, ln, config), nil
}

// Dial connects to the address on the named network
func (config ClientConfig) Dial() (net.Conn, error) {
	if strings.ToLower(config.Transport) != "starbridge" {
		return nil, errors.New("incorrect transport name")
	}

	keyBytes, keyError := base64.StdEncoding.DecodeString(config.ServerPublicKey)
	if keyError != nil {
		return nil, keyError
	}

	publicKey := darkstar.BytesToPublicKey(keyBytes)

	keyCheckError := CheckPublicKey(publicKey)
	if keyCheckError != nil {
		return nil, keyCheckError
	}

	dialTimeout := time.Minute * 5
	conn, dialErr := net.DialTimeout("tcp", config.ServerAddress, dialTimeout)
	if dialErr != nil {
		return nil, dialErr
	}

	replicantConfig := getClientConfig(config.ServerAddress, keyBytes)
	transportConn, err := NewClientConnection(replicantConfig, conn)

	if err != nil {
		if conn != nil {
			_ = conn.Close()
		}
		return nil, err
	}

	return transportConn, nil
}

func NewClient(config ClientConfig, dialer proxy.Dialer) TransportClient {
	return TransportClient{
		Config:  config,
		Address: config.ServerAddress,
		Dialer:  dialer,
	}
}

func NewServer(config ServerConfig, address string, dialer proxy.Dialer) TransportServer {
	return TransportServer{
		Config:  config,
		Address: address,
		Dialer:  dialer,
	}
}

// Dial creates outgoing transport connection
func (transport *TransportClient) Dial() (net.Conn, error) {
	keyBytes, keyError := base64.StdEncoding.DecodeString(transport.Config.ServerPublicKey)
	if keyError != nil {
		return nil, keyError
	}

	publicKey := darkstar.BytesToPublicKey(keyBytes)

	keyCheckError := CheckPublicKey(publicKey)
	if keyCheckError != nil {
		return nil, keyCheckError
	}

	replicantConfig := getClientConfig(transport.Address, keyBytes)

	dialTimeout := time.Minute * 5
	conn, dialErr := net.DialTimeout("tcp", transport.Address, dialTimeout)
	if dialErr != nil {
		return nil, dialErr
	}

	dialConn := conn

	transportConn, err := NewClientConnection(replicantConfig, conn)

	if err != nil {
		_ = dialConn.Close()
		return nil, err
	}

	return transportConn, nil
}

func (transport *TransportServer) Listen() (net.Listener, error) {
	addr, resolveErr := ResolveAddr(transport.Address)
	if resolveErr != nil {
		return nil, resolveErr
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return newStarbridgeTransportListener(transport.Address, ln, transport.Config), nil
}

func NewClientConnection(config replicant.ClientConfig, conn net.Conn) (net.Conn, error) {
	return replicant.NewClientConnection(conn, config)
}

func NewServerConnection(config replicant.ServerConfig, conn net.Conn) (net.Conn, error) {
	return replicant.NewServerConnection(conn, config)
}

func NewReplicantClientConnectionState(config replicant.ClientConfig) (*replicant.ConnectionState, error) {
	return replicant.NewReplicantClientConnectionState(config)
}

func NewReplicantServerConnectionState(config replicant.ServerConfig, polishServer polish.Server, conn net.Conn) (*replicant.ConnectionState, error) {
	return replicant.NewReplicantServerConnectionState(config, polishServer, conn)
}

func getClientConfig(serverAddress string, serverPublicKey []byte) replicant.ClientConfig {
	polishClientConfig := polish.DarkStarPolishClientConfig{
		ServerAddress:   serverAddress,
		ServerPublicKey: base64.StdEncoding.EncodeToString(serverPublicKey),
	}

	toneburstClientConfig := toneburst.StarburstConfig{
		Mode: "SMTPClient",
	}

	clientConfig := replicant.ClientConfig{
		Toneburst: toneburstClientConfig,
		Polish:    polishClientConfig,
	}

	return clientConfig
}

func getServerConfig(serverAddress string, serverPrivateKey []byte) replicant.ServerConfig {
	polishServerConfig := polish.DarkStarPolishServerConfig{
		ServerAddress:    serverAddress,
		ServerPrivateKey: base64.StdEncoding.EncodeToString(serverPrivateKey),
	}

	toneburstServerConfig := toneburst.StarburstConfig{
		Mode: "SMTPServer",
	}

	serverConfig := replicant.ServerConfig{
		Toneburst: toneburstServerConfig,
		Polish:    polishServerConfig,
	}

	return serverConfig
}

func CheckPrivateKey(privKey crypto.PrivateKey) (success bool) {
	defer func() {
		if panicError := recover(); panicError != nil {
			success = false
		} else {
			success = true
		}
	}()

	keyExchange := ecdh.Generic(elliptic.P256())
	_, pubKey, keyError := keyExchange.GenerateKey(rand.Reader)
	if keyError != nil {
		success = false
		return
	}

	// verify that the given key bytes are on the chosen elliptic curve
	success = keyExchange.ComputeSecret(privKey, pubKey) != nil
	return
}

func CheckPublicKey(pubkey crypto.PublicKey) (keyError error) {
	defer func() {
		if panicError := recover(); panicError != nil {
			keyError = errors.New("panicked on public key check")
		}
	}()

	// verify that the given key bytes are on the chosen elliptic curve
	keyExchange := ecdh.Generic(elliptic.P256())
	result := keyExchange.Check(pubkey)
	keyError = result
	return
}

func GenerateKeys() (publicKeyString, privateKeyString *string, keyError error) {
	keyExchange := ecdh.Generic(elliptic.P256())
	clientEphemeralPrivateKey, clientEphemeralPublicKeyPoint, keyError := keyExchange.GenerateKey(rand.Reader)
	if keyError != nil {
		return nil, nil, keyError
	}

	privateKeyBytes, ok := clientEphemeralPrivateKey.([]byte)
	if !ok {
		return nil, nil, errors.New("failed to convert privateKey to bytes")
	}

	publicKeyBytes, keyByteError := darkstar.PublicKeyToBytes(clientEphemeralPublicKeyPoint)
	if keyByteError != nil {
		return nil, nil, keyByteError
	}

	privateKey := base64.StdEncoding.EncodeToString(privateKeyBytes)
	publicKey := base64.StdEncoding.EncodeToString(publicKeyBytes)
	return &publicKey, &privateKey, nil
}

func GenerateNewConfigPair(address string) (*ServerConfig, *ClientConfig, error) {
	publicKey, privateKey, keyError := GenerateKeys()
	if keyError != nil {
		return nil, nil, keyError
	}

	serverConfig := ServerConfig{
		ServerAddress:    address,
		ServerPrivateKey: *privateKey,
		Transport:        "Starbridge",
	}

	clientConfig := ClientConfig{
		ServerAddress:   address,
		ServerPublicKey: *publicKey,
		Transport:       "Starbridge",
	}

	return &serverConfig, &clientConfig, nil
}

func GenerateConfigFiles(address string) error {
	serverConfig, clientConfig, configError := GenerateNewConfigPair(address)
	if configError != nil {
		return configError
	}

	clientConfigBytes, clientMarshalError := json.Marshal(clientConfig)
	if clientMarshalError != nil {
		return clientMarshalError
	}

	serverConfigBytes, serverMarshalError := json.Marshal(serverConfig)
	if serverMarshalError != nil {
		return serverMarshalError
	}

	serverConfigWriteError := ioutil.WriteFile("StarbridgeServerConfig.json", serverConfigBytes, 0644)
	if serverConfigWriteError != nil {
		return serverConfigWriteError
	}

	clientConfigWriteError := ioutil.WriteFile("StarbridgeClientConfig.json", clientConfigBytes, 0644)
	if clientConfigWriteError != nil {
		return clientConfigWriteError
	}

	return nil
}

// Resolve an address string into a net.TCPAddr. We are a bit more strict than
// net.ResolveTCPAddr; we don't allow an empty host or port, and the host part
// must be a literal IP address.
func ResolveAddr(addrStr string) (*net.TCPAddr, error) {
	ipStr, portStr, err := net.SplitHostPort(addrStr)
	if err != nil {
		// Before the fixing of bug #7011, tor doesn't put brackets around IPv6
		// addresses. Split after the last colon, assuming it is a port
		// separator, and try adding the brackets.
		parts := strings.Split(addrStr, ":")
		if len(parts) <= 2 {
			return nil, err
		}
		addrStr := "[" + strings.Join(parts[:len(parts)-1], ":") + "]:" + parts[len(parts)-1]
		ipStr, portStr, err = net.SplitHostPort(addrStr)
	}
	if err != nil {
		return nil, err
	}
	if ipStr == "" {
		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a host part", addrStr))
	}
	if portStr == "" {
		return nil, net.InvalidAddrError(fmt.Sprintf("address string %q lacks a port part", addrStr))
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, net.InvalidAddrError(fmt.Sprintf("not an IP string: %q", ipStr))
	}
	port, err := parsePort(portStr)
	if err != nil {
		return nil, err
	}
	return &net.TCPAddr{IP: ip, Port: port}, nil
}

func parsePort(portStr string) (int, error) {
	port, err := strconv.ParseUint(portStr, 10, 16)
	return int(port), err
}

package vmess

// this file is a thin wrapper around github.com/sagernet/sing-vmess
// it ignores the 'destination' part of vmess protocol since http-proxy
// is handling that
// additionally, any 'legacy' code is removed (alterIds, etc)

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"math"
	"net"
	"time"

	"github.com/gofrs/uuid/v5"
	vmess "github.com/sagernet/sing-vmess"
	"github.com/sagernet/sing/common"
	"github.com/sagernet/sing/common/buf"
	"github.com/sagernet/sing/common/bufio"
	E "github.com/sagernet/sing/common/exceptions"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/common/replay"
	"github.com/sagernet/sing/common/rw"
)

var (
	ErrBadHeader    = errors.New("bad header")
	ErrBadTimestamp = errors.New("bad timestamp")
	ErrReplay       = errors.New("replayed request")
	ErrBadRequest   = errors.New("bad request")
	ErrBadVersion   = errors.New("bad version")
)

type vmessListener struct {
	net.Listener
	userKey      map[int][16]byte
	userIdCipher map[int]cipher.Block
	replayFilter replay.Filter
}

func NewVMessListener(baseListener net.Listener, uuids []string) (net.Listener, error) {
	userKeyMap := make(map[int][16]byte)
	userIdCipherMap := make(map[int]cipher.Block)
	for user, userId := range uuids {
		userUUID := uuid.FromStringOrNil(userId)
		if userUUID == uuid.Nil {
			return nil, E.New("invalid user uuid: ", userId)
		}
		userCmdKey := vmess.Key(userUUID)
		userKeyMap[user] = userCmdKey
		userIdCipher, err := aes.NewCipher(vmess.KDF(userCmdKey[:], vmess.KDFSaltConstAuthIDEncryptionKey)[:16])
		if err != nil {
			return nil, err
		}
		userIdCipherMap[user] = userIdCipher
	}
	replayFilter := replay.NewSimple(time.Second * 120)
	return &vmessListener{baseListener, userKeyMap, userIdCipherMap, replayFilter}, nil
}

func (l *vmessListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	const headerLenBufferLen = 2 + vmess.CipherOverhead
	const aeadMinHeaderLen = 16 + headerLenBufferLen + 8 + vmess.CipherOverhead + 42
	minHeaderLen := aeadMinHeaderLen

	requestBuffer := buf.New()
	defer requestBuffer.Release()

	n, err := requestBuffer.ReadOnceFrom(conn)
	if err != nil {
		if errors.Is(err, io.EOF) {
			conn.Close()
			return conn, nil
		}
		return nil, err
	}
	if n < minHeaderLen {
		return nil, ErrBadHeader
	}
	authId := requestBuffer.To(16)
	var decodedId [16]byte
	var user int
	var found bool
	for currUser, userIdBlock := range l.userIdCipher {
		userIdBlock.Decrypt(decodedId[:], authId)
		timestamp := int64(binary.BigEndian.Uint64(decodedId[:]))
		checksum := binary.BigEndian.Uint32(decodedId[12:])
		if crc32.ChecksumIEEE(decodedId[:12]) != checksum {
			continue
		}
		if math.Abs(math.Abs(float64(timestamp))-float64(time.Now().Unix())) > 120 {
			return nil, ErrBadTimestamp
		}
		if !l.replayFilter.Check(decodedId[:]) {
			return nil, ErrReplay
		}
		user = currUser
		found = true
		break
	}

	if !found {
		return nil, ErrBadRequest
	}

	cmdKey := l.userKey[user]
	var headerReader io.Reader
	var headerBuffer []byte

	var reader io.Reader

	if requestBuffer.Len() < aeadMinHeaderLen {
		return nil, ErrBadHeader
	}

	reader = conn

	const nonceIndex = 16 + headerLenBufferLen
	connectionNonce := requestBuffer.Range(nonceIndex, nonceIndex+8)

	lengthKey := vmess.KDF(cmdKey[:], vmess.KDFSaltConstVMessHeaderPayloadLengthAEADKey, authId, connectionNonce)[:16]
	lengthNonce := vmess.KDF(cmdKey[:], vmess.KDFSaltConstVMessHeaderPayloadLengthAEADIV, authId, connectionNonce)[:12]
	lengthBuffer, err := newAesGcm(lengthKey).Open(requestBuffer.Index(16), lengthNonce, requestBuffer.Range(16, nonceIndex), authId)
	if err != nil {
		return nil, err
	}

	const headerIndex = nonceIndex + 8
	headerLength := int(binary.BigEndian.Uint16(lengthBuffer))
	needRead := headerLength + headerIndex + vmess.CipherOverhead - requestBuffer.Len()
	if needRead > 0 {
		_, err = requestBuffer.ReadFullFrom(conn, needRead)
		if err != nil {
			return nil, err
		}
	}

	headerKey := vmess.KDF(cmdKey[:], vmess.KDFSaltConstVMessHeaderPayloadAEADKey, authId, connectionNonce)[:16]
	headerNonce := vmess.KDF(cmdKey[:], vmess.KDFSaltConstVMessHeaderPayloadAEADIV, authId, connectionNonce)[:12]
	headerBuffer, err = newAesGcm(headerKey).Open(requestBuffer.Index(headerIndex), headerNonce, requestBuffer.Range(headerIndex, headerIndex+headerLength+vmess.CipherOverhead), authId)
	if err != nil {
		return nil, err
	}
	// replace with < if support mux
	if len(headerBuffer) <= 38 {
		return nil, E.Extend(ErrBadHeader, io.ErrShortBuffer)
	}
	requestBuffer.Advance(headerIndex + headerLength + vmess.CipherOverhead)
	headerReader = bytes.NewReader(headerBuffer[38:])

	version := headerBuffer[0]
	if version != vmess.Version {
		return nil, E.Extend(ErrBadVersion, version)
	}

	requestBodyKey := make([]byte, 16)
	requestBodyNonce := make([]byte, 16)

	copy(requestBodyKey, headerBuffer[17:33])
	copy(requestBodyNonce, headerBuffer[1:17])

	responseHeader := headerBuffer[33]
	option := headerBuffer[34]
	paddingLen := int(headerBuffer[35] >> 4)
	security := headerBuffer[35] & 0x0F
	command := headerBuffer[37]
	switch command {
	case vmess.CommandTCP, vmess.CommandUDP, vmess.CommandMux:
	default:
		return nil, E.New("unknown command: ", command)
	}
	if command == vmess.CommandUDP && option == 0 {
		return nil, E.New("bad packet connection")
	}
	if paddingLen > 0 {
		_, err = io.CopyN(io.Discard, headerReader, int64(paddingLen))
		if err != nil {
			return nil, E.Extend(ErrBadHeader, "bad padding")
		}
	}
	err = rw.SkipN(headerReader, 4)
	if err != nil {
		return nil, err
	}
	if requestBuffer.Len() > 0 {
		reader = bufio.NewCachedReader(reader, requestBuffer)
	}
	reader = vmess.CreateReader(reader, nil, requestBodyKey, requestBodyNonce, requestBodyKey, requestBodyNonce, security, option)
	if option&vmess.RequestOptionChunkStream != 0 && command == vmess.CommandTCP || command == vmess.CommandMux {
		reader = bufio.NewChunkReader(reader, vmess.ReadChunkSize)
	}
	rawConn := rawServerConn{
		Conn:           conn,
		requestKey:     requestBodyKey,
		requestNonce:   requestBodyNonce,
		responseHeader: responseHeader,
		security:       security,
		option:         option,
		reader:         bufio.NewExtendedReader(reader),
	}
	return &serverConn{rawConn}, nil
}

type rawServerConn struct {
	net.Conn
	requestKey     []byte
	requestNonce   []byte
	responseHeader byte
	security       byte
	option         byte
	reader         N.ExtendedReader
	writer         N.ExtendedWriter
}

func (c *rawServerConn) writeResponse() error {
	responseBuffer := buf.NewSize(2 + vmess.CipherOverhead + 4 + vmess.CipherOverhead)
	defer responseBuffer.Release()

	_responseKey := sha256.Sum256(c.requestKey[:])
	responseKey := _responseKey[:16]
	_responseNonce := sha256.Sum256(c.requestNonce[:])
	responseNonce := _responseNonce[:16]

	headerLenKey := vmess.KDF(responseKey, vmess.KDFSaltConstAEADRespHeaderLenKey)[:16]
	headerLenNonce := vmess.KDF(responseNonce, vmess.KDFSaltConstAEADRespHeaderLenIV)[:12]
	headerLenCipher := newAesGcm(headerLenKey)
	binary.BigEndian.PutUint16(responseBuffer.Extend(2), 4)
	headerLenCipher.Seal(responseBuffer.Index(0), headerLenNonce, responseBuffer.Bytes(), nil)
	responseBuffer.Extend(vmess.CipherOverhead)

	headerKey := vmess.KDF(responseKey, vmess.KDFSaltConstAEADRespHeaderPayloadKey)[:16]
	headerNonce := vmess.KDF(responseNonce, vmess.KDFSaltConstAEADRespHeaderPayloadIV)[:12]
	headerCipher := newAesGcm(headerKey)
	common.Must(
		responseBuffer.WriteByte(c.responseHeader),
		responseBuffer.WriteByte(c.option),
		responseBuffer.WriteZeroN(2),
	)
	const headerIndex = 2 + vmess.CipherOverhead
	headerCipher.Seal(responseBuffer.Index(headerIndex), headerNonce, responseBuffer.From(headerIndex), nil)
	responseBuffer.Extend(vmess.CipherOverhead)

	_, err := c.Conn.Write(responseBuffer.Bytes())
	if err != nil {
		return err
	}

	c.writer = bufio.NewExtendedWriter(vmess.CreateWriter(c.Conn, nil, c.requestKey, c.requestNonce, responseKey, responseNonce, c.security, c.option))
	return nil
}

func (c *rawServerConn) Close() error {
	return common.Close(
		c.Conn,
		c.reader,
	)
}

func (c *rawServerConn) FrontHeadroom() int {
	return vmess.MaxFrontHeadroom
}

func (c *rawServerConn) RearHeadroom() int {
	return vmess.MaxRearHeadroom
}

func (c *rawServerConn) NeedHandshake() bool {
	return c.writer == nil
}

func (c *rawServerConn) NeedAdditionalReadDeadline() bool {
	return true
}

func (c *rawServerConn) Upstream() any {
	return c.Conn
}

type serverConn struct {
	rawServerConn
}

func (c *serverConn) Read(b []byte) (n int, err error) {
	return c.reader.Read(b)
}

func (c *serverConn) Write(b []byte) (n int, err error) {
	if c.writer == nil {
		err = c.writeResponse()
		if err != nil {
			return
		}
	}
	return c.writer.Write(b)
}

func (c *serverConn) ReadBuffer(buffer *buf.Buffer) error {
	return c.reader.ReadBuffer(buffer)
}

func (c *serverConn) WriteBuffer(buffer *buf.Buffer) error {
	if c.writer == nil {
		err := c.writeResponse()
		if err != nil {
			buffer.Release()
			return err
		}
	}
	return c.writer.WriteBuffer(buffer)
}

func (c *serverConn) WriteTo(w io.Writer) (n int64, err error) {
	return bufio.Copy(w, c.reader)
}

func (c *serverConn) ReadFrom(r io.Reader) (n int64, err error) {
	if c.writer == nil {
		err = c.writeResponse()
		if err != nil {
			return
		}
	}
	return bufio.Copy(c.writer, r)
}

func newAesGcm(key []byte) cipher.AEAD {
	block, err := aes.NewCipher(key)
	common.Must(err)
	outCipher, err := cipher.NewGCM(block)
	common.Must(err)
	return outCipher
}

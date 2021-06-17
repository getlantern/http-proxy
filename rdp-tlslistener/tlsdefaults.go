package rdplistener

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"math/big"
	"net"
	"strings"
	"time"
	"unicode/utf16"
)

var (
	generatedCert *tls.Certificate
	generatedKey  *rsa.PrivateKey
)

// BuildListenerConfig builds a tls.Config for a listener at the given addr
func BuildListenerConfig() (*tls.Config, error) {
	RDPCert, err := makeRDPcert()
	if err != nil {
		// TODO: explode
	}

	tlsConfig := &tls.Config{
		PreferServerCipherSuites: true,
	}
	tlsConfig.Certificates = []tls.Certificate{*RDPCert}

	return tlsConfig, nil
}

// makeRDPHostname Generates a host name that looks very windows-like
func makeRDPHostname() string {
	// There are a few kinds of host name windows can have:
	// WIN-L5655A9TLJD -- Aka, desktop windows
	// \x001\x000\x00_\x000\x00_\x008\x00_\x004 -- Aka, UTF16 hostnames like '10_0_8_4'
	// The other common ones one shodan are cloud spesific, so we will stay clear of them for now
	coinToss := make([]byte, 3)
	rand.Read(coinToss)
	if coinToss[0] > 70 {
		// Windows Desktop hostname, at 30% population
		// Sample of hosts from Shodan:
		// WIN-4NLQSKDG430
		// WIN-S0UMC4DPMOI
		// WIN-APHMTPC3J62
		// WIN-LNGJ3Q8LNJ2
		// WIN-GGVEFCJ5KQ7
		// WIN-L5655A9TLJD
		// WIN-EUOA81ACRT5
		// WIN-LCU8727LJ68
		// WIN-HUGL9V6VR83
		// WIN-N2M1KDQT65Q
		// for some reason windows hosts names seem to follow base32hex just without 'B'
		return fmt.Sprintf("WIN-%s", SecureRandomString("0123456789ACDEFGHIJKLNMOPQRSTUV", 11))
	} else {
		// UTF16 IP address name, seems super common in SEA when I looked at shodan.
		v4Address := make(net.IP, 4)
		rand.Read(v4Address)

		if coinToss[1] > 60 {
			// 10.0.0.0/8
			v4Address[0] = 10
		} else if coinToss[1] > 80 {
			// 192.168.0.0/16
			v4Address[0] = 192
			v4Address[1] = 168
		} else {
			// 172.16.0.0/12
			// 172.16.0.0–172.31.255.255
			v4Address[0] = 172
			v4Address[1] = 16 + (16 % coinToss[2])
		}

		IPString := strings.Replace(v4Address.String(), ".", "_", -1)
		return utf16sToString(utf16.Encode([]rune(IPString)))
	}

}

// ♥ windows obsession with UTF-16 and all of the misery it causes later on
func utf16sToString(shorts []uint16) string {
	finalString := make([]byte, 0)
	var bytes [2]byte

	for _, v := range shorts {
		bytes[0] = byte(v >> 8)
		bytes[1] = byte(v & 255)
		finalString = append(finalString, bytes[0], bytes[1])
	}

	return string(finalString)
}

// Peace to -- https://stackoverflow.com/a/55860599
func SecureRandomString(availableCharBytes string, length int) string {
	// Compute bitMask
	availableCharLength := len(availableCharBytes)
	if availableCharLength == 0 || availableCharLength > 256 {
		panic("availableCharBytes length must be greater than 0 and less than or equal to 256")
	}
	var bitLength byte
	var bitMask byte
	for bits := availableCharLength - 1; bits != 0; {
		bits = bits >> 1
		bitLength++
	}
	bitMask = 1<<bitLength - 1

	// Compute bufferSize
	bufferSize := length + length/3

	// Create random string
	result := make([]byte, length)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			// Random byte buffer is empty, get a new one
			randomBytes = make([]byte, bufferSize)
			_, err := rand.Read(randomBytes)
			if err != nil {
				log.Fatal("Unable to generate random bytes")
			}
		}
		// Mask bytes to get an index into the character slice
		if idx := int(randomBytes[j%length] & bitMask); idx < availableCharLength {
			result[i] = availableCharBytes[idx]
			i++
		}
	}

	return string(result)
}

// makeRDPcert initializes a PK + cert for use by a server proxy. Generated on system boot every time
// since there is not much point in keeping the cert around. (Or bonus that it looks different some of the time)
func makeRDPcert() (output *tls.Certificate, err error) {
	if generatedCert != nil {
		return generatedCert, nil
	}

	var rsaKey *rsa.PrivateKey
	rsaKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	generatedKey = rsaKey

	// Key made, Let's craft a very Microsoft RDP-ey looking cert
	hostName := makeRDPHostname()

	// Windows RDP certs last for 190 days, but to prevent things looking too "new", we will back date the cert
	// so by the time we fetch the cert, we are 0% to 50% into the lifetime of the cert.
	WindowsCertValidFor := int64(86400 * 190)
	CertStartOffset := (time.Now().UnixNano() % (WindowsCertValidFor)) / 2
	// CertStartOffset wraps every 0.016416 seconds
	CertStart := time.Now().Add((time.Second * time.Duration(CertStartOffset)) * -1)
	CertEnd := CertStart.Add(time.Hour * 24 * 190)

	// SN needs to be random
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: hostName,
		},
		Issuer: pkix.Name{
			CommonName: hostName,
		},
		NotBefore:             CertStart,
		NotAfter:              CertEnd,
		Extensions:            []pkix.Extension{},
		BasicConstraintsValid: true,
		/*
				KeyUsage: 30
			    0... .... = digitalSignature: False
			    .0.. .... = contentCommitment: False
			    ..1. .... = keyEncipherment: True
			    ...1 .... = dataEncipherment: True
			    .... 0... = keyAgreement: False
			    .... .0.. = keyCertSign: False
			    .... ..0. = cRLSign: False
			    .... ...0 = encipherOnly: False
			    0... .... = decipherOnly: False
		*/
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment,
	}

	cert, err := x509.CreateCertificate(rand.Reader, template, template, generatedKey.Public(), generatedKey)
	var outCert tls.Certificate
	outCert.Certificate = append(outCert.Certificate, cert)
	outCert.PrivateKey = generatedKey
	generatedCert = &outCert

	return &outCert, nil
}

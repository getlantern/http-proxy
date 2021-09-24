package rdplistener

import (
	"bytes"
	"log"
	"testing"
)

func TestRDPCookieStripping(t *testing.T) {

	// Taken from wireshark
	// 0030   03 00 00 2f 2a e0 00 00 00 00                           .../*.....
	// 0040   00 43 6f 6f 6b 69 65 3a 20 6d 73 74 73 68 61 73   .Cookie: mstshas
	// 0050   68 3d 41 64 6d 69 6e 69 73 74 72 0d 0a 01 00 08   h=Administr.....
	// 0060   00 0b 00 00 00                                    .....

	HasCookie := "\x03\x00\x00\x2f\x2a\xe0\x00\x00\x00\x00" +
		"\x00\x43\x6f\x6f\x6b\x69\x65\x3a\x20\x6d\x73\x74\x73\x68\x61\x73" +
		"\x68\x3d\x41\x64\x6d\x69\x6e\x69\x73\x74\x72\x0d\x0a\x01\x00\x08" +
		"\x00\x0b\x00\x00\x00"

	HasCookieStripped := "\x03\x00\x00\x2f\x2a\xe0\x00\x00\x00\x00" +
		"\x00\x01\x00\x08\x00\x0b\x00\x00\x00"

	StrippedCookie := maybeRemoveRoutingCookie([]byte(HasCookie))
	if bytes.Compare([]byte(HasCookieStripped), StrippedCookie) != 0 {
		log.Printf("%#v", StrippedCookie)
		t.Fail()
	}

	// Now we will test the crash-ey cases

	HasCookie = "\x03\x00\x00\x2f\x2a\xe0\x00\x00\x00\x00" +
		"\x00\x43\x6f\x6f\x6b\x69\x65\x3a\x20\x6d\x73\x74\x73\x68\x61\x73" +
		"\x68\x3d\x41\x64\x6d\x69\x6e\x69\x73\x74\x72"
	StrippedCookie = maybeRemoveRoutingCookie([]byte(HasCookie))
	if bytes.Compare([]byte(HasCookie), StrippedCookie) != 0 {
		t.Fail()
	}

}

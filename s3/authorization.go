package s3

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	sdk "github.com/aws/aws-sdk-go/aws/signer/v4"
)

const (
	bucket = "getlantern-replica-replica"
	region = "ap-northeast-1"
)

func newCredentials() *credentials.Credentials {
	credentials := credentials.NewStaticCredentials(
		"AKIAZYODAL6XMYV5L5EH",
		"3XNkncV0HjRotWctTrAkCbUqF7So27jgi3kxG4ws",
		"",
	)
	return credentials
}

func GenerateSignature(req *http.Request) (authHeader http.Header, err error) {
	credentials := newCredentials()

	data, err := ioutil.ReadAll(req.Body)
	// hash := fmt.Sprintf("%x", sha256.Sum256(data))
	// req.Header.Set("X-Amz-Content-Sha256", hash)

	bodyReader := bytes.NewReader(data)
	service := "s3"
	// expiration := time.Duration(1000 * time.Second)
	signTime := time.Now()

	signer := sdk.NewSigner(credentials)
	authHeader, err = signer.Sign(req, bodyReader, service, region, signTime)
	return
}

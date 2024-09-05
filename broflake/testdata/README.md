Directory for storing testdata for broflake.

## Creating certificates

```bash
openssl genpkey -algorithm RSA -out key.pem -pkeyopt rsa_keygen_bits:2048
openssl req -new -key key.pem -out csr.pem -subj "/CN=127.0.0.1"
openssl x509 -req -days 365 -in csr.pem -signkey key.pem -out cert.pem
```

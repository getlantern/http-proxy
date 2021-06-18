FROM bitnami/redis:latest

COPY "cert.pem" .
COPY "key.pem" .

ENV REDIS_TLS_ENABLED=true
ENV REDIS_TLS_CERT_FILE="cert.pem"
ENV REDIS_TLS_KEY_FILE="key.pem"
ENV REDIS_TLS_AUTH_CLIENTS=no
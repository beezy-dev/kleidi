# Example config to run kleidi with cert auth in standalone docker container
# Before starting, do not forget to configure auth/cert in Vault with the generated intermediate CA cert.
# File kleidi.env is used to populate env variables that are being read by cert auth from the container.
# Adjust to your local configuration.
#!/usr/bin/bash
docker run -d --privileged --restart always \
  --net=host --name kleidi \
  -v /etc/kleidi/vault-config-cert.json:/etc/kleidi/config.json \
  -v /var/run/kleidi/:/var/run/kleidi/:rw \
  -v /etc/kleidi/tls/vault-cert.pem:/etc/ssl/certs/vault-cert.pem \
  -v /etc/kleidi/tls/clientcert-intermediate.pem:/etc/ssl/certs/clientcert-intermediate.pem \
  -v /etc/kleidi/tls/clientkey-intermediate.pem:/etc/ssl/certs/clientkey-intermediate.pem \
  --env-file /etc/kleidi/kleidi.env \
  github.com/beezy-dev/kleidi-kms-plugin:latest \
  -provider=hvault -configfile=/etc/kleidi/config.json -debugmode=true -listen=unix:///var/run/kleidi/kms.socket

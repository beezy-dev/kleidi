#!/usr/bin/bash
# create root CA with serial and index
# works in the current directory
echo "0ABC" > serial.txt
touch index.txt
openssl req -x509 -config conf/openssl-ca.conf -days 365 -newkey rsa:4096 -sha256 -nodes -out cacert.pem -outform PEM

# create intermediate CA and it's folder structure
mkdir intermediate
touch intermediate/index.txt
# create CSR
openssl req -config conf/openssl-intermediate.conf -newkey rsa:4096 -sha256 -nodes -out intermediate/intermediate.csr -outform PEM
# sign it with the previous root CA
openssl ca -config conf/openssl-ca.conf -policy signing_policy -extensions v3_intermediate_ca -out intermediate/intermediate-cacert.pem -infiles intermediate/intermediate.csr
mv intermediate-cakey.pem intermediate/intermediate-cakey.pem
# create client csr and key
openssl req -config conf/openssl-client.conf -newkey rsa:2048 -sha256 -nodes -out clientcert-intermediate.csr -outform PEM
openssl ca -config conf/openssl-intermediate.conf -rand_serial -policy signing_policy -extensions signing_req -out clientcert-intermediate.pem -infiles clientcert-intermediate.csr

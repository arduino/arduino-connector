#!/bin/bash
# call the script as
# ./provision.sh "https://arduino-tmp.s3.amazonaws.com/arduino-connector?AWSAccessKeyId=ASIAISSOELENOR6FZ7GQ&Expires=1530282287&x-amz-security-token=FQoDYXdzEKj%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FwEaDGyqTPGX6ftwuPydtiLPAryRWXiPOIjHgwS6FqM49j6qbe%2BuC67yFlmcuiTpqCC7Am7m0iIYBC6C95f%2Be00hexK649t2ezbKvmj2%2BINPplPmrosbynyr16RqbdTgvXY5VmhHcImHSG1o1ouyRkwTTzB0iFPGHVCouZgpCYDhYOxBkofKEWdQhppkTPF3NrRtCJw47spz0gITe4FeJeB9YIsndNEHdUauViCGcG4kjTiEpkFTY5rXhYHhf2d4aCtdeZ8YrUGTUnBdoi6FxlkUA9aRMEfQ3XGFS1mzDkXovSNNXtUz6qa9M2%2Fpq7gZIt73wNTzCJ8c7SIyi6Ptxaw2nKsTKG%2FOhX5M5Q01UqUOm5iyxG7lhoNZNG7uAVCOTy7wuIt3YBr76MUAUtMC0JmtJUaUBcI9OxqNREfPMgq%2BhOtc8bFp0POC0wfTK4nxisTGf01m5%2BBjKF9cH%2BZ%2FdVdZKIiZydkF&Signature=sC0jceFfSoUfTtU3EUoejGQB8sM%3D"
cd $HOME
# remove old files
rm -f arduino-connector* certificate*
# uninstall previous installations of connector
sudo systemctl stop ArduinoConnector
sudo rm -f /etc/systemd/system/ArduinoConnector.service
# download connector
wget -O arduino-connector -nc ${1}

chmod +x arduino-connector
# generate certs key and csr
sudo -E ./arduino-connector -provision
# returns 
# Version: 2.0.103
# Generate private key
# Generate csr
# -----BEGIN CERTIFICATE REQUEST-----\nMIIBeTCCAR4CAQAwgY0xCzAJBgNVBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRl\nMQ8wDQYDVQQHEwZNeUNpdHkxFDASBgNVBAoTC0NvbXBhbnkgTHRkMQswCQYDVQQL\nEwJJVDEUMBIGA1UEAxMLZXhhbXBsZS5jb20xHzA
# dBgkqhkiG9w0BCQEMEHRlc3RA\nZXhhbXBsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBAATjUzWLt/X6Z8lZ\nJLEgjit9lSQAb54gEB0r73S11ddCWHRS3H261qT/p5FgpwJsmg0zaFnAj+njZQaZ\nN7DlwRXroC4wLAYJKoZIhvcNAQkOMR8wHTAbBgNVHREEFDASg
# RB0ZXN0QGV4YW1w\nbGUuY29tMAoGCCqGSM49BAMCA0kAMEYCIQDadwwf+47GM5+5zi4/Ujhh+d9jKIM4\ne3EORJURK2mdSAIhAKgxVnTZ28gn8OZTwRWomlPNXZ9fwjdbrvX7IibPm0sM\n-----END CERTIFICATE REQUEST-----\n
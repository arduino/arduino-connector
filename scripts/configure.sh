#!/bin/bash
# call the script as
# ./configure.sh "containtel:1061e580-d2f2-4c06-9e48-51eb05ba700d" "a19g5nbe27wn47.iot.us-east-1.amazonaws.com" "-----BEGIN CERTIFICATE-----\nMIIB6zCCAZGgAwIBAgIRALMj3Il+l86W5JZM3gYIKoZIzj0EAwIwfjEL\nMAkGA1UEBhMCVVMxFzAVBgNVBAoTDkFyZHVpbm8gTExDIFVTMQswCQYDVQQLEwJJ\nVDFJMEcGA1UEAxNANTU5Nzg0MWU2OTI4ODA3YzhmZWE4ZGY3YzljZGM3OWRi\nMWM5MTY5MGQ4YTUwNmI5OTE5NzU4YTU0Y2Q3OTAeFw0xODA2MjYxMjAwMDBaFw00\nOTA2MjYxMjAwMDBaMGwxCzAJBgNVBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRl\nMQ8wDQYDVQQHEwZNeUNpdHkxFDASBgNVBAoTC0NvbXBhbnkgTHRkMQswCQYDVQQL\nEwJJVDEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMB\nBwNCAATjUzWLt/X6Z8lZJLEgjit9lSQAb54gEB0r73S11ddCWHRS3H261qT/p5Fg\npwJsmg0zaFnAj+njZQaZN7DlwRXrowIwADAKBggqhkjOPQQDAgNIADBFAiEAwkVH\nJYFJX5Nh2U62RcoPCoNq0xm+DVgmVHsxztRu2eoCIFA2mPMO8lEKGeATYC6ufPwy\nyD5koUHTO2TzAhKSpH45\n-----END CERTIFICATE-----\n"

cd $HOME
# write all into the arcuino-connector.cfg
# id=containtel:1061e580-d2f2-4c06-9e48-51eb05ba700d
# url=a19g5nbe27wn47.iot.us-east-1.amazonaws.com
# http_proxy=
# https_proxy=
# all_proxy=
# authurl=https://hydra.arduino.cc
# apiurl=https://api2.arduino.cc

rm -f arduino-connector.cfg
echo "http_proxy=" >> arduino-connector.cfg
echo "https_proxy=" >> arduino-connector.cfg
echo "all_proxy=" >> arduino-connector.cfg
echo "authurl=https://hydra.arduino.cc" >> arduino-connector.cfg
echo "apiurl=https://api2.arduino.cc" >> arduino-connector.cfg
echo "id=$1" >> arduino-connector.cfg
echo "url=$2" >> arduino-connector.cfg

# write certificate in  certificate.pem
echo "$3" > certificate.pem
sudo chown $USER certificate.pem
sed -i 's/\\n/\n/g' certificate.pem

# use connector to test MQTT and register the device
sudo -E ./arduino-connector -configure -config ./arduino-connector.cfg

# start the service to see if everything works after systemd cleanup
sudo -E ./arduino-connector -install
sudo systemctl start ArduinoConnector
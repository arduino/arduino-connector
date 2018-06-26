#!/bin/bash
# call connector step `provision` and extract the csr
chmod +x arduino-connector
sudo -E ./arduino-connector -provision
# returns 
# Version: 2.0.103
# Generate private key
# Generate csr
# -----BEGIN CERTIFICATE REQUEST-----\nMIIBeTCCAR4CAQAwgY0xCzAJBgNVBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRl\nMQ8wDQYDVQQHEwZNeUNpdHkxFDASBgNVBAoTC0NvbXBhbnkgTHRkMQswCQYDVQQL\nEwJJVDEUMBIGA1UEAxMLZXhhbXBsZS5jb20xHzA
# dBgkqhkiG9w0BCQEMEHRlc3RA\nZXhhbXBsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBAATjUzWLt/X6Z8lZ\nJLEgjit9lSQAb54gEB0r73S11ddCWHRS3H261qT/p5FgpwJsmg0zaFnAj+njZQaZ\nN7DlwRXroC4wLAYJKoZIhvcNAQkOMR8wHTAbBgNVHREEFDASg
# RB0ZXN0QGV4YW1w\nbGUuY29tMAoGCCqGSM49BAMCA0kAMEYCIQDadwwf+47GM5+5zi4/Ujhh+d9jKIM4\ne3EORJURK2mdSAIhAKgxVnTZ28gn8OZTwRWomlPNXZ9fwjdbrvX7IibPm0sM\n-----END CERTIFICATE REQUEST-----\n


# create the device via iot-api
curl 'https://api2.arduino.cc/iot/v1/devices' -X PUT -H 'Pragma: no-cache' -H 'Origin: https://create-intel.arduino.cc' -H 'Accept-Encoding: gzip, deflate, br' -H 'Accept-Language: en-US,en;q=0.9' -H 'Authorization: Bearer gpEqK5tj-9s2CniMutSjeXN1ZkkL4JQbCUFC-O-sHME.hZsmoJmam8E5IJWKs4NBB1OaUiK0qDF50Vp6zpjdf3E' -H 'Content-Type: application/json;charset=UTF-8' -H 'Accept: application/json, text/plain, */*' -H 'Cache-Control: no-cache' -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36' -H 'Connection: keep-alive' -H 'Referer: https://create-intel.arduino.cc/getting-started/intel-platforms' --data-binary '{"name":"cURL-device","type":"generic"}' --compressed | jq

# returns 
# {
#   "href": "/iot/v1/devices/containtel:1061e580-d2f2-4c06-9e48-51eb05ba700d",
#   "id": "containtel:1061e580-d2f2-4c06-9e48-51eb05ba700d",
#   "name": "cURL-device",
#   "user_id": "4f80b0dd-db91-46c7-96c7-87871d296721"
# }


# connect the device via iot-api (that means obtain aws-iot endpoint)
curl 'https://api2.arduino.cc/iot/v1/devices/connect' -X POST -H 'Pragma: no-cache' -H 'Origin: https://create-intel.arduino.cc' -H 'Accept-Encoding: gzip, deflate, br' -H 'Accept-Language: en-US,en;q=0.9' -H 'Authorization: Bearer gpEqK5tj-9s2CniMutSjeXN1ZkkL4JQbCUFC-O-sHME.hZsmoJmam8E5IJWKs4NBB1OaUiK0qDF50Vp6zpjdf3E' -H 'Accept: application/json, text/plain, */*' -H 'Cache-Control: no-cache' -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36' -H 'Connection: keep-alive' -H 'Referer: https://create-intel.arduino.cc/getting-started/intel-platforms' -H 'Content-Length: 0' --compressed | jq
# returns 
# {
#   "signed_websocket": "wss://a19g5nbe27wn47.iot.us-east-1.amazonaws.com/mqtt?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ASIAIZW64KX73EBOWLNQ%2F20180626%2Fus-east-1%2Fiotdevicegateway%2Faws4_request&X-Amz-Date=20180626T123020Z&X-Amz-Security-Token=FQoDYXdzEKb%2F%2F%2F%2F%2F%2F%2F%2F%2F%2FwEaDNFQ8AJTOlNJ4cRzWSL%2BAd8dGWOxfYYZ7%2BRtvgDJ4L4wSm%2BkOv2W4A0dtag16IA5Jtq4c4q5%2FlabF1ysdQM7%2BC3YFt%2Fol4C5A2oEcQJzp0fU8hJ4SUByacIwyOQ%2B6FIYGA7n7kO2VKYjVexTEUlc3P6WwNq3Gw6jxznZxrwwO0F8%2F0Vy4PSLkcwLE9IywvzMjbcN0rRxy9kjkP%2BTopMv%2F5JjxQRVfrDD39cnQzs3MUzPWa3kWofHKheVKDGpN56a%2BuLPJ0%2F3YLgnAhHCd2g%2FRlABCp58EV3Dy4P%2BnuzqmzNZAdtyr%2FjS6IU%2FEJ9lmP8H8mGRdtpCy5aVAk6fOu5iF0Ud%2FucXZpuSVZdiq0NUKNzjyNkF&X-Amz-Signature=787741e4091095476d32f0cf7ac67404a1b59f51c6c2d66f7df4fb41f8924a59&X-Amz-SignedHeaders=host",
#   "url": "a19g5nbe27wn47.iot.us-east-1.amazonaws.com"
# }


# update the device via iot-api providing the csr to get the certificate
curl 'https://api2.arduino.cc/iot/v1/devices/containtel:703a0310-6166-4ed9-b8a0-27ca976e3b23' -X POST -H 'Pragma: no-cache' -H 'Origin: https://create-intel.arduino.cc' -H 'Accept-Encoding: gzip, deflate, br' -H 'Accept-Language: en-US,en;q=0.9' -H 'Authorization: Bearer gpEqK5tj-9s2CniMutSjeXN1ZkkL4JQbCUFC-O-sHME.hZsmoJmam8E5IJWKs4NBB1OaUiK0qDF50Vp6zpjdf3E' -H 'Accept: application/json, text/plain, */*' -H 'Cache-Control: no-cache' -H 'User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36' -H 'Connection: keep-alive' -H 'Referer: https://create-intel.arduino.cc/getting-started/intel-platforms' --data-binary '{"csr":"-----BEGIN CERTIFICATE REQUEST-----\nMIIBdzCCAR4CAQAwgY0xCzAJBgNVBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRl\nMQ8wDQYDVQQHEwZNeUNpdHkxFDASBgNVBAoTC0NvbXBhbnkgTHRkMQswCQYDVQQL\nEwJJVDEUMBIGA1UEAxMLZXhhbXBsZS5jb20xHzAdBgkqhkiG9w0BCQEMEHRlc3RA\nZXhhbXBsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAAQ5X5kA9YIS9fut\nkPlHvtrUTy1oBI0Z0PWU34rJ3csRp+oCblmk7TtYpqzGuETY3KHoi0Vplsf9qrkw\ncaC1z5WGoC4wLAYJKoZIhvcNAQkOMR8wHTAbBgNVHREEFDASgRB0ZXN0QGV4YW1w\nbGUuY29tMAoGCCqGSM49BAMCA0cAMEQCIAFHn1ErU3HdEMDHya6iKkXVyuNovVcg\nToeMA94/rKNxAiBqQQUdd5xRnaYUqnfz4wqjQQJTYXbCaCGRlps4yQN3Hg==\n-----END CERTIFICATE REQUEST-----\n"}'  --compressed | jq
# returns
# {
#   "certificate": "-----BEGIN CERTIFICATE-----\nMIIB6zCCAZGgAwIBAgIRALMj3Il+l86W5JZM3gYIKoZIzj0EAwIwfjEL\nMAkGA1UEBhMCVVMxFzAVBgNVBAoTDkFyZHVpbm8gTExDIFVTMQswCQYDVQQLEwJJ\nVDFJMEcGA1UEAxNANTU5Nzg0MWU2OTI4ODA3YzhmZWE4ZGY3YzljZGM3OWRi\nMWM5MTY5MGQ4YTUwNmI5OTE5NzU4YTU0Y2Q3OTAeFw0xODA2MjYxMjAwMDBaFw00\nOTA2MjYxMjAwMDBaMGwxCzAJBgNVBAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRl\nMQ8wDQYDVQQHEwZNeUNpdHkxFDASBgNVBAoTC0NvbXBhbnkgTHRkMQswCQYDVQQL\nEwJJVDEUMBIGA1UEAxMLZXhhbXBsZS5jb20wWTATBgcqhkjOPQIBBggqhkjOPQMB\nBwNCAATjUzWLt/X6Z8lZJLEgjit9lSQAb54gEB0r73S11ddCWHRS3H261qT/p5Fg\npwJsmg0zaFnAj+njZQaZN7DlwRXrowIwADAKBggqhkjOPQQDAgNIADBFAiEAwkVH\nJYFJX5Nh2U62RcoPCoNq0xm+DVgmVHsxztRu2eoCIFA2mPMO8lEKGeATYC6ufPwy\nyD5koUHTO2TzAhKSpH45\n-----END CERTIFICATE-----\n",
#   "compressed": {
#     "not_after": "2049-06-26T12:00:00Z",
#     "not_before": "2018-06-26T12:00:00Z",
#     "serial": "b323dc897e97ce96e4964cde510a0386",
#     "signature": "c245472581495f9361d94eb645ca0f0a836ad319be0d5826547ed46ed9ea503698f30ef2510a19e013602eae7cfc32c83e64a141d33b64f3021292a47e39"
#   },
#   "href": "/iot/v1/devices/containtel:1061e580-d2f2-4c06-9e48-51eb05ba700d",
#   "id": "containtel:1061e580-d2f2-4c06-9e48-51eb05ba700d",
#   "name": "cURL-device",
#   "type": "generic",
#   "user_id": "4f80b0dd-db91-46c7-96c7-87871d296721"
# }


# write all into the arcuino-connector.cfg
# id=containtel:1061e580-d2f2-4c06-9e48-51eb05ba700d
# url=a19g5nbe27wn47.iot.us-east-1.amazonaws.com
# http_proxy=
# https_proxy=
# all_proxy=
# authurl=https://hydra.arduino.cc
# apiurl=https://api2.arduino.cc

# write certificate in  certificate.pem

# use connector to test MQTT and register the device
sudo -E ./arduino-connector -configure -config ./arduino-connector.cfg

# start the service to see if everything works after systemd cleanup
sudo rm -f /etc/systemd/system/ArduinoConnector.service
sudo -E ./arduino-connector -install
sudo systemctl start ArduinoConnector
#!/bin/bash
set -e
# teardown 
source cert_arn.sh
echo ${CERT_ARN}
CERT_ID=${CERT_ARN##*/}
aws iot --profile arduino detach-policy --policy-name "DevicePolicy" --target ${CERT_ARN}
aws iot --profile arduino detach-thing-principal --thing-name "testThingVagrant" --principal ${CERT_ARN}
aws iot --profile arduino delete-thing --thing-name "testThingVagrant"
aws iot --profile arduino update-certificate --certificate-id ${CERT_ID} --new-status INACTIVE
aws iot --profile arduino delete-certificate --certificate-id ${CERT_ID} 
echo "cleanup files..."
rm -f cert.pem privateKey.pem publicKey.pem rootCA.pem cert_arn.sh
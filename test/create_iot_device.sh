#!/bin/bash
set -e


RAW_CERT_ARN=$(aws iot --profile arduino create-keys-and-certificate --set-as-active --certificate-pem-outfile cert.pem --public-key-outfile publicKey.pem --private-key-outfile privateKey.pem --query 'certificateArn')
temp="${RAW_CERT_ARN%\"}"
CERT_ARN="${temp#\"}"
RAW_IOT_ENDPOINT=$(aws iot --profile arduino describe-endpoint --query 'endpointAddress')
temp="${RAW_IOT_ENDPOINT%\"}"
IOT_ENDPOINT="${temp#\"}"

cat > cert_arn.sh <<EOL
#!/bin/bash
export CERT_ARN=${CERT_ARN}
export IOT_ENDPOINT=${IOT_ENDPOINT}
EOL
chmod +x cert_arn.sh


aws iot --profile arduino create-thing --thing-name "testThingVagrant"
aws iot --profile arduino attach-principal-policy --principal ${CERT_ARN} --policy-name "DevicePolicy"
aws iot --profile arduino attach-thing-principal --thing-name "testThingVagrant" --principal ${CERT_ARN}



wget --quiet -O rootCA.pem "https://www.symantec.com/content/en/us/enterprise/verisign/roots/VeriSign-Class%203-Public-Primary-Certification-Authority-G5.pem"
# now you can use: 
#  mosquitto_sub --cert cert.pem --key privateKey.pem --cafile rootCA.pem -h xxxx.iot.us-east-1.amazonaws.com -p 8883 -t '$aws/things/devops-test:75b87fe3-169d-4603-a018-7fde9c667850/#' -d -i testThingVagrant

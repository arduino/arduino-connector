#!/bin/bash
set -e

# upload arduino-connector-binary and generate installer
aws --profile arduino s3 cp ../scripts/arduino-connector-dev.sh s3://arduino-tmp/arduino-connector.sh
SHELL_INSTALLER=$(aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector.sh --expires-in $(expr 3600 \* 72))
#use this link i the wget of the getting started script
aws --profile arduino s3 cp ../arduino-connector s3://arduino-tmp/
ARDUINO_CONNECTOR=$(aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector  --expires-in $(expr 3600 \* 72))
# use the output as the argument of arduino-connector-dev.sh qhen launching getting started script:

cat >ui_gen_install.sh <<EOL
#!/bin/bash

# this device was created for the test user in devices-dev environment
export AUTHURL='https://hydra-dev.arduino.cc/'
export APIURL='https://api-dev.arduino.cc'
export CHECK_RO_FS=true
export ENV_VARS_TO_LOAD="HDDL_INSTALL_DIR=/opt/intel/computer_vision_sdk/inference_engine/external/hddl/,ENV_TEST_PATH=/tmp"
export id=devops-test:c4d6adc7-a2ca-43ec-9ea6-20568bf407fc

wget -O install.sh "${SHELL_INSTALLER}"
chmod +x install.sh
./install.sh "${ARDUINO_CONNECTOR}"
EOL

chmod +x ui_gen_install.sh

cat >setup_host_test_env.sh <<EOL
#!/bin/bash

EOL

chmod +x setup_host_test_env.sh
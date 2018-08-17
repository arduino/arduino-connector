
## Generate temporary installer script
```
aws-google-auth -p arduino       
go build -ldflags "-X main.version=2.0.21" github.com/arduino/arduino-connector
aws --profile arduino s3 cp arduino-connector-dev.sh s3://arduino-tmp/arduino-connector.sh
aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector.sh --expires-in $(expr 3600 \* 72)
#use this link i the wget of the getting started script
aws --profile arduino s3 cp arduino-connector s3://arduino-tmp/
aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector  --expires-in $(expr 3600 \* 72)
# use the output as the argument of arduino-connector-dev.sh qhen launching getting started script:

export id=containtel:a4ae70c4-b7ff-40c8-83c1-1e10ee166241
wget -O install.sh <aws signed link dev-sh>
chmod +x install.sh
./install.sh <aws signed link dev connector>

```

i.e
```
export id=containtel:a4ae70c4-b7ff-40c8-83c1-1e10ee166241
wget -O install.sh  "https://arduino-tmp.s3.amazonaws.com/arduino-connector.sh?AWSAccessKeyId=ASIAJJFZDTIGHJCWMGQA&Expires=1529771794&x-amz-security-token=FQoDYXdzEBoaDD8duZwY18MeYFd3CyLPAjxH7ijRrTBwduS9r8Dqm06%2BT%2B6p57cOU4I1Bn3d09lMVjPi4dhNQboAxLnYSI%2BNqxUo%2BbgNDxRbIVxzgvGWQHw7Seepjniy%2FvCKpR7DuxyNe%2B5DxA15O1fGZDQkqadxlky5jkXk1Vn9TBtGa4NCRMgIoatRBtkHI7XKpouWNYhh2jYo7ezeDRQO3m1WR7WieqVlh%2BdscL0NevGGMOh3MYf5Wsm069GuA31FmTslp3SaChf7Mq7uOI5X9XIu%2B9kcWnxXoo7dMCk5Ixq5WLkB%2BUlTt6iL4bxK7FKdlT%2FUsf5DSfBcCGwcyI2nBuFB6yjPeS5AAm0ZUU6DaEd9KUc8Fxq9M1tEQ3DnjGnKZcbaOU%2FGWw7bnOPhLcl6eiNIOtZxsvZ4MCTY3YUnO4rna4fVNScjIqMwNdb8psFarGH1Gn0e4DRNt22LFshjGZdNi01RKI%2BFqtkF&Signature=jI00Smxp33Y72ijdRJsXMIYx9h0%3D"
chmod +x install.sh
./install.sh "https://arduino-tmp.s3.amazonaws.com/arduino-connector?AWSAccessKeyId=ASIAJJFZDTIGHJCWMGQA&Expires=1529771799&x-amz-security-token=FQoDYXdzEBoaDD8duZwY18MeYFd3CyLPAjxH7ijRrTBwduS9r8Dqm06%2BT%2B6p57cOU4I1Bn3d09lMVjPi4dhNQboAxLnYSI%2BNqxUo%2BbgNDxRbIVxzgvGWQHw7Seepjniy%2FvCKpR7DuxyNe%2B5DxA15O1fGZDQkqadxlky5jkXk1Vn9TBtGa4NCRMgIoatRBtkHI7XKpouWNYhh2jYo7ezeDRQO3m1WR7WieqVlh%2BdscL0NevGGMOh3MYf5Wsm069GuA31FmTslp3SaChf7Mq7uOI5X9XIu%2B9kcWnxXoo7dMCk5Ixq5WLkB%2BUlTt6iL4bxK7FKdlT%2FUsf5DSfBcCGwcyI2nBuFB6yjPeS5AAm0ZUU6DaEd9KUc8Fxq9M1tEQ3DnjGnKZcbaOU%2FGWw7bnOPhLcl6eiNIOtZxsvZ4MCTY3YUnO4rna4fVNScjIqMwNdb8psFarGH1Gn0e4DRNt22LFshjGZdNi01RKI%2BFqtkF&Signature=BTsZzRhHnf%2Fl%2BWsXfJ9MB1ir318%3D"

```
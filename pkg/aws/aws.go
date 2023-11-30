package aws

import "github.com/aws/aws-sdk-go-v2/service/ec2"

type Aws struct {
	svc *ec2.Client
}

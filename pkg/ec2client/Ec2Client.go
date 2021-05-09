package ec2client

import (
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"log"
	"os"
)

var (
	ec2Client *ec2.Client
)

func Initialize(awsProfile string) error {
	if awsProfile != "" {
		os.Setenv("AWS_PROFILE", awsProfile)

		// --profile command line argument is stronger than those environment variables
		//
		os.Unsetenv("AWS_ACCESS_KEY_ID")
		os.Unsetenv("AWS_SECRET_ACCESS_KEY")
		os.Unsetenv("AWS_DEFAULT_REGION")
	}

	var err error

	ec2Client, err = initializeEc2Client()
	return err
}

func initializeAwsConfig() (aws.Config, error) {
	config, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		log.Printf("Unable to load SDK config, %v\n", err)
		return config, err
	}

	creds, err := config.Credentials.Retrieve(context.TODO())

	if err != nil {
		log.Printf("Unable to get credentials %v", err)
		return config, err
	}

	log.Printf("AWS Config initialized, region: %v, access key: %v\n", config.Region, creds.AccessKeyID)
	return config, nil
}

func initializeEc2Client() (*ec2.Client, error) {
	awsConfig, err := initializeAwsConfig()

	if err != nil {
		return nil, err
	}

	return ec2.NewFromConfig(awsConfig), nil
}

func getSecurityGroupById(input *ec2.DescribeSecurityGroupsInput) (*types.SecurityGroup, error) {
	output, err := ec2Client.DescribeSecurityGroups(context.TODO(), input)

	if err != nil {
		return nil, err
	}

	if len(output.SecurityGroups) == 0 {
		return nil, nil
	}

	if len(output.SecurityGroups) > 1 {
		log.Printf("Excpecting to get exactly 1 security group, got %v", len(output.SecurityGroups))
		return nil, errors.New("DescribeSecurityGroups bad Results")
	}

	return &output.SecurityGroups[0], nil
}

func GetSecurityGroupById(securityGroupId string) (*types.SecurityGroup, error) {
	input := ec2.DescribeSecurityGroupsInput{}

	input.GroupIds = []string{securityGroupId}

	return getSecurityGroupById(&input)
}

func GetSecurityGroupByFilter(filterKey string, filterValue string) (*types.SecurityGroup, error) {
	input := ec2.DescribeSecurityGroupsInput{}

	input.Filters = []types.Filter{
		types.Filter{
			Name:   &filterKey,
			Values: []string{filterValue},
		},
	}

	return getSecurityGroupById(&input)
}

func AuthorizeSecurityGroupIngress(securityGroupId string, port int32, cidr string, description string) error {
	input := ec2.AuthorizeSecurityGroupIngressInput{}

	protocol := "tcp"

	input.GroupId = &securityGroupId
	input.IpPermissions = []types.IpPermission{
		types.IpPermission{
			IpProtocol: &protocol,
			FromPort:   port,
			ToPort:     port,
			IpRanges: []types.IpRange{
				types.IpRange{
					CidrIp:      &cidr,
					Description: &description,
				},
			},
		},
	}

	_, err := ec2Client.AuthorizeSecurityGroupIngress(context.TODO(), &input)

	return err
}

func RevokeSecurityGroupIngress(securityGroupId string, port int32, cidr string) error {
	input := ec2.RevokeSecurityGroupIngressInput{}

	protocol := "tcp"

	input.GroupId = &securityGroupId
	input.IpPermissions = []types.IpPermission{
		types.IpPermission{
			IpProtocol: &protocol,
			FromPort:   port,
			ToPort:     port,
			IpRanges: []types.IpRange{
				types.IpRange{
					CidrIp: &cidr,
				},
			},
		},
	}

	_, err := ec2Client.RevokeSecurityGroupIngress(context.TODO(), &input)

	return err
}

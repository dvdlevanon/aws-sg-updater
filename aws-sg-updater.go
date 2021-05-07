package main

import (
	"strings"
	"flag"
	"strconv"
	"fmt"
	"log"
	"errors"
	"os"
	"os/user"
	"time"
	"io/ioutil"
	"path/filepath"
	"aws-sg-updater/pkg/ec2client"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/rdegges/go-ipify"
	"github.com/kirsle/configdir"
	"github.com/google/uuid"
)

type EntryDetails struct {
	Port int32
	Cidr string
	Uuid string
	Description string
}

var (
	awsProfileConnectParam = flag.String("profile", "", "AWS Cli Profile to use")
	securityGroupIdParam = flag.String("security-group-id", "", "Security group id")
	securityGroupNameParam = flag.String("security-group-name", "", "Security group name")
	useNameTag = flag.Bool("use-name-tag", false, "Use Name tag instead of security group name, relevant when --security-group-name specified")
	portParam = flag.String("port", "", "Security group port number")
	dateFormat = flag.String("date-format", "2006-01-02", "Go format of the date to put in the description")
)

func getAwsConnectProfile() string {
	profile := *awsProfileConnectParam

	if profile != "" {
		return profile
	}

	profile = os.Getenv("AWS_PROFILE")

	if profile != "" {
		return profile
	}

	return ""
}

func initialize(awsProfile string) bool {
	return ec2client.Initialize(awsProfile) == nil
}

func updateSecurityGroup() {
	details, err := getEntryDetails()
	
	if err != nil {
		return
	}
	
	securityGroup, err := getSecurityGroup(*securityGroupIdParam, *securityGroupNameParam, *useNameTag)
	
	if err != nil {
		return
	}
	
	log.Printf("Updating %v with %v:%v - %v", *securityGroup.GroupName, details.Cidr, details.Port, details.Uuid)
	
	if cleanupOldEntryIfExists(securityGroup, details) {
		createNewEntry(securityGroup, details)
	}
}

func getSecurityGroup(securityGroupId string, securityGroupName string, useNameTag bool) (*types.SecurityGroup, error) {
	securityGroup, err := getSecurityGroupSilent(securityGroupId, securityGroupName, useNameTag)
	
	if err != nil {
		log.Printf("Error getting security group (provided id: %v) (provided name: %v) (use tag? %v) %v", 
			securityGroupId, securityGroupName, useNameTag, err)
	}
	
	if securityGroup == nil {
		log.Printf("Security group not found (provided id: %v) (provided name: %v) (use tag? %v)",
			securityGroupId, securityGroupName, useNameTag)
		return nil, errors.New("Security group not found")
	}
	
	return securityGroup, err
}

func getSecurityGroupSilent(securityGroupId string, securityGroupName string, useNameTag bool) (*types.SecurityGroup, error) {
	if securityGroupId != "" {
		return ec2client.GetSecurityGroupById(securityGroupId)
	} else if securityGroupName != "" {
		return getSecurityGroupByName(securityGroupName, useNameTag)
	} else {
		return nil, errors.New("Neither security group id nor security group name provided")
	}
}

func getSecurityGroupByName(securityGroupName string, useNameTag bool) (*types.SecurityGroup, error) {
	if useNameTag {
		return ec2client.GetSecurityGroupByFilter("tag:Name", securityGroupName)
	} else {
		return ec2client.GetSecurityGroupByFilter("group-name", securityGroupName)
	}
}

func getEntryDetails() (EntryDetails, error) {
	port, err := strconv.Atoi(*portParam)
	
	if err != nil {
		return EntryDetails{}, err
	}
	
	cidr, err := getCidr()
	
	if err != nil {
		return EntryDetails{}, err
	}
	
	uuid, err := getPersistedUuid()
	
	if err != nil {
		return EntryDetails{}, err
	}
	
	description, err := buildDescription(uuid, port)
	
	if err != nil {
		return EntryDetails{}, err
	}
	
	return EntryDetails {
		Port: int32(port),
		Cidr: cidr,
		Uuid: uuid,
		Description: description,
	}, nil
}

func getCidr() (string, error) {
	ip, err := ipify.GetIp()
	
	if err != nil {
		log.Printf("Couldn't get my IP address:", err)
		return "", err
	}
	
	return fmt.Sprintf("%s/32", ip), nil
}

func getPersistedUuid() (string, error) {
	uuidFile, err := getPersistedUuidFile()
	
	if err != nil {
		return "", err
	}
	
	if _, err := os.Stat(uuidFile); os.IsNotExist(err) {
		if err := initializeUuidFile(uuidFile); err != nil {
			return "", err
		}
	}
	
	return readUuidFile(uuidFile)
}

func getPersistedUuidFile() (string, error) {
	configPath := configdir.LocalConfig("aws-sg-updater")
	err := configdir.MakePath(configPath)
	
	if err != nil {
		log.Printf("Error getting UUID file %v", err)
		return "", err
	}
	
	return filepath.Join(configPath, "uuid"), nil
}

func initializeUuidFile(uuidFile string) error {
	file, err := os.Create(uuidFile)
	
	if err != nil {
		log.Printf("Error openning UUID file for writing %v, %v", uuidFile, err)
		return err
	}
	
	defer file.Close()
	file.WriteString(uuid.NewString())
	return nil
}

func readUuidFile(uuidFile string) (string, error) {
	data, err := ioutil.ReadFile(uuidFile)
	
	if err != nil {
		log.Printf("Error openning UUID file %v, %v", uuidFile, err)
		return "", err
	}
	
	return string(data), nil
}

func buildDescription(uuid string, port int) (string, error) {
	formattedDate := getFormattedDate(*dateFormat)
	username, err := getCurrentUserName()
	
	if err != nil {
		return "", err
	}
	
	return fmt.Sprintf("%s %s port:%d - %s", username, formattedDate, port, uuid), nil
}

func getFormattedDate(dateFormat string) string {
	currentTime := time.Now()
	return currentTime.Format("2006-01-02")
}

func getCurrentUserName() (string, error) {
	user, err := user.Current()
    
    if err != nil {
        log.Printf("Error getting current user name %v", err)
        return "", err
    }

	return user.Username, nil
}

func cleanupOldEntryIfExists(securityGroup *types.SecurityGroup, details EntryDetails) bool {
	oldCidr := findOldCidr(securityGroup, details)
	
	if oldCidr == "" {
		return true
	}
	
	if oldCidr == details.Cidr {
		log.Printf("Cidr is up to date.")
		return false
	}
	
	err := ec2client.RevokeSecurityGroupIngress(
		*securityGroup.GroupId,
		details.Port,
		oldCidr)
	
	if err != nil {
		log.Printf("Error revoking to %v %v %v", *securityGroup.GroupId, details, err)
		return true
	}
	
	log.Printf("Successfully revoked old Cidr %v", oldCidr)
	return true
}

func findOldCidr(securityGroup *types.SecurityGroup, details EntryDetails) string {
	for _, inPermission := range securityGroup.IpPermissions {
		if inPermission.FromPort != details.Port {
			continue
		}
		
		if inPermission.ToPort != details.Port {
			continue
		}
		
		for _, ipRange := range inPermission.IpRanges {
			if ipRange.Description == nil {
				continue
			}
			
			if ipRange.CidrIp == nil {
				continue
			}
						
			if strings.Contains(*ipRange.Description, details.Uuid) {
				return *ipRange.CidrIp
			}
		}
	}
	
	return ""
}

func createNewEntry(securityGroup *types.SecurityGroup, details EntryDetails) {
	err := ec2client.AuthorizeSecurityGroupIngress(
		*securityGroup.GroupId,
		details.Port,
		details.Cidr,
		details.Description)
	
	if err != nil {
		log.Printf("Error authorizing to %v %v %v", *securityGroup.GroupId, details, err)
		return
	}
	
	log.Printf("Successfully added authorize role")
}

func main() {
	flag.Parse()
	
	awsProfile := getAwsConnectProfile()
	
	if !initialize(awsProfile) {
		return
	}
	
	updateSecurityGroup()
}

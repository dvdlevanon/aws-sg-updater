
# Title

A command line utility which helps to add an ingress policy to an AWS security group. The policy have the CIDR public IP of the calling process, and a single port specified in the command line.


## Installation 

```bash 
curl -L -o aws-sg-updater https://github.com/dvdlevanon/aws-sg-updater/releases/download/0.0.1/aws-sg-updater-0.0.1-x68_64
install -t /usr/local/bin aws-sg-updater
```

## Building from source

Install aws-sg-updater from source

```bash
git clone https://github.com/dvdlevanon/aws-sg-updater.git
go build aws-sg-updater.go
./aws-sg-updater
```

## Usage/Examples

### Prerequisite
Make sure you have some sort of AWS credentials, profile, environment variables, aws roles or whatever. Read here for more information: https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html

### Add or update a Port using secuirty group name
```bash
aws-sg-updater --profile <aws_profile> --security-group-name default --port 22
```

### Add or update a Port using secuirty group id
```bash
aws-sg-updater --profile <aws_profile> --security-group-id sg-XXXXXXXX --port 22
```

## How the update work?

AWS doesn't support updating an ingress role in the security group, hence we have to remove the old one and add a new one. We use a machine specific UUID in order to detect old entries added from the same machine. The first time aws-sg-updater is run, it generate a UUID and store it OS specifc configuration file.

  
## Authors

- [@dvdlevanon](https://www.github.com/dvdlevanon)

  
## Contributing

Contributions are always welcome!
[GPL-3](https://choosealicense.com/licenses/gpl-3.0//)

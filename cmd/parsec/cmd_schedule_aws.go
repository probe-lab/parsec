package main

import (
	"fmt"

	"golang.org/x/sync/errgroup"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dennis-tra/parsec/pkg/parsec"
	"github.com/guseggert/clustertest/cluster/aws"
	"github.com/guseggert/clustertest/cluster/basic"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/dennis-tra/parsec/pkg/config"
)

var ScheduleAWSCommand = &cli.Command{
	Name:   "aws",
	Action: ScheduleAWSAction,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "nodeagent",
			Usage:   "path to the nodeagent binary",
			Value:   "/home/parsec/nodeagent", // correct if you use the default docker image
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_NODEAGENT_BIN"},
		},
		&cli.StringSliceFlag{
			Name:    "regions",
			Usage:   "the AWS regions to use, if using an AWS cluster",
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_REGIONS"},
		},
		&cli.StringFlag{
			Name:    "instance-type",
			Usage:   "the EC2 instance type to run the experiment on",
			Value:   "t2.micro",
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_INSTANCE_TYPE"},
		},
		&cli.StringSliceFlag{
			Name:    "key-names",
			Usage:   "the SSH key pair names in each region",
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_KEY_NAMES"},
		},
		&cli.StringSliceFlag{
			Name:    "public-subnet-ids",
			Usage:   "The public subnet IDs to run the cluster in",
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_PUBLIC_SUBNET_IDS"},
		},
		&cli.StringSliceFlag{
			Name:    "instance-profile-arns",
			Usage:   "The instance profiles to run the Kubo nodes with",
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_INSTANCE_PROFILE_ARNS"},
		},
		&cli.StringSliceFlag{
			Name:    "instance-security-group-ids",
			Usage:   "The security groups of the Kubo instances",
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_SECURITY_GROUP_IDS"},
		},
		&cli.StringSliceFlag{
			Name:    "s3-bucket-arns",
			Usage:   "The S3 buckets where the nodeagent binaries are stored",
			EnvVars: []string{"PARSEC_SCHEDULE_AWS_S3_BUCKET_ARNS"},
		},
	},
}

func ScheduleAWSAction(c *cli.Context) error {
	log.Infoln("Starting Parsec AWS scheduler...")

	conf, err := config.DefaultScheduleAWSConfig.Apply(c)
	if err != nil {
		return fmt.Errorf("parse command line flags: %w", err)
	}

	nodesChan := make(chan *parsec.Node, len(conf.Regions))

	errg := errgroup.Group{}
	for i, region := range conf.Regions {
		i := i
		region := region
		errg.Go(func() error {
			cl := aws.NewCluster().
				WithNodeAgentBin(conf.NodeAgent).
				WithSession(session.Must(session.NewSession(&awssdk.Config{Region: &region}))).
				WithPublicSubnetID(conf.PublicSubnetIDs[i]).
				WithInstanceProfileARN(conf.InstanceProfileARNs[i]).
				WithInstanceSecurityGroupID(conf.InstanceSecurityGroupIDs[i]).
				WithS3BucketARN(conf.S3BucketARNs[i]).
				WithInstanceType(conf.InstanceType).
				WithKeyName(conf.KeyNames[i])

			pc := parsec.NewCluster(basic.New(cl).Context(c.Context), region, conf.InstanceType, conf.ServerHost, conf.ServerPort)

			log.Infoln("Initializing aws node")
			n, err := pc.NewNode(i)
			if err != nil {
				return fmt.Errorf("new aws node: %w", err)
			}

			nodesChan <- n

			return nil
		})
	}

	if err = errg.Wait(); err != nil {
		return fmt.Errorf("wait for new nodes: %w", err)
	}

	nodes := make([]*parsec.Node, len(conf.Regions))
	for i := 0; i < len(conf.Regions); i++ {
		nodes[i] = <-nodesChan
	}
	close(nodesChan)

	return ScheduleAction(c, nodes)
}

package main

import (
	"encoding/json"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/iam"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/container"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "k8s-infra")
		cfgGlobal := config.New(ctx, "")

		cloud := cfg.Get("cloud")
		if cloud == "" {
			cloud = "aws"
		}
		if cloud != "aws" && cloud != "gcp" {
			cloud = "aws"
		}

		projectName := cfg.Get("projectName")
		if projectName == "" {
			projectName = "k8s-infra"
		}
		environment := cfg.Get("environment")
		if environment == "" {
			environment = "dev"
		}
		clusterName := cfg.Get("clusterName")
		if clusterName == "" {
			clusterName = "k8s-infra-cluster"
		}
		clusterVersion := cfg.Get("clusterVersion")
		if clusterVersion == "" {
			clusterVersion = "1.29"
		}
		desiredSize := cfg.GetInt("nodeDesiredSize")
		if desiredSize == 0 {
			desiredSize = 2
		}

		commonTags := pulumi.StringMap{
			"Environment": pulumi.String(environment),
			"Project":     pulumi.String(projectName),
		}

		if cloud == "aws" {
			return deployEKS(ctx, deployEKSConfig{
				cfg:          cfg,
				cfgGlobal:    cfgGlobal,
				projectName:  projectName,
				environment:  environment,
				clusterName:  clusterName,
				clusterVersion: clusterVersion,
				desiredSize:  desiredSize,
				commonTags:   commonTags,
			})
		}

		return deployGKE(ctx, deployGKEConfig{
			cfg:            cfg,
			cfgGlobal:      cfgGlobal,
			projectName:    projectName,
			environment:    environment,
			clusterName:    clusterName,
			clusterVersion: clusterVersion,
			desiredSize:    desiredSize,
			commonTags:     commonTags,
		})
	})
}

type deployEKSConfig struct {
	cfg            *config.Config
	cfgGlobal      *config.Config
	projectName    string
	environment    string
	clusterName    string
	clusterVersion string
	desiredSize    int
	commonTags     pulumi.StringMap
}

func deployEKS(ctx *pulumi.Context, c deployEKSConfig) error {
	awsRegion := c.cfgGlobal.Require("aws:region")
	nodeInstanceType := c.cfg.Get("nodeInstanceType")
	if nodeInstanceType == "" {
		nodeInstanceType = "t3.medium"
	}
	vpcCidr := c.cfg.Get("vpcCidr")
	if vpcCidr == "" {
		vpcCidr = "10.0.0.0/16"
	}

	// --- VPC ---
	vpc, err := ec2.NewVpc(ctx, "vpc", &ec2.VpcArgs{
		CidrBlock:          pulumi.String(vpcCidr),
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		Tags: pulumi.StringMap{
			"Name": pulumi.String(c.projectName + "-vpc"),
			"Environment": pulumi.String(c.environment),
			"kubernetes.io/cluster/" + c.clusterName: pulumi.String("shared"),
		},
	})
	if err != nil {
		return err
	}

	igw, err := ec2.NewInternetGateway(ctx, "igw", &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags: pulumi.StringMap{
			"Name":        pulumi.String(c.projectName + "-igw"),
			"Environment": pulumi.String(c.environment),
		},
	})
	if err != nil {
		return err
	}

	azs := []string{awsRegion + "a", awsRegion + "b"}
	subnetCidrs := []string{"10.0.1.0/24", "10.0.2.0/24"}
	var subnetIds []pulumi.StringInput
	var subnets []*ec2.Subnet
	for i, az := range azs {
		sub, err := ec2.NewSubnet(ctx, "subnet-"+az, &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String(subnetCidrs[i]),
			AvailabilityZone:    pulumi.String(az),
			MapPublicIpOnLaunch: pulumi.Bool(true),
			Tags: pulumi.StringMap{
				"Name": pulumi.String(c.projectName + "-subnet-" + az),
				"Environment": pulumi.String(c.environment),
				"kubernetes.io/cluster/" + c.clusterName: pulumi.String("shared"),
				"kubernetes.io/role/elb":              pulumi.String("1"),
			},
		})
		if err != nil {
			return err
		}
		subnetIds = append(subnetIds, sub.ID())
		subnets = append(subnets, sub)
	}

	routeTable, err := ec2.NewRouteTable(ctx, "public-rt", &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Routes: ec2.RouteTableRouteArray{
			&ec2.RouteTableRouteArgs{
				CidrBlock: pulumi.String("0.0.0.0/0"),
				GatewayId: igw.ID(),
			},
		},
		Tags: pulumi.StringMap{
			"Name":        pulumi.String(c.projectName + "-public-rt"),
			"Environment": pulumi.String(c.environment),
		},
	})
	if err != nil {
		return err
	}

	for i, sub := range subnets {
		_, err = ec2.NewRouteTableAssociation(ctx, "rta-"+azs[i], &ec2.RouteTableAssociationArgs{
			SubnetId:     sub.ID(),
			RouteTableId: routeTable.ID(),
		})
		if err != nil {
			return err
		}
	}

	// --- IAM roles ---
	clusterRoleAssume, _ := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{"Effect": "Allow", "Principal": map[string]interface{}{"Service": "eks.amazonaws.com"}, "Action": "sts:AssumeRole"},
		},
	})
	clusterRole, err := iam.NewRole(ctx, "eks-cluster-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(clusterRoleAssume)),
		Tags:             c.commonTags,
	})
	if err != nil {
		return err
	}
	_, err = iam.NewRolePolicyAttachment(ctx, "eks-cluster-policy", &iam.RolePolicyAttachmentArgs{
		Role: clusterRole.Name, PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
	})
	if err != nil {
		return err
	}

	nodeRoleAssume, _ := json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{"Effect": "Allow", "Principal": map[string]interface{}{"Service": "ec2.amazonaws.com"}, "Action": "sts:AssumeRole"},
		},
	})
	nodeRole, err := iam.NewRole(ctx, "eks-node-role", &iam.RoleArgs{
		AssumeRolePolicy: pulumi.String(string(nodeRoleAssume)),
		Tags:             c.commonTags,
	})
	if err != nil {
		return err
	}
	for _, policy := range []string{"AmazonEKSWorkerNodePolicy", "AmazonEKS_CNI_Policy", "AmazonEC2ContainerRegistryReadOnly"} {
		_, err = iam.NewRolePolicyAttachment(ctx, "node-"+policy, &iam.RolePolicyAttachmentArgs{
			Role: nodeRole.Name, PolicyArn: pulumi.String("arn:aws:iam::aws:policy/"+policy),
		})
		if err != nil {
			return err
		}
	}

	// --- EKS cluster and node group ---
	cluster, err := eks.NewCluster(ctx, "cluster", &eks.ClusterArgs{
		Name:    pulumi.String(c.clusterName),
		Version: pulumi.String(c.clusterVersion),
		RoleArn: clusterRole.Arn,
		VpcConfig: &eks.ClusterVpcConfigArgs{
			SubnetIds:        pulumi.StringArray(subnetIds),
			PublicAccessCidrs: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
		},
		Tags: c.commonTags,
	})
	if err != nil {
		return err
	}

	_, err = eks.NewNodeGroup(ctx, "node-group", &eks.NodeGroupArgs{
		ClusterName:   cluster.Name,
		NodeGroupName: pulumi.String(c.clusterName + "-nodes"),
		NodeRoleArn:   nodeRole.Arn,
		SubnetIds:     pulumi.StringArray(subnetIds),
		InstanceTypes: pulumi.StringArray{pulumi.String(nodeInstanceType)},
		ScalingConfig: &eks.NodeGroupScalingConfigArgs{
			DesiredSize: pulumi.Int(c.desiredSize),
			MinSize:     pulumi.Int(1),
			MaxSize:     pulumi.Int(4),
		},
		Tags: c.commonTags,
	})
	if err != nil {
		return err
	}

	ctx.Export("cloud", pulumi.String("aws"))
	ctx.Export("vpcId", vpc.ID())
	ctx.Export("clusterName", cluster.Name)
	ctx.Export("clusterEndpoint", cluster.Endpoint)
	ctx.Export("updateKubeconfigCommand", cluster.Name.ApplyT(func(name string) string {
		return "aws eks update-kubeconfig --region " + awsRegion + " --name " + name
	}).(pulumi.StringOutput))
	return nil
}

type deployGKEConfig struct {
	cfg            *config.Config
	cfgGlobal      *config.Config
	projectName    string
	environment    string
	clusterName    string
	clusterVersion string
	desiredSize    int
	commonTags     pulumi.StringMap
}

func deployGKE(ctx *pulumi.Context, c deployGKEConfig) error {
	gcpProject := c.cfg.Get("gcpProject")
	if gcpProject == "" {
		gcpProject = c.cfgGlobal.Get("gcp:project")
	}
	if gcpProject == "" {
		gcpProject = c.cfg.Require("gcpProject")
	}

	gcpRegion := c.cfg.Get("gcpRegion")
	if gcpRegion == "" {
		gcpRegion = c.cfgGlobal.Get("gcp:region")
	}
	if gcpRegion == "" {
		gcpRegion = "us-central1"
	}

	nodeMachineType := c.cfg.Get("nodeMachineType")
	if nodeMachineType == "" {
		nodeMachineType = "e2-medium"
	}

	// --- GKE cluster (no default node pool; we add one below) ---
	cluster, err := container.NewCluster(ctx, "cluster", &container.ClusterArgs{
		Name:                   pulumi.String(c.clusterName),
		Location:               pulumi.String(gcpRegion),
		Project:                pulumi.String(gcpProject),
		InitialNodeCount:       pulumi.Int(1), // required when not using RemoveDefaultNodePool; we remove default pool
		MinMasterVersion:      pulumi.String(c.clusterVersion),
		RemoveDefaultNodePool:  pulumi.Bool(true),
		DeletionProtection:     pulumi.Bool(false),
		ResourceLabels: pulumi.StringMap{
			"environment": pulumi.String(c.environment),
			"project":     pulumi.String(c.projectName),
		},
	})
	if err != nil {
		return err
	}

	_, err = container.NewNodePool(ctx, "node-pool", &container.NodePoolArgs{
		Cluster:  cluster.Name,
		Location: pulumi.String(gcpRegion),
		Project:  pulumi.String(gcpProject),
		NodeCount: pulumi.Int(c.desiredSize),
		NodeConfig: &container.NodePoolNodeConfigArgs{
			MachineType: pulumi.String(nodeMachineType),
			OauthScopes: pulumi.StringArray{
				pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
			},
		},
		Autoscaling: &container.NodePoolAutoscalingArgs{
			MinNodeCount: pulumi.Int(1),
			MaxNodeCount: pulumi.Int(4),
		},
	})
	if err != nil {
		return err
	}

	ctx.Export("cloud", pulumi.String("gcp"))
	ctx.Export("clusterName", cluster.Name)
	ctx.Export("clusterEndpoint", cluster.Endpoint)
	ctx.Export("getCredentialsCommand", pulumi.All(gcpProject, gcpRegion, cluster.Name).ApplyT(func(args []interface{}) string {
		project := args[0].(string)
		region := args[1].(string)
		name := args[2].(string)
		return "gcloud container clusters get-credentials " + name + " --region " + region + " --project " + project
	}).(pulumi.StringOutput))
	return nil
}

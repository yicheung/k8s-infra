terraform {
  backend "s3" {
    bucket         = "k8s-infra-tfstate-bucket"
    key            = "ec2/terraform.tfstate"
    region         = "eu-west-1"
    dynamodb_table = "k8s-infra-tfstate-lock"
    encrypt        = true
  }
}

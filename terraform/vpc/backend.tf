terraform {
  backend "s3" {
    bucket         = "k8s-infra-tfstate-bucket"
    key            = "vpc/terraform.tfstate"
    region         = "eu-west-1"
    dynamodb_table = "k8s-infra-tfstate-lock"
    encrypt        = true
  }
}

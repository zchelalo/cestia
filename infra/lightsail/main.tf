terraform {
  required_version = ">= 1.10.0, < 2.0.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  profile             = var.aws_profile
  region              = var.aws_region
  allowed_account_ids = [var.expected_aws_account_id]
}

data "aws_caller_identity" "current" {}

locals {
  common_tags = merge(
    {
      Organization = var.organization
      Project      = var.project
      Environment  = var.environment
      CostCenter   = var.cost_center
      ManagedBy    = "terraform"
      Stack        = "lightsail-production"
    },
    var.tags,
  )

  firewall_rules = concat(
    length(var.ssh_allowed_cidrs) == 0 ? [] : [{
      protocol  = "tcp"
      from_port = 22
      to_port   = 22
      cidrs     = var.ssh_allowed_cidrs
    }],
    [
      {
        protocol  = "tcp"
        from_port = 80
        to_port   = 80
        cidrs     = ["0.0.0.0/0"]
      },
      {
        protocol  = "tcp"
        from_port = 443
        to_port   = 443
        cidrs     = ["0.0.0.0/0"]
      },
    ],
  )

  backups_bucket_name = coalesce(
    var.backup_bucket_name,
    "${var.project}-${var.environment}-backups-${data.aws_caller_identity.current.account_id}",
  )
}

# Fail the plan before creating anything when the active credentials do not
# belong to the personal account.
resource "terraform_data" "account_guard" {
  input = data.aws_caller_identity.current.account_id

  lifecycle {
    precondition {
      condition     = data.aws_caller_identity.current.account_id == var.expected_aws_account_id
      error_message = "Refusing to manage Cestia infrastructure: credentials resolve to AWS account ${data.aws_caller_identity.current.account_id}, not the expected account ${var.expected_aws_account_id}."
    }
  }
}

# The instance is replaceable. The database lives on the attached disk and the
# backups live in S3, so destroying and recreating the VM loses nothing.
resource "aws_lightsail_instance" "app" {
  name              = var.instance_name
  availability_zone = var.availability_zone
  blueprint_id      = "ubuntu_24_04"
  bundle_id         = var.instance_bundle
  key_pair_name     = var.key_pair_name

  tags = local.common_tags
}

resource "aws_lightsail_static_ip" "app" {
  name = var.static_ip_name
}

resource "aws_lightsail_static_ip_attachment" "app" {
  static_ip_name = aws_lightsail_static_ip.app.name
  instance_name  = aws_lightsail_instance.app.name
}

resource "aws_lightsail_instance_public_ports" "app" {
  instance_name = aws_lightsail_instance.app.name

  dynamic "port_info" {
    for_each = local.firewall_rules
    content {
      protocol  = port_info.value.protocol
      from_port = port_info.value.from_port
      to_port   = port_info.value.to_port
      cidrs     = port_info.value.cidrs
    }
  }
}

# PostgreSQL data and generated secrets live on this disk, not on the instance
# root volume. It survives instance deletion and can be reattached to a
# replacement VM.
resource "aws_lightsail_disk" "data" {
  name              = var.data_disk_name
  size_in_gb        = var.data_disk_size_gb
  availability_zone = var.availability_zone
  tags              = local.common_tags

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_lightsail_disk_attachment" "data" {
  disk_name     = aws_lightsail_disk.data.name
  instance_name = aws_lightsail_instance.app.name
  disk_path     = var.data_disk_path
}

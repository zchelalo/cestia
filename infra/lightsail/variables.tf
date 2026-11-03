variable "organization" {
  description = "Owning organization used in tags."
  type        = string
  default     = "chelalo"
}

variable "project" {
  description = "Project name used in resource names and tags."
  type        = string
  default     = "cestia"
}

variable "environment" {
  description = "Deployment environment."
  type        = string
  default     = "production"
}

variable "cost_center" {
  description = "Cost center used in tags and cost reports."
  type        = string
  default     = "personal-cestia"
}

variable "aws_profile" {
  description = "AWS CLI profile used by Terraform."
  type        = string
  default     = "default"
}

variable "aws_region" {
  description = "AWS region for Lightsail, S3, SES and Route 53 resources."
  type        = string
  default     = "us-east-1"
}

variable "expected_aws_account_id" {
  description = "Personal AWS account ID. Apply fails when credentials resolve to any other account."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{12}$", var.expected_aws_account_id))
    error_message = "expected_aws_account_id must be a 12-digit AWS account ID."
  }
}

variable "instance_name" {
  description = "Lightsail instance resource name."
  type        = string
  default     = "cestia-app"
}

variable "instance_bundle" {
  description = "Lightsail bundle ID. The default is the cheapest Linux bundle; verify current IDs with `aws lightsail get-bundles`."
  type        = string
  default     = "nano_3_0"
}

variable "key_pair_name" {
  description = "Existing Lightsail key pair used for administrative SSH. Null disables key-based SSH until one is attached."
  type        = string
  default     = null
  nullable    = true
}

variable "availability_zone" {
  description = "AZ for the instance and the data disk (region-specific, both must match)."
  type        = string
  default     = "us-east-1a"
}

variable "static_ip_name" {
  description = "Name for the Lightsail static IP."
  type        = string
  default     = "cestia-app-ip"
}

variable "root_domain" {
  description = "Apex domain already hosted in Route 53."
  type        = string
  default     = "chelalo.me"
}

variable "app_domain" {
  description = "Canonical production hostname, without scheme."
  type        = string
  default     = "cestia.chelalo.me"

  validation {
    condition     = can(regex("^[a-z0-9](?:[a-z0-9.-]{0,251}[a-z0-9])?$", var.app_domain))
    error_message = "app_domain must be a valid lowercase DNS hostname without scheme."
  }
}

variable "ses_domain" {
  description = "Domain identity verified in SES. Empty reuses app_domain."
  type        = string
  default     = ""
}

variable "mail_from_domain" {
  description = "Custom MAIL FROM subdomain for SES. Empty derives mail.<ses_domain>."
  type        = string
  default     = ""
}

variable "smtp_from_address" {
  description = "From address allowed by the SMTP IAM policy. Empty derives no-reply@<ses_domain>."
  type        = string
  default     = ""
}

variable "manage_dns" {
  description = "Create the A, DKIM and MAIL FROM records in the existing Route 53 hosted zone."
  type        = bool
  default     = true
}

variable "github_repo" {
  description = "GitHub repository allowed to assume the deployment role, as owner/name."
  type        = string
  default     = "zchelalo/cestia"
}

variable "github_environment" {
  description = "GitHub Actions environment allowed to assume the deployment role."
  type        = string
  default     = "production"
}

variable "create_github_oidc_provider" {
  description = "Create the GitHub OIDC provider. Set false when the account already has one."
  type        = bool
  default     = true
}

variable "tls_email" {
  description = "Contact email for ACME TLS certificates. Empty omits the Caddy global email."
  type        = string
  default     = ""
}

variable "ssh_allowed_cidrs" {
  description = "CIDRs allowed to SSH. Empty disables public SSH; never use 0.0.0.0/0 in production."
  type        = list(string)
  default     = []

  validation {
    condition     = !contains(var.ssh_allowed_cidrs, "0.0.0.0/0")
    error_message = "Public SSH from 0.0.0.0/0 is not allowed."
  }
}

variable "data_disk_name" {
  description = "Lightsail block storage disk that holds PostgreSQL and generated secrets."
  type        = string
  default     = "cestia-data"
}

variable "data_disk_size_gb" {
  description = "Data disk size in GB. Lightsail disks start at 8 GB."
  type        = number
  default     = 8

  validation {
    condition     = var.data_disk_size_gb >= 8
    error_message = "data_disk_size_gb must be at least 8."
  }
}

variable "data_disk_path" {
  description = "Device path where Lightsail attaches the data disk."
  type        = string
  default     = "/dev/xvdf"
}

variable "backup_bucket_name" {
  description = "Globally unique S3 bucket for PostgreSQL backups. Null derives a stable name from the AWS account ID."
  type        = string
  default     = null
  nullable    = true
}

variable "backup_retention_days" {
  description = "Days that PostgreSQL backups are kept in S3 before expiring."
  type        = number
  default     = 14

  validation {
    condition     = var.backup_retention_days >= 7 && var.backup_retention_days <= 365
    error_message = "backup_retention_days must be between 7 and 365."
  }
}

variable "noncurrent_retention_days" {
  description = "Days that noncurrent S3 object versions are kept before expiring. Current versions never expire."
  type        = number
  default     = 90

  validation {
    condition     = var.noncurrent_retention_days >= 1 && var.noncurrent_retention_days <= 3650
    error_message = "noncurrent_retention_days must be between 1 and 3650."
  }
}

variable "budget_limit_usd" {
  description = "Monthly account budget in USD. AWS Budget only alerts; it does not cap spending."
  type        = number
  default     = 8

  validation {
    condition     = var.budget_limit_usd > 0
    error_message = "budget_limit_usd must be greater than zero."
  }
}

variable "alert_email" {
  description = "Mailbox that receives AWS Budget notifications."
  type        = string

  validation {
    condition     = can(regex("^[^@[:space:]]+@[^@[:space:]]+\\.[^@[:space:]]+$", var.alert_email))
    error_message = "alert_email must be a valid email address."
  }
}

variable "tags" {
  description = "Additional tags merged into common tags."
  type        = map(string)
  default     = {}
}

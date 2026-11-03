variable "organization" {
  description = "Owning organization used in tags."
  type        = string
  default     = "chelalo"
}

variable "project" {
  description = "Project name used in tags."
  type        = string
  default     = "cestia"
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
  description = "AWS region for the Terraform state backend."
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

variable "state_bucket_name" {
  description = "Globally unique S3 bucket name for Terraform state."
  type        = string

  validation {
    condition     = length(var.state_bucket_name) >= 3 && length(var.state_bucket_name) <= 63
    error_message = "state_bucket_name must be a valid globally unique S3 bucket name."
  }
}

variable "tags" {
  description = "Additional tags merged into common tags."
  type        = map(string)
  default     = {}
}

variable "noncurrent_retention_days" {
  description = "Days that noncurrent Terraform state versions are kept before expiring. Current versions never expire."
  type        = number
  default     = 90

  validation {
    condition     = var.noncurrent_retention_days >= 1 && var.noncurrent_retention_days <= 3650
    error_message = "noncurrent_retention_days must be between 1 and 3650."
  }
}

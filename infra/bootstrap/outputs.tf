output "state_bucket_name" {
  description = "S3 bucket to use in backend configuration."
  value       = aws_s3_bucket.terraform_state.id
}

output "backend_config_example" {
  description = "Backend configuration values for infra/lightsail."
  value = {
    bucket       = aws_s3_bucket.terraform_state.id
    key          = "production/lightsail/terraform.tfstate"
    region       = var.aws_region
    encrypt      = true
    use_lockfile = true
  }
}

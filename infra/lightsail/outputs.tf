output "instance_name" {
  description = "Lightsail instance name used by the deployment workflow."
  value       = aws_lightsail_instance.app.name
}

output "instance_public_ip" {
  description = "Current instance public IP. May differ from the static IP until the attachment settles."
  value       = aws_lightsail_instance.app.public_ip_address
}

output "static_ip" {
  description = "Attached static IP used by the Route 53 A record."
  value       = aws_lightsail_static_ip.app.ip_address
}

output "app_domain" {
  description = "Production hostname served by Caddy."
  value       = var.app_domain
}

output "app_url" {
  description = "Canonical production URL."
  value       = "https://${var.app_domain}"
}

output "ssh_command_hint" {
  description = "Example SSH command using the administrative key."
  value       = "ssh -i <secure-path>/<key>.pem ubuntu@${aws_lightsail_static_ip.app.ip_address}"
}

output "data_disk_name" {
  description = "Lightsail disk that holds PostgreSQL and generated secrets."
  value       = aws_lightsail_disk.data.name
}

output "data_disk_device_path" {
  description = "Device path where the data disk is attached."
  value       = var.data_disk_path
}

output "backups_bucket" {
  description = "Private S3 bucket for PostgreSQL backups."
  value       = aws_s3_bucket.backups.id
}

output "runtime_iam_user" {
  description = "IAM user allowed to manage the backups bucket. Create its access key once after apply and store it on the VM."
  value       = aws_iam_user.runtime.name
}

output "ses_domain" {
  description = "Domain identity verified in SES."
  value       = aws_sesv2_email_identity.domain.email_identity
}

output "mail_from_domain" {
  description = "Custom MAIL FROM subdomain configured for SES."
  value       = local.mail_from_domain
}

output "smtp_from_address" {
  description = "From address allowed by the SMTP IAM policy."
  value       = local.smtp_from_address
}

output "ses_smtp_user" {
  description = "IAM user for SES SMTP. Derive its SMTP credentials once after apply and store them on the VM."
  value       = aws_iam_user.ses_smtp.name
}

output "github_deploy_role_arn" {
  description = "IAM role assumed by GitHub Actions through OIDC. Store it as the AWS_DEPLOY_ROLE_ARN repository secret."
  value       = aws_iam_role.github_deploy.arn
}

output "budget_name" {
  description = "AWS Budget that alerts on the monthly account cost."
  value       = aws_budgets_budget.monthly.name
}

output "bootstrap_command" {
  description = "Run once on the instance to install PostgreSQL on the data disk, Caddy and the API service."
  value = trimspace(join(" ", [
    "CESTIA_APP_DOMAIN='${var.app_domain}'",
    "CESTIA_ENVIRONMENT='${var.environment}'",
    "CESTIA_AWS_REGION='${var.aws_region}'",
    "CESTIA_BACKUP_BUCKET='${aws_s3_bucket.backups.id}'",
    "CESTIA_TLS_EMAIL='${var.tls_email}'",
    "CESTIA_DATA_DISK_DEVICE='${var.data_disk_path}'",
    "CESTIA_BACKUP_SCRIPT='infra/lightsail/runtime/scripts/backup-postgres.sh'",
    "sudo -E bash infra/lightsail/scripts/bootstrap-instance.sh",
  ]))
}

output "api_health_url" {
  description = "Local health check used by the deployment script."
  value       = "http://127.0.0.1:8000/api/health"
}

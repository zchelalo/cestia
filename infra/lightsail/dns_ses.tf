locals {
  ses_domain        = var.ses_domain != "" ? var.ses_domain : var.app_domain
  mail_from_domain  = var.mail_from_domain != "" ? var.mail_from_domain : "mail.${local.ses_domain}"
  smtp_from_address = var.smtp_from_address != "" ? var.smtp_from_address : "no-reply@${local.ses_domain}"
}

# The apex domain already lives in Route 53; Terraform only adds the records
# Cestia needs instead of creating a second hosted zone.
data "aws_route53_zone" "root" {
  count        = var.manage_dns ? 1 : 0
  name         = var.root_domain
  private_zone = false
}

resource "aws_route53_record" "app" {
  count   = var.manage_dns ? 1 : 0
  zone_id = data.aws_route53_zone.root[0].zone_id
  name    = var.app_domain
  type    = "A"
  ttl     = 300
  records = [aws_lightsail_static_ip.app.ip_address]
}

resource "aws_sesv2_email_identity" "domain" {
  email_identity = local.ses_domain
  tags           = local.common_tags
}

# Easy DKIM: SES provisions exactly three CNAME tokens for the identity.
resource "aws_route53_record" "dkim" {
  count   = var.manage_dns ? 3 : 0
  zone_id = data.aws_route53_zone.root[0].zone_id
  name    = "${aws_sesv2_email_identity.domain.dkim_signing_attributes[0].tokens[count.index]}._domainkey.${local.ses_domain}"
  type    = "CNAME"
  ttl     = 600
  records = ["${aws_sesv2_email_identity.domain.dkim_signing_attributes[0].tokens[count.index]}.dkim.amazonses.com"]

  depends_on = [aws_sesv2_email_identity.domain]
}

# Custom MAIL FROM so SPF aligns with the sending domain.
resource "aws_sesv2_email_identity_mail_from_attributes" "domain" {
  email_identity         = aws_sesv2_email_identity.domain.email_identity
  mail_from_domain       = local.mail_from_domain
  behavior_on_mx_failure = "USE_DEFAULT_VALUE"
}

resource "aws_route53_record" "mail_from_mx" {
  count   = var.manage_dns ? 1 : 0
  zone_id = data.aws_route53_zone.root[0].zone_id
  name    = local.mail_from_domain
  type    = "MX"
  ttl     = 600
  records = ["10 feedback-smtp.${var.aws_region}.amazonses.com"]

  depends_on = [aws_sesv2_email_identity_mail_from_attributes.domain]
}

resource "aws_route53_record" "mail_from_spf" {
  count   = var.manage_dns ? 1 : 0
  zone_id = data.aws_route53_zone.root[0].zone_id
  name    = local.mail_from_domain
  type    = "TXT"
  ttl     = 600
  records = ["v=spf1 include:amazonses.com ~all"]

  depends_on = [aws_sesv2_email_identity_mail_from_attributes.domain]
}

# The API sends mail over SES SMTP. Terraform creates the user and its scoped
# policy but never the access key: derive SMTP credentials once after apply and
# store them in /etc/cestia/smtp.env on the instance.
resource "aws_iam_user" "ses_smtp" {
  name = "${var.project}-${var.environment}-smtp"
  tags = local.common_tags
}

data "aws_iam_policy_document" "ses_smtp" {
  statement {
    sid = "SendCestiaMail"
    actions = [
      "ses:SendEmail",
      "ses:SendRawEmail",
    ]
    # SES SMTP SendRawEmail authorizes the identity through ses:FromAddress;
    # the action does not accept the identity ARN as a resource restriction.
    resources = ["*"]

    condition {
      test     = "StringEquals"
      variable = "ses:FromAddress"
      values   = [local.smtp_from_address]
    }
  }
}

resource "aws_iam_user_policy" "ses_smtp" {
  name   = "${var.project}-${var.environment}-smtp"
  user   = aws_iam_user.ses_smtp.name
  policy = data.aws_iam_policy_document.ses_smtp.json
}

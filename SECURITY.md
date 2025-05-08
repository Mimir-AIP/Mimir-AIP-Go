# Security Policy

## Supported Versions

Currently supported versions of Mimir-AIP for security updates:

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Reporting a Vulnerability

We take the security of Mimir-AIP seriously. If you believe you have found a security vulnerability, please follow these steps:

1. **Do Not** disclose the vulnerability publicly
2. Email the details to [security@mimir-aip.org]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Any suggestions for mitigation

### What to Expect

- Acknowledgment of your report within 48 hours
- Regular updates on our progress
- Credit for your responsible disclosure (if desired)
- Notification when the vulnerability is fixed

## Security Best Practices

When using Mimir-AIP:

1. **API Keys & Secrets**
   - Never commit API keys or secrets to version control
   - Use environment variables or secure secret management
   - Rotate keys regularly

2. **Access Control**
   - Implement proper access controls for your pipelines
   - Restrict network access appropriately
   - Use principle of least privilege

3. **Input Validation**
   - Validate all inputs in custom plugins
   - Sanitize data from external sources
   - Use safe parsing for YAML configurations

4. **Dependencies**
   - Keep dependencies up to date
   - Review security advisories
   - Use dependency scanning tools

## Security Features

- Environment-based configuration
- Secure defaults for plugin execution
- Logging of security-relevant events
- Test mode for safe validation

## Updates

Security updates will be released as soon as possible after validation. Subscribe to security advisories for notifications.
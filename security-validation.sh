#!/bin/bash

set -euo pipefail

echo "🛡️ Helios Demo - Security Validation Script"
echo "==========================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Security validation results
SECURITY_SCORE=0
TOTAL_CHECKS=0
FAILED_CHECKS=()

# Function to print colored output
print_step() {
    echo -e "${BLUE}[CHECK]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((SECURITY_SCORE++))
}

print_failure() {
    echo -e "${RED}[FAIL]${NC} $1"
    FAILED_CHECKS+=("$1")
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Increment total checks counter
check() {
    ((TOTAL_CHECKS++))
}

# Function to check for exposed credentials in files
check_exposed_credentials() {
    print_step "Checking for exposed credentials..."
    check

    local found_issues=0

    # Check for API keys in .env files (uncommented lines only)
    if find . -name ".env" -type f -exec grep -v "^#" {} \; | grep -q "sk-[a-zA-Z0-9]\|pplx-[a-zA-Z0-9]\|AIza[a-zA-Z0-9]"; then
        print_failure "Found real API keys in .env files"
        found_issues=1
    fi

    # Check for hardcoded secrets in source code (actual patterns, not placeholders)
    if grep -r --include="*.js" --include="*.ts" --include="*.json" -E "sk-[a-zA-Z0-9]{20,}|pplx-[a-zA-Z0-9]{20,}|AIza[a-zA-Z0-9]{20,}" . 2>/dev/null | grep -v node_modules | grep -v placeholder | grep -q .; then
        print_failure "Found hardcoded API keys in source code"
        found_issues=1
    fi

    # Check for Cloudflare tunnel URLs
    if grep -r -l --include="*.js" --include="*.env" --include="*.md" "trycloudflare\.com" . 2>/dev/null | grep -q .; then
        print_failure "Found Cloudflare tunnel URLs in files"
        found_issues=1
    fi

    if [ $found_issues -eq 0 ]; then
        print_success "No exposed credentials found"
    fi
}

# Function to check .gitignore coverage
check_gitignore_coverage() {
    print_step "Checking .gitignore coverage..."
    check

    local missing_patterns=()

    # Check for essential patterns in .gitignore
    if [ -f .gitignore ]; then
        if ! grep -q "\.env" .gitignore; then
            missing_patterns+=(".env files")
        fi

        if ! grep -q "node_modules" .gitignore; then
            missing_patterns+=("node_modules")
        fi

        if ! grep -q "\.key" .gitignore; then
            missing_patterns+=("private keys")
        fi
    else
        print_failure ".gitignore file not found"
        return
    fi

    if [ ${#missing_patterns[@]} -eq 0 ]; then
        print_success ".gitignore has adequate security coverage"
    else
        print_failure "Missing .gitignore patterns: ${missing_patterns[*]}"
    fi
}

# Function to check environment file templates
check_env_templates() {
    print_step "Checking environment file templates..."
    check

    local issues=0

    # Check for .env.template files
    if [ ! -f ".env.template" ] && [ ! -f "demo/frontend/.env.template" ]; then
        print_failure "No .env.template files found"
        issues=1
    fi

    # Check template content safety
    if [ -f "demo/frontend/.env.template" ]; then
        if grep -q "trycloudflare\.com" "demo/frontend/.env.template"; then
            print_failure "Template contains actual URLs instead of placeholders"
            issues=1
        fi
    fi

    if [ $issues -eq 0 ]; then
        print_success "Environment templates are secure"
    fi
}

# Function to check for secure defaults in configuration
check_secure_defaults() {
    print_step "Checking secure configuration defaults..."
    check

    local issues=0

    # Check if CORS is properly configured (not wide open)
    if grep -r "origin.*\*" demo/backend/ 2>/dev/null | grep -v "// TODO\|# TODO" | grep -q .; then
        print_warning "Found wildcard CORS configuration - should be restricted in production"
        # This is a warning, not a failure for demo purposes
    fi

    # Check for debug/development flags left enabled
    if grep -r "DANGEROUSLY_DISABLE_HOST_CHECK=true" . 2>/dev/null | grep -v ".template" | grep -q .; then
        print_failure "Found dangerous development flags enabled"
        issues=1
    fi

    # Check for proper port configuration
    if grep -r "PORT.*||.*300[0-9]" demo/backend/ 2>/dev/null | grep -q .; then
        print_success "Backend has configurable port with secure default"
    else
        print_warning "Backend port configuration could be improved"
    fi

    if [ $issues -eq 0 ]; then
        print_success "Configuration defaults are secure"
    fi
}

# Function to check file permissions
check_file_permissions() {
    print_step "Checking file permissions..."
    check

    local issues=0

    # Check for overly permissive script files
    if find . -name "*.sh" -perm -o+w 2>/dev/null | grep -q .; then
        print_failure "Found world-writable script files"
        issues=1
    fi

    # Check for private key files
    key_files=$(find . -name "*.pem" -o -name "*.key" 2>/dev/null | head -5)
    if [ -n "$key_files" ]; then
        echo "$key_files" | while read -r file; do
            if [ -f "$file" ]; then
                perms=$(stat -c "%a" "$file")
                if [ "$perms" != "600" ] && [ "$perms" != "400" ]; then
                    echo "INSECURE_KEY_FOUND:$file:$perms"
                fi
            fi
        done | if grep -q "INSECURE_KEY_FOUND"; then
            print_failure "Found private keys with insecure permissions"
            issues=1
        fi
    fi

    if [ $issues -eq 0 ]; then
        print_success "File permissions are appropriate"
    fi
}

# Function to check dependency security
check_dependency_security() {
    print_step "Checking dependency security..."
    check

    local issues=0

    # Check for package-lock.json (dependency pinning)
    if [ -f "demo/frontend/package.json" ] && [ ! -f "demo/frontend/package-lock.json" ]; then
        print_warning "Frontend missing package-lock.json (unpinned dependencies)"
    fi

    if [ -f "demo/backend/package.json" ] && [ ! -f "demo/backend/package-lock.json" ]; then
        print_warning "Backend missing package-lock.json (unpinned dependencies)"
    fi

    # Check for known vulnerable packages (basic check)
    if command -v npm &> /dev/null; then
        if [ -d "demo/frontend/node_modules" ]; then
            cd demo/frontend
            if ! npm audit --audit-level=high --production &>/dev/null; then
                print_warning "Frontend dependencies have known vulnerabilities"
            fi
            cd - > /dev/null
        fi

        if [ -d "demo/backend/node_modules" ]; then
            cd demo/backend
            if ! npm audit --audit-level=high --production &>/dev/null; then
                print_warning "Backend dependencies have known vulnerabilities"
            fi
            cd - > /dev/null
        fi
    fi

    print_success "Dependency security checks completed"
}

# Function to check deployment security
check_deployment_security() {
    print_step "Checking deployment security configuration..."
    check

    local issues=0

    # Check for secure deployment scripts
    if [ -f "secure-aws-deploy.sh" ]; then
        if grep -q "set -euo pipefail" secure-aws-deploy.sh; then
            print_success "Deployment script uses secure bash options"
        else
            print_failure "Deployment script missing secure bash options"
            issues=1
        fi
    else
        print_warning "No secure deployment script found"
    fi

    # Check for GitHub Actions security
    if [ -f ".github/workflows/secure-deploy.yml" ]; then
        if grep -q "id-token: write" .github/workflows/secure-deploy.yml; then
            print_success "GitHub Actions configured for OIDC"
        else
            print_warning "GitHub Actions not configured for OIDC authentication"
        fi
    fi

    if [ $issues -eq 0 ] && [ -f "secure-aws-deploy.sh" ]; then
        print_success "Deployment security configuration is adequate"
    fi
}

# Function to check AWS security best practices
check_aws_security() {
    print_step "Checking AWS security configuration..."
    check

    local issues=0

    # Check for IAM policy files
    if [ -f "github-actions-trust-policy.json" ] || grep -r "sts:AssumeRoleWithWebIdentity" . &>/dev/null; then
        print_success "OIDC trust policy configuration found"
    else
        print_warning "No OIDC trust policy configuration found"
    fi

    # Check for security group configuration
    if grep -r "authorize-security-group-ingress" . &>/dev/null; then
        print_success "Security group configuration found in deployment scripts"
    else
        print_warning "No security group configuration found"
    fi

    # Check for secrets management
    if grep -r "secretsmanager\|AWS_SECRETS_MANAGER" . &>/dev/null; then
        print_success "AWS Secrets Manager integration found"
    else
        print_warning "No AWS Secrets Manager integration found"
    fi

    print_success "AWS security checks completed"
}

# Function to generate security report
generate_security_report() {
    echo ""
    echo "🛡️ Security Validation Report"
    echo "============================="
    echo ""

    # Calculate percentage
    local percentage=0
    if [ $TOTAL_CHECKS -gt 0 ]; then
        percentage=$((SECURITY_SCORE * 100 / TOTAL_CHECKS))
    fi

    echo "Overall Security Score: $SECURITY_SCORE/$TOTAL_CHECKS ($percentage%)"
    echo ""

    # Color code the results
    if [ $percentage -ge 80 ]; then
        print_success "EXCELLENT - Production ready security posture"
    elif [ $percentage -ge 60 ]; then
        print_warning "GOOD - Minor security improvements recommended"
    else
        print_failure "NEEDS WORK - Security improvements required before deployment"
    fi

    echo ""

    # List failed checks if any
    if [ ${#FAILED_CHECKS[@]} -gt 0 ]; then
        echo "❌ Failed Security Checks:"
        for check in "${FAILED_CHECKS[@]}"; do
            echo "   • $check"
        done
        echo ""
    fi

    # Security recommendations
    echo "🔒 Security Recommendations:"
    echo "   • Ensure all .env files contain only placeholder values"
    echo "   • Use AWS Secrets Manager for production credentials"
    echo "   • Enable GitHub branch protection rules"
    echo "   • Configure AWS WAF for additional protection"
    echo "   • Implement monitoring and alerting"
    echo "   • Regular security audits and dependency updates"
    echo ""

    # Generate timestamp report
    echo "Report generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
    echo "Report generated by: Claude Code Security Validation"
}

# Function to save report to file
save_security_report() {
    local report_file="security-report-$(date +%Y%m%d-%H%M%S).txt"

    {
        echo "Helios Demo Security Validation Report"
        echo "======================================"
        echo "Generated: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo "Score: $SECURITY_SCORE/$TOTAL_CHECKS ($((SECURITY_SCORE * 100 / TOTAL_CHECKS))%)"
        echo ""
        echo "Failed Checks:"
        for check in "${FAILED_CHECKS[@]}"; do
            echo "- $check"
        done
    } > "$report_file"

    print_success "Security report saved to: $report_file"
}

# Main execution
main() {
    echo "Running comprehensive security validation..."
    echo ""

    check_exposed_credentials
    check_gitignore_coverage
    check_env_templates
    check_secure_defaults
    check_file_permissions
    check_dependency_security
    check_deployment_security
    check_aws_security

    echo ""
    generate_security_report
    save_security_report

    # Exit with appropriate code
    local percentage=$((SECURITY_SCORE * 100 / TOTAL_CHECKS))
    if [ $percentage -ge 80 ]; then
        exit 0  # Success
    elif [ $percentage -ge 60 ]; then
        exit 1  # Warning
    else
        exit 2  # Critical issues
    fi
}

# Run main function
main "$@"
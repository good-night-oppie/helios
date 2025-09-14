#!/bin/bash

set -euo pipefail

echo "🛡️ Quick Security Validation for Helios Demo"
echo "============================================"
echo

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

score=0
total=0

check_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    ((score++))
}

check_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
}

check_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check 1: No real API keys in .env files
((total++))
echo "Checking for real API keys in .env files..."
if grep -h "=" .env demo/frontend/.env 2>/dev/null | grep -v "^#" | grep -qE "(sk-[a-zA-Z0-9]{40,}|pplx-[a-zA-Z0-9]{40,}|AIza[a-zA-Z0-9]{30,})"; then
    check_fail "Found real API keys in .env files"
else
    check_pass "No real API keys found in .env files"
fi

# Check 2: .gitignore includes .env
((total++))
echo "Checking .gitignore coverage..."
if [ -f .gitignore ] && grep -q "\.env" .gitignore; then
    check_pass ".gitignore properly excludes .env files"
else
    check_fail ".gitignore missing or doesn't exclude .env files"
fi

# Check 3: No Cloudflare URLs in committed files
((total++))
echo "Checking for Cloudflare tunnel URLs..."
if grep -r "trycloudflare\.com" demo/ 2>/dev/null | grep -v node_modules | grep -q .; then
    check_fail "Found Cloudflare tunnel URLs in demo files"
else
    check_pass "No Cloudflare tunnel URLs found"
fi

# Check 4: Template files exist and are safe
((total++))
echo "Checking environment templates..."
if [ -f "demo/frontend/.env.template" ] && ! grep -q "trycloudflare\.com" demo/frontend/.env.template; then
    check_pass "Environment templates are safe"
else
    check_fail "Missing or unsafe environment templates"
fi

# Check 5: Secure deployment script exists
((total++))
echo "Checking deployment scripts..."
if [ -f "secure-aws-deploy.sh" ] && [ -x "secure-aws-deploy.sh" ]; then
    check_pass "Secure deployment script exists and is executable"
else
    check_warn "Secure deployment script missing or not executable"
fi

# Check 6: GitHub setup script exists
((total++))
echo "Checking GitHub integration..."
if [ -f "github-setup.sh" ] && [ -x "github-setup.sh" ]; then
    check_pass "GitHub setup script exists and is executable"
else
    check_warn "GitHub setup script missing or not executable"
fi

# Check 7: No world-writable scripts
((total++))
echo "Checking file permissions..."
if find . -name "*.sh" -perm -002 2>/dev/null | grep -q .; then
    check_fail "Found world-writable script files"
else
    check_pass "Script file permissions are secure"
fi

echo ""
echo "=== Security Validation Summary ==="
percentage=$((score * 100 / total))
echo "Score: $score/$total ($percentage%)"

if [ $percentage -ge 85 ]; then
    echo -e "${GREEN}✅ EXCELLENT${NC} - Ready for secure deployment"
elif [ $percentage -ge 70 ]; then
    echo -e "${YELLOW}⚠️  GOOD${NC} - Minor improvements recommended"
else
    echo -e "${RED}❌ NEEDS WORK${NC} - Security issues must be resolved"
fi

echo ""
echo "Next steps:"
echo "1. Run: ./secure-aws-deploy.sh    # Deploy to AWS securely"
echo "2. Run: ./github-setup.sh         # Setup GitHub repository"
echo "3. Configure AWS OIDC for production CI/CD"
echo ""

if [ $percentage -ge 70 ]; then
    echo -e "${GREEN}🚀 Ready for 24-hour showcase deployment!${NC}"
    exit 0
else
    echo -e "${RED}🛡️  Please fix security issues before deployment${NC}"
    exit 1
fi
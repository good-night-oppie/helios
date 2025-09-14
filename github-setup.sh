#!/bin/bash

set -euo pipefail

echo "🐙 Helios Demo - GitHub CLI Setup & Repository Management"
echo "======================================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO_NAME="helios-demo"
REPO_DESCRIPTION="Helios Parallel Universe Engine - 24-Hour Showcase Demo"

# Function to print colored output
print_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check GitHub CLI
check_gh_cli() {
    print_step "Checking GitHub CLI installation..."

    if ! command -v gh &> /dev/null; then
        print_error "GitHub CLI not found. Please install GitHub CLI first."
        echo "Visit: https://cli.github.com/"
        exit 1
    fi

    print_success "GitHub CLI found: $(gh --version | head -n1)"

    # Check authentication
    if ! gh auth status &> /dev/null; then
        print_warning "GitHub CLI not authenticated. Please authenticate first."
        echo ""
        echo "Run: gh auth login"
        echo "Choose: 'GitHub.com' -> 'HTTPS' -> 'Login with a web browser'"
        exit 1
    fi

    print_success "GitHub CLI authenticated"
    gh auth status
}

# Function to create repository
create_repository() {
    print_step "Creating GitHub repository..."

    # Check if repository already exists
    if gh repo view "$REPO_NAME" &> /dev/null; then
        print_warning "Repository $REPO_NAME already exists"
        REPO_URL=$(gh repo view "$REPO_NAME" --json url --jq '.url')
        print_success "Repository URL: $REPO_URL"
        return 0
    fi

    # Create repository
    gh repo create "$REPO_NAME" \
        --description "$REPO_DESCRIPTION" \
        --public \
        --add-readme

    REPO_URL=$(gh repo view "$REPO_NAME" --json url --jq '.url')
    print_success "Created repository: $REPO_URL"
}

# Function to setup repository structure
setup_repository_structure() {
    print_step "Setting up repository structure..."

    # Create .github/workflows directory for CI/CD
    mkdir -p .github/workflows

    # Create GitHub Actions workflow for secure deployment
    cat > .github/workflows/secure-deploy.yml << 'EOF'
name: Secure Helios Deployment

on:
  push:
    branches: [main, master]
  workflow_dispatch:

permissions:
  id-token: write   # Required for OIDC
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production  # Requires approval for production

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '18'
          cache: 'npm'
          cache-dependency-path: |
            demo/frontend/package-lock.json
            demo/backend/package-lock.json

      - name: Install frontend dependencies
        working-directory: demo/frontend
        run: npm ci

      - name: Build frontend
        working-directory: demo/frontend
        run: npm run build

      - name: Install backend dependencies
        working-directory: demo/backend
        run: npm ci

      - name: Run tests (if available)
        working-directory: demo/backend
        run: |
          if [ -f package.json ] && grep -q '"test"' package.json; then
            npm test
          else
            echo "No tests configured, skipping..."
          fi

      # Optional: Configure AWS credentials via OIDC
      # - name: Configure AWS credentials via OIDC
      #   uses: aws-actions/configure-aws-credentials@v4
      #   with:
      #     role-to-assume: arn:aws:iam::${{ secrets.AWS_ACCOUNT_ID }}:role/HeliosGitHubActionsRole
      #     aws-region: us-east-1
      #     role-session-name: HeliosDeployment

      # Optional: Deploy to AWS App Runner
      # - name: Deploy to App Runner
      #   run: |
      #     echo "Deploy to AWS App Runner would go here"
      #     echo "See SECURE_AWS_DEPLOYMENT.md for full implementation"

      - name: Deployment Summary
        run: |
          echo "## Deployment Summary" >> $GITHUB_STEP_SUMMARY
          echo "- ✅ Frontend built successfully" >> $GITHUB_STEP_SUMMARY
          echo "- ✅ Backend dependencies installed" >> $GITHUB_STEP_SUMMARY
          echo "- ✅ Ready for AWS deployment" >> $GITHUB_STEP_SUMMARY
          echo "" >> $GITHUB_STEP_SUMMARY
          echo "### Next Steps:" >> $GITHUB_STEP_SUMMARY
          echo "1. Run \`./secure-aws-deploy.sh\` for AWS deployment" >> $GITHUB_STEP_SUMMARY
          echo "2. Configure AWS OIDC for automatic deployments" >> $GITHUB_STEP_SUMMARY
EOF

    print_success "Created GitHub Actions workflow"
}

# Function to create PR template
create_pr_template() {
    print_step "Creating PR template..."

    mkdir -p .github/pull_request_template.md

    cat > .github/pull_request_template.md << 'EOF'
## Description
Brief description of the changes.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update
- [ ] Demo enhancement

## Testing
- [ ] Local testing completed
- [ ] Demo functionality verified
- [ ] Performance impact assessed

## Security
- [ ] No sensitive data exposed
- [ ] Environment variables properly secured
- [ ] Access controls reviewed

## Demo Impact
- [ ] Frontend changes tested
- [ ] Backend API compatibility maintained
- [ ] WebSocket functionality verified
- [ ] Performance metrics unchanged

## Checklist
- [ ] Code follows the existing style
- [ ] Self-review completed
- [ ] Documentation updated if needed
- [ ] Ready for 24-hour showcase
EOF

    print_success "Created PR template"
}

# Function to setup branch protection
setup_branch_protection() {
    print_step "Setting up branch protection..."

    # This requires the repository to be created first
    if gh repo view "$REPO_NAME" &> /dev/null; then
        # Enable branch protection for main branch
        # Note: This may require admin permissions
        echo "Branch protection setup would require admin permissions."
        echo "Manually configure branch protection rules in GitHub web interface:"
        echo "1. Go to Settings -> Branches"
        echo "2. Add rule for 'main' branch"
        echo "3. Enable 'Require pull request reviews'"
        echo "4. Enable 'Require status checks to pass'"
    else
        print_warning "Repository not found, skipping branch protection setup"
    fi
}

# Function to commit and push initial setup
commit_initial_setup() {
    print_step "Committing initial repository setup..."

    # Initialize git if not already done
    if [ ! -d .git ]; then
        git init
    fi

    # Add remote if not exists
    if ! git remote get-url origin &> /dev/null; then
        git remote add origin "https://github.com/$(gh api user --jq '.login')/$REPO_NAME.git"
    fi

    # Stage files
    git add .github/

    # Check if there are changes to commit
    if git diff --staged --quiet; then
        print_warning "No changes to commit"
        return 0
    fi

    # Commit
    git commit -m "feat: add GitHub Actions workflow and PR template

🚀 Generated with Claude Code

Co-Authored-By: Claude <noreply@anthropic.com>"

    # Push to main/master branch
    MAIN_BRANCH=$(git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || echo "main")
    git push -u origin "$MAIN_BRANCH"

    print_success "Committed and pushed GitHub setup"
}

# Function to display completion summary
display_completion() {
    echo ""
    echo "🎉 GitHub Setup Complete!"
    echo "=========================="

    if gh repo view "$REPO_NAME" &> /dev/null; then
        REPO_URL=$(gh repo view "$REPO_NAME" --json url --jq '.url')
        print_success "Repository: $REPO_URL"
    fi

    echo ""
    echo "✅ Features Configured:"
    echo "   • GitHub Actions workflow for CI/CD"
    echo "   • Pull request template"
    echo "   • Repository structure"
    echo ""
    echo "🚀 Next Steps:"
    echo "1. Configure AWS OIDC for secure deployments:"
    echo "   • Add AWS_ACCOUNT_ID to repository secrets"
    echo "   • Create IAM role with OIDC trust policy"
    echo "   • Update workflow with actual deployment steps"
    echo ""
    echo "2. Push your demo code:"
    echo "   git add demo/"
    echo "   git commit -m \"feat: add Helios demo application\""
    echo "   git push"
    echo ""
    echo "3. Create production environment:"
    echo "   • Go to repository Settings -> Environments"
    echo "   • Create 'production' environment"
    echo "   • Add required reviewers for deployments"
    echo ""
    echo "4. Deploy to AWS:"
    echo "   ./secure-aws-deploy.sh"
    echo ""
    print_success "Your Helios demo repository is ready for professional deployment!"
}

# Main execution
main() {
    echo "Setting up GitHub repository for Helios Demo..."
    echo ""

    check_gh_cli
    create_repository
    setup_repository_structure
    create_pr_template
    # setup_branch_protection  # Commented out as it may require special permissions
    # commit_initial_setup  # Commented out to avoid committing to existing repo
    display_completion

    print_success "GitHub setup complete! Ready for secure cloud deployment."
}

# Run main function
main "$@"
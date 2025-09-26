# GitHub Repository Setup Guide

This document provides step-by-step instructions for configuring the GitHub repository for optimal project management and community collaboration.

## 🔒 Branch Protection Rules

### Main Branch Protection

Navigate to **Settings → Branches** and create a rule for `main`:

```yaml
Branch name pattern: main

Protect matching branches:
✅ Require a pull request before merging
  ✅ Require approvals: 1
  ✅ Dismiss stale PR approvals when new commits are pushed
  ✅ Require review from code owners (if CODEOWNERS file exists)

✅ Require status checks to pass before merging
  ✅ Require branches to be up to date before merging
  Status checks to require:
    - build
    - test
    - lint

✅ Require conversation resolution before merging
✅ Require signed commits (recommended)
✅ Require linear history (optional)
✅ Restrict pushes that create files larger than 100MB
✅ Include administrators
```

### Develop Branch Protection (if using GitFlow)

```yaml
Branch name pattern: develop

Protect matching branches:
✅ Require a pull request before merging
  ✅ Require approvals: 1
✅ Require status checks to pass before merging
✅ Include administrators
```

## 🔐 Repository Secrets

Navigate to **Settings → Secrets and variables → Actions**:

### Required Secrets

```yaml
DOCKER_USERNAME     # Docker Hub username for image publishing
DOCKER_PASSWORD     # Docker Hub password/token
COSIGN_PRIVATE_KEY  # Private key for artifact signing
COSIGN_PASSWORD     # Password for the private key
```

### Optional Secrets (for enhanced CI/CD)

```yaml
CODECOV_TOKEN       # For code coverage reporting
SONAR_TOKEN         # For SonarCloud integration
SLACK_WEBHOOK       # For build notifications
```

## 🏷️ Issue Labels

Navigate to **Settings → Issues → Labels** and create:

### Type Labels

- `bug` (🔴 #d73a4a) - Something isn't working
- `enhancement` (🔵 #a2eeef) - New feature or request
- `documentation` (🟢 #0075ca) - Improvements or additions to documentation
- `question` (🟣 #d876e3) - Further information is requested
- `duplicate` (⚪ #cfd3d7) - This issue or pull request already exists
- `invalid` (⚪ #e4e669) - This doesn't seem right
- `wontfix` (⚪ #ffffff) - This will not be worked on

### Priority Labels

- `priority/critical` (🔴 #b60205) - Critical priority
- `priority/high` (🟠 #d93f0b) - High priority
- `priority/medium` (🟡 #fbca04) - Medium priority
- `priority/low` (🟢 #0e8a16) - Low priority

### Status Labels

- `status/triage` (⚪ #ededed) - Needs triage
- `status/in-progress` (🟡 #fbca04) - Currently being worked on
- `status/blocked` (🔴 #b60205) - Blocked by external dependency
- `status/needs-review` (🟣 #d876e3) - Needs code review

### Community Labels

- `good first issue` (🟢 #7057ff) - Good for newcomers
- `help wanted` (🔵 #008672) - Extra attention is needed
- `hacktoberfest` (🟠 #ff8c00) - Hacktoberfest eligible

## ⚙️ Repository Settings

### General Settings

Navigate to **Settings → General**:

```yaml
Features:
✅ Issues
✅ Projects
✅ Wiki (optional)
✅ Discussions (recommended for community)

Pull Requests:
✅ Allow merge commits
✅ Allow squash merging
✅ Allow rebase merging
✅ Always suggest updating pull request branches
✅ Automatically delete head branches
✅ Allow auto-merge

Archives:
✅ Include Git LFS objects in archives
```

### Security Settings

Navigate to **Settings → Security**:

```yaml
Security and analysis:
✅ Dependency graph
✅ Dependabot alerts
✅ Dependabot security updates
✅ Dependabot version updates
✅ Code scanning (GitHub Advanced Security)
✅ Secret scanning (GitHub Advanced Security)
```

## 📊 Project Management

### GitHub Projects

1. Navigate to **Projects** tab
2. Create a new project (Beta)
3. Set up columns:
   - 📋 Backlog
   - 🔄 In Progress
   - 👀 In Review
   - ✅ Done

### Milestones

Navigate to **Issues → Milestones** and create:

- v1.1.0 - Next minor release
- v2.0.0 - Next major release
- Documentation improvements
- Performance optimizations

## 🗣️ Discussions Setup

Navigate to **Settings → Features** and enable Discussions:

### Categories to Create

1. **General** - General discussions about the project
2. **Ideas** - Share ideas for new features
3. **Q&A** - Ask the community for help
4. **Show and tell** - Show off something you've built
5. **Announcements** - Updates from maintainers

## 🤖 GitHub Apps (Recommended)

Consider installing these GitHub Apps:

### Code Quality

- **Codecov** - Code coverage reporting
- **SonarCloud** - Code quality and security analysis
- **CodeClimate** - Automated code review

### Project Management

- **Linear** - Issue tracking integration
- **Notion** - Documentation integration

### Security

- **Snyk** - Vulnerability scanning
- **WhiteSource** - License compliance

## 📈 Insights and Analytics

### Repository Insights

Navigate to **Insights** to monitor:

- Traffic (views, clones)
- Commits activity
- Code frequency
- Contributors
- Community standards

### Actions Usage

Monitor GitHub Actions usage in **Settings → Billing**

## 🔄 Automation Rules

### Auto-assign Issues

Create `.github/ISSUE_TEMPLATE/config.yml`:

```yaml
blank_issues_enabled: false
contact_links:
  - name: Community Support
    url: https://github.com/onigirazu-cfg/onigirazu/discussions
    about: Please ask and answer questions here.
  - name: Security Issues
    url: mailto:security@example.com
    about: Please report security vulnerabilities privately.
```

### CODEOWNERS File

Create `.github/CODEOWNERS`:

```
# Global owners
* @maintainer-username

# Go code
*.go @go-expert-username

# Documentation
*.md @docs-maintainer-username

# CI/CD
.github/ @devops-expert-username
```

## 📋 Checklist for Repository Setup

- [ ] Branch protection rules configured
- [ ] Repository secrets added
- [ ] Issue labels created
- [ ] Repository settings configured
- [ ] Discussions enabled and configured
- [ ] Projects created
- [ ] Milestones defined
- [ ] CODEOWNERS file created
- [ ] Security features enabled
- [ ] GitHub Apps installed (optional)

## 🚀 Next Steps

After completing the repository setup:

1. **Create your first milestone** for the next release
2. **Set up project boards** for better issue tracking
3. **Configure notifications** for team members
4. **Create team discussions** for project planning
5. **Set up integrations** with your development tools

## 📞 Support

If you need help with any of these configurations, please:

- Check the [GitHub Documentation](https://docs.github.com)
- Ask in [GitHub Community](https://github.community)
- Create an issue in this repository

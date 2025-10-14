#!/bin/bash

# Release Helper Script for Onigirazu
# This script helps manage releases and provides utilities for version management

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
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

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to get current version
get_current_version() {
    git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"
}

# Function to validate version format
validate_version() {
    local version="$1"
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$ ]]; then
        print_error "Invalid version format: $version"
        print_status "Expected format: v1.2.3 or v1.2.3-alpha.1"
        return 1
    fi
    return 0
}

# Function to increment version
increment_version() {
    local version="$1"
    local type="$2"

    # Remove 'v' prefix
    version=${version#v}

    # Split version into parts
    IFS='.' read -ra PARTS <<< "$version"
    local major="${PARTS[0]}"
    local minor="${PARTS[1]}"
    local patch="${PARTS[2]%%-*}"  # Remove any pre-release suffix

    case "$type" in
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "patch")
            patch=$((patch + 1))
            ;;
        *)
            print_error "Invalid increment type: $type"
            return 1
            ;;
    esac

    echo "v${major}.${minor}.${patch}"
}

# Function to check for uncommitted changes
check_clean_working_tree() {
    if ! git diff-index --quiet HEAD --; then
        print_error "Working tree is not clean. Please commit or stash your changes."
        return 1
    fi
    return 0
}

# Function to check if on main branch
check_main_branch() {
    local current_branch=$(git branch --show-current)
    if [ "$current_branch" != "main" ] && [ "$current_branch" != "master" ]; then
        print_warning "You are not on the main branch (current: $current_branch)"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            return 1
        fi
    fi
    return 0
}

# Function to analyze commits since last tag
analyze_commits() {
    local latest_tag=$(get_current_version)
    print_status "Analyzing commits since $latest_tag..."

    local commits_since_tag=$(git rev-list ${latest_tag}..HEAD --count)
    print_status "Commits since tag: $commits_since_tag"

    if [ "$commits_since_tag" -eq 0 ]; then
        print_warning "No commits since last tag"
        return 1
    fi

    # Check for conventional commit types
    local feat_commits=$(git log ${latest_tag}..HEAD --oneline --grep="^feat" --count)
    local fix_commits=$(git log ${latest_tag}..HEAD --oneline --grep="^fix" --count)
    local breaking_commits=$(git log ${latest_tag}..HEAD --oneline --grep="BREAKING CHANGE" --count)

    print_status "Feature commits: $feat_commits"
    print_status "Fix commits: $fix_commits"
    print_status "Breaking commits: $breaking_commits"

    # Suggest release type
    if [ "$breaking_commits" -gt 0 ]; then
        echo "major"
    elif [ "$feat_commits" -gt 0 ]; then
        echo "minor"
    elif [ "$fix_commits" -gt 0 ]; then
        echo "patch"
    else
        echo "patch"  # Default to patch for other changes
    fi
}

# Function to generate changelog
generate_changelog() {
    local latest_tag="$1"
    local next_tag="$2"

    print_status "Generating changelog from $latest_tag to $next_tag..."

    echo "## $next_tag ($(date +%Y-%m-%d))"
    echo ""

    # Add features
    local feat_commits=$(git log ${latest_tag}..HEAD --oneline --grep="^feat" --pretty=format:"- %s (%h)")
    if [ -n "$feat_commits" ]; then
        echo "### ✨ New Features"
        echo "$feat_commits"
        echo ""
    fi

    # Add fixes
    local fix_commits=$(git log ${latest_tag}..HEAD --oneline --grep="^fix" --pretty=format:"- %s (%h)")
    if [ -n "$fix_commits" ]; then
        echo "### 🐛 Bug Fixes"
        echo "$fix_commits"
        echo ""
    fi

    # Add other changes
    local other_commits=$(git log ${latest_tag}..HEAD --oneline --invert-grep --grep="^feat" --grep="^fix" --pretty=format:"- %s (%h)")
    if [ -n "$other_commits" ]; then
        echo "### 🔧 Other Changes"
        echo "$other_commits"
        echo ""
    fi
}

# Function to create release
create_release() {
    local version="$1"
    local prerelease="${2:-false}"

    validate_version "$version" || return 1
    check_clean_working_tree || return 1
    check_main_branch || return 1

    print_status "Creating release $version..."

    # Generate changelog
    local current_version=$(get_current_version)
    local changelog=$(generate_changelog "$current_version" "$version")

    # Create and push tag
    git tag -a "$version" -m "Release $version"
    git push origin "$version"

    print_success "Tag $version created and pushed"

    # Create GitHub release if gh CLI is available
    if command_exists gh; then
        print_status "Creating GitHub release..."

        local release_flags=""
        if [ "$prerelease" = "true" ]; then
            release_flags="--prerelease"
        fi

        echo "$changelog" | gh release create "$version" $release_flags --title "Release $version" --notes-file -

        print_success "GitHub release created"
    else
        print_warning "GitHub CLI not available. Please create the release manually."
        print_status "Changelog:"
        echo "$changelog"
    fi
}

# Function to show release status
show_release_status() {
    print_status "Release Status:"
    echo

    local current_version=$(get_current_version)
    print_status "Current version: $current_version"

    local commits_since_tag=$(git rev-list ${current_version}..HEAD --count)
    print_status "Commits since last release: $commits_since_tag"

    if [ "$commits_since_tag" -gt 0 ]; then
        local suggested_type=$(analyze_commits)
        local next_version=$(increment_version "$current_version" "$suggested_type")
        print_status "Suggested next version: $next_version ($suggested_type)"

        echo
        print_status "Recent commits:"
        git log ${current_version}..HEAD --oneline --max-count=10
    else
        print_success "No commits since last release"
    fi
}

# Function to prepare release
prepare_release() {
    local release_type="$1"

    if [ -z "$release_type" ]; then
        local suggested_type=$(analyze_commits)
        print_status "Suggested release type: $suggested_type"
        read -p "Release type (major/minor/patch) [$suggested_type]: " release_type
        release_type=${release_type:-$suggested_type}
    fi

    local current_version=$(get_current_version)
    local next_version=$(increment_version "$current_version" "$release_type")

    print_status "Preparing release: $current_version → $next_version"

    # Run tests
    print_status "Running tests..."
    if ! make test; then
        print_error "Tests failed. Please fix before releasing."
        return 1
    fi

    # Run linting
    print_status "Running linting..."
    if ! make lint; then
        print_error "Linting failed. Please fix before releasing."
        return 1
    fi

    # Build
    print_status "Testing build..."
    if ! make build; then
        print_error "Build failed. Please fix before releasing."
        return 1
    fi

    print_success "Pre-release checks passed"

    # Confirm release
    echo
    print_status "Ready to create release $next_version"
    generate_changelog "$current_version" "$next_version"
    echo

    read -p "Create release? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        create_release "$next_version"
    else
        print_status "Release cancelled"
    fi
}

# Function to create hotfix
create_hotfix() {
    local version="$1"

    if [ -z "$version" ]; then
        local current_version=$(get_current_version)
        local next_version=$(increment_version "$current_version" "patch")
        print_status "Suggested hotfix version: $next_version"
        read -p "Hotfix version [$next_version]: " version
        version=${version:-$next_version}
    fi

    validate_version "$version" || return 1

    print_status "Creating hotfix release $version..."

    # Run quick tests
    print_status "Running tests..."
    if ! go test ./...; then
        print_error "Tests failed. Please fix before releasing hotfix."
        return 1
    fi

    create_release "$version"
}

# Function to list releases
list_releases() {
    print_status "Recent releases:"
    git tag --sort=-version:refname | head -10
}

# Main function
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}    Onigirazu Release Helper           ${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo

    # Check if we're in the right directory
    if [ ! -f "go.mod" ] || [ ! -d "cmd/onigirazu" ]; then
        print_error "This script must be run from the project root directory"
        exit 1
    fi

    case "${1:-status}" in
        "status")
            show_release_status
            ;;

        "prepare")
            prepare_release "$2"
            ;;

        "create")
            if [ -z "$2" ]; then
                print_error "Version required for create command"
                print_status "Usage: $0 create v1.2.3"
                exit 1
            fi
            create_release "$2" "$3"
            ;;

        "hotfix")
            create_hotfix "$2"
            ;;

        "changelog")
            local current_version=$(get_current_version)
            local next_version="$2"
            if [ -z "$next_version" ]; then
                local suggested_type=$(analyze_commits)
                next_version=$(increment_version "$current_version" "$suggested_type")
            fi
            generate_changelog "$current_version" "$next_version"
            ;;

        "list")
            list_releases
            ;;

        "analyze")
            analyze_commits
            ;;

        "help")
            echo "Usage: $0 [command] [options]"
            echo
            echo "Commands:"
            echo "  status              - Show release status (default)"
            echo "  prepare [type]      - Prepare and create release (major/minor/patch)"
            echo "  create <version>    - Create specific version release"
            echo "  hotfix [version]    - Create hotfix release"
            echo "  changelog [version] - Generate changelog"
            echo "  list                - List recent releases"
            echo "  analyze             - Analyze commits for release type"
            echo "  help                - Show this help"
            echo
            echo "Examples:"
            echo "  $0 status           # Show current release status"
            echo "  $0 prepare minor    # Prepare minor release"
            echo "  $0 create v1.2.3    # Create specific version"
            echo "  $0 hotfix           # Create hotfix with auto-increment"
            ;;

        *)
            print_error "Unknown command: $1"
            print_status "Run '$0 help' for usage information"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"

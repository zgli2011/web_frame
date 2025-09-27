#!/bin/bash

# Proto插件构建脚本

set -e

PROTO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLUGINS_DIR="$PROTO_DIR/plugins"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

build_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"

    if [ ! -d "$plugin_dir" ]; then
        log_error "Plugin directory not found: $plugin_dir"
        return 1
    fi

    if [ ! -f "$plugin_dir/main.go" ]; then
        log_error "main.go not found in: $plugin_dir"
        return 1
    fi

    log_info "Building plugin: $plugin_name"

    cd "$plugin_dir"

    # 构建插件
    local binary_name="protoc-gen-$plugin_name"
    if go build -o "$binary_name" main.go; then
        log_success "Built $plugin_name -> $plugin_dir/$binary_name"

        # 检查是否有版本信息
        if ./"$binary_name" -version 2>/dev/null; then
            log_info "Version: $(./"$binary_name" -version)"
        fi
    else
        log_error "Failed to build plugin: $plugin_name"
        return 1
    fi
}

list_plugins() {
    log_info "Available plugins:"
    for plugin_dir in "$PLUGINS_DIR"/*; do
        if [ -d "$plugin_dir" ]; then
            local plugin_name=$(basename "$plugin_dir")
            local main_go="$plugin_dir/main.go"
            local binary="$plugin_dir/protoc-gen-$plugin_name"

            if [ -f "$main_go" ]; then
                local status=""
                if [ -f "$binary" ]; then
                    status="${GREEN}[BUILT]${NC}"
                else
                    status="${YELLOW}[NOT BUILT]${NC}"
                fi
                echo -e "  - $plugin_name $status"
            fi
        fi
    done
}

build_all_plugins() {
    log_info "Building all plugins..."
    local success_count=0
    local total_count=0

    for plugin_dir in "$PLUGINS_DIR"/*; do
        if [ -d "$plugin_dir" ] && [ -f "$plugin_dir/main.go" ]; then
            local plugin_name=$(basename "$plugin_dir")
            ((total_count++))

            if build_plugin "$plugin_name"; then
                ((success_count++))
            fi
            echo
        fi
    done

    log_info "Build summary: $success_count/$total_count plugins built successfully"
}

clean_plugins() {
    log_info "Cleaning built plugins..."
    local cleaned_count=0

    for plugin_dir in "$PLUGINS_DIR"/*; do
        if [ -d "$plugin_dir" ]; then
            local plugin_name=$(basename "$plugin_dir")
            local binary="$plugin_dir/protoc-gen-$plugin_name"

            if [ -f "$binary" ]; then
                rm "$binary"
                log_info "Removed: $binary"
                ((cleaned_count++))
            fi
        fi
    done

    log_success "Cleaned $cleaned_count plugin binaries"
}

install_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    local binary="$plugin_dir/protoc-gen-$plugin_name"

    if [ ! -f "$binary" ]; then
        log_error "Plugin not built: $plugin_name. Run: $0 $plugin_name"
        return 1
    fi

    # 检查 $GOBIN 或 $GOPATH/bin
    local install_dir="$GOBIN"
    if [ -z "$install_dir" ] && [ -n "$GOPATH" ]; then
        install_dir="$GOPATH/bin"
    fi
    if [ -z "$install_dir" ]; then
        install_dir="$HOME/go/bin"
    fi

    if [ ! -d "$install_dir" ]; then
        mkdir -p "$install_dir"
    fi

    cp "$binary" "$install_dir/"
    log_success "Installed $plugin_name to $install_dir/protoc-gen-$plugin_name"
}

show_usage() {
    cat << EOF
Proto Plugins Build Script

Usage: $0 [COMMAND] [PLUGIN_NAME]

Commands:
  build [PLUGIN_NAME]  Build specific plugin or all plugins if no name provided
  list                 List all available plugins
  clean               Clean all built plugin binaries
  install PLUGIN_NAME  Install plugin binary to \$GOBIN or \$GOPATH/bin
  help                Show this help message

Examples:
  $0 build                    # Build all plugins
  $0 build web-registry       # Build web-registry plugin only
  $0 list                     # List available plugins
  $0 clean                    # Clean built binaries
  $0 install web-registry     # Install web-registry plugin

Plugin directory structure:
  plugins/
  ├── web-registry/
  │   ├── main.go
  │   └── protoc-gen-web-registry (built binary)
  └── other-plugin/
      ├── main.go
      └── protoc-gen-other-plugin (built binary)

EOF
}

main() {
    case "${1:-build}" in
        "build")
            if [ -n "$2" ]; then
                build_plugin "$2"
            else
                build_all_plugins
            fi
            ;;
        "list")
            list_plugins
            ;;
        "clean")
            clean_plugins
            ;;
        "install")
            if [ -z "$2" ]; then
                log_error "Plugin name required for install command"
                show_usage
                exit 1
            fi
            install_plugin "$2"
            ;;
        "help"|"-h"|"--help")
            show_usage
            ;;
        *)
            log_error "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"
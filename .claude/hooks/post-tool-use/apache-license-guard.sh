#!/bin/bash

# Apache License Guard Hook
# Automatically ensures Apache License 2.0 compliance on all Go file edits
# Copyright 2025 Oppie Thunder Contributors
# Licensed under the Apache License, Version 2.0

set -e

HOOK_NAME="apache-license-guard"
TOOL_NAME="$1"
EXIT_CODE="$2"
MODIFIED_FILES="$3"

# Only run on successful Edit/MultiEdit operations
if [[ "$EXIT_CODE" != "0" ]] || [[ ! "$TOOL_NAME" =~ ^(Edit|MultiEdit)$ ]]; then
    exit 0
fi

# Apache License header template
APACHE_HEADER="// Copyright 2025 Oppie Thunder Contributors
//
// Licensed under the Apache License, Version 2.0 (the \"License\");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an \"AS IS\" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

"

log() {
    echo "[$HOOK_NAME] $1" >&2
}

fix_license_header() {
    local file="$1"
    
    # Check if file exists and is a Go file
    if [[ ! -f "$file" ]] || [[ ! "$file" =~ \.go$ ]]; then
        return 0
    fi
    
    # Check if file already has Apache license
    if grep -q "Licensed under the Apache License, Version 2.0" "$file" 2>/dev/null; then
        log "✅ $file - Already has Apache License 2.0"
        return 0
    fi
    
    # Check for other license indicators
    if grep -qE "(MIT License|GPL|BSD|Copyright.*License)" "$file" 2>/dev/null; then
        log "⚠️  $file - Non-Apache license detected, fixing..."
        
        # Remove existing license headers (first comment block)
        awk 'BEGIN { in_header = 1; blank_count = 0 }
             /^\/\// && in_header { next }
             /^$/ && in_header { blank_count++; if (blank_count >= 2) in_header = 0; next }
             /^[^\/]/ { in_header = 0 }
             !in_header { print }' "$file" > "$file.tmp"
        
        # Add Apache header
        {
            echo "$APACHE_HEADER"
            cat "$file.tmp"
        } > "$file"
        
        rm "$file.tmp"
        log "✅ $file - Apache License 2.0 header added"
        
    else
        # No license detected, add Apache header
        log "📝 $file - No license detected, adding Apache License 2.0..."
        
        {
            echo "$APACHE_HEADER"
            cat "$file"
        } > "$file.tmp"
        
        mv "$file.tmp" "$file"
        log "✅ $file - Apache License 2.0 header added"
    fi
}

# Process modified files (if provided as comma-separated list)
if [[ -n "$MODIFIED_FILES" ]]; then
    IFS=',' read -ra FILES <<< "$MODIFIED_FILES"
    for file in "${FILES[@]}"; do
        fix_license_header "$file"
    done
else
    # Fallback: check all Go files in the project (last 5 minutes modified)
    log "Checking recently modified Go files..."
    find . -name "*.go" -mtime -1 -type f | while read -r file; do
        fix_license_header "$file"
    done
fi

log "Apache License compliance check completed"
exit 0
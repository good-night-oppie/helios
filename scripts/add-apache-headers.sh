#!/bin/bash

# Script to add Apache License 2.0 headers to Go files
# Copyright 2025 Oppie Thunder Contributors
# Licensed under the Apache License, Version 2.0

set -e

HEADER="// Copyright 2025 Oppie Thunder Contributors
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

echo "Adding Apache License 2.0 headers to Go files..."

# Find all Go files, excluding vendor directory
find . -name "*.go" -type f | grep -v vendor | while read -r file; do
    # Check if file already has Apache license header
    if ! grep -q "Apache License" "$file"; then
        echo "Adding header to: $file"
        
        # Create temporary file with header + original content
        {
            echo "$HEADER"
            cat "$file"
        } > "$file.tmp"
        
        # Replace original file
        mv "$file.tmp" "$file"
    else
        echo "Skipping (already has license): $file"
    fi
done

echo "✅ Apache License headers added to all Go files"
#!/usr/bin/env python3
"""
Script to update YAML files from old syntax to new simplified syntax.
Converts:
  module:
    type: package
    name: git
To:
  package:
    name: git
"""

import os
import re
import sys
from pathlib import Path

def update_yaml_syntax(content):
    """Update YAML content from old syntax to new syntax."""
    lines = content.split('\n')
    result = []
    i = 0

    while i < len(lines):
        line = lines[i]

        # Check if this is a module: line
        if re.match(r'^(\s*)module:\s*$', line):
            indent = re.match(r'^(\s*)', line).group(1)

            # Look ahead for type: field
            j = i + 1
            module_type = None
            module_args = []

            while j < len(lines):
                next_line = lines[j]

                # Check if we're still in the module block
                if not next_line.strip():
                    j += 1
                    continue

                # Get indentation of next line
                next_indent_match = re.match(r'^(\s*)', next_line)
                if not next_indent_match:
                    break

                next_indent = next_indent_match.group(1)

                # If indentation is less or equal to module:, we're done
                if len(next_indent) <= len(indent):
                    break

                # Check for type: field
                type_match = re.match(r'^\s*type:\s*["\']?(\w+)["\']?\s*$', next_line)
                if type_match:
                    module_type = type_match.group(1)
                    j += 1
                    continue

                # This is a module argument
                module_args.append(next_line)
                j += 1

            # If we found a module type, convert to new syntax
            if module_type:
                result.append(f"{indent}{module_type}:")
                result.extend(module_args)
                i = j
                continue

        result.append(line)
        i += 1

    return '\n'.join(result)

def process_file(filepath):
    """Process a single YAML file."""
    try:
        with open(filepath, 'r', encoding='utf-8') as f:
            content = f.read()

        # Check if file contains old syntax
        if 'type:' not in content or 'module:' not in content:
            return False

        updated_content = update_yaml_syntax(content)

        # Only write if content changed
        if updated_content != content:
            with open(filepath, 'w', encoding='utf-8') as f:
                f.write(updated_content)
            return True

        return False
    except Exception as e:
        print(f"Error processing {filepath}: {e}", file=sys.stderr)
        return False

def main():
    """Main function."""
    base_dir = Path(__file__).parent
    examples_dir = base_dir / 'examples'

    if not examples_dir.exists():
        print(f"Examples directory not found: {examples_dir}", file=sys.stderr)
        return 1

    updated_files = []

    # Process all .yml files in examples directory
    for yml_file in examples_dir.glob('*.yml'):
        if process_file(yml_file):
            updated_files.append(yml_file.name)
            print(f"✓ Updated: {yml_file.name}")
        else:
            print(f"  Skipped: {yml_file.name}")

    print(f"\nTotal files updated: {len(updated_files)}")

    return 0

if __name__ == '__main__':
    sys.exit(main())

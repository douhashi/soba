#!/usr/bin/env python3
import re
import sys

def convert_log_call(match):
    """Convert old-style log calls to new format with context."""
    indent = match.group(1)
    level = match.group(2)
    message = match.group(3)
    args = match.group(4) if match.group(4) else ""

    # Parse the key-value pairs
    fields = []
    if args:
        # Split by comma, but handle nested structures
        parts = []
        current = ""
        paren_depth = 0
        in_quotes = False
        quote_char = None

        for char in args:
            if char in ('"', "'") and not in_quotes:
                in_quotes = True
                quote_char = char
            elif char == quote_char and in_quotes:
                in_quotes = False
                quote_char = None
            elif not in_quotes:
                if char == '(':
                    paren_depth += 1
                elif char == ')':
                    paren_depth -= 1
                elif char == ',' and paren_depth == 0:
                    parts.append(current.strip())
                    current = ""
                    continue
            current += char
        if current.strip():
            parts.append(current.strip())

        # Convert each part to Field
        i = 0
        while i < len(parts):
            if i + 1 < len(parts):
                key = parts[i].strip('"').strip("'")
                value = parts[i + 1]
                fields.append(f'logging.Field{{Key: "{key}", Value: {value}}}')
                i += 2
            else:
                i += 1

    # Build the new call
    if fields:
        fields_str = ",\n" + indent + "\t" + f",\n{indent}\t".join(fields) + ",\n" + indent
        return f'{indent}d.logger.{level}(ctx, "{message}"{fields_str})'
    else:
        return f'{indent}d.logger.{level}(ctx, "{message}")'

def fix_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()

    # Pattern to match logger calls
    pattern = r'^(\s*)d\.logger\.(Info|Debug|Error|Warn)\("([^"]+)"(?:,\s*(.+?))?\)$'

    lines = content.split('\n')
    new_lines = []

    for line in lines:
        match = re.match(pattern, line)
        if match:
            new_lines.append(convert_log_call(match))
        else:
            new_lines.append(line)

    with open(filepath, 'w') as f:
        f.write('\n'.join(new_lines))

if __name__ == "__main__":
    fix_file("internal/service/daemon.go")
    print("Fixed daemon.go logger calls")
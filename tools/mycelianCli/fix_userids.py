#!/usr/bin/env python3
"""
Script to remove userID references from mycelianCli main.go
"""
import re

def fix_main_go():
    with open('main.go', 'r') as f:
        content = f.read()

    # Remove userID from variable declarations
    content = re.sub(r'var userID, ', 'var ', content)

    # Remove user_id from logging statements
    content = re.sub(r'\s*Str\("user_id", userID\)\.\n', '\n', content)
    content = re.sub(r'\s*Str\("user_id", userID\)\.\s*', '\n\t\t\t\t', content)

    # Remove user-id flag definitions
    content = re.sub(r'\s*cmd\.Flags\(\)\.StringVar\(&userID, "user-id"[^)]*\)\n', '', content)

    # Remove user-id required flag markers
    content = re.sub(r'\s*_ = cmd\.MarkFlagRequired\("user-id"\)\n', '', content)

    # Fix command descriptions
    content = re.sub(r'for a user', '', content)
    content = re.sub(r'for users', '', content)

    # Clean up any standalone userID variables
    content = re.sub(r'var userID string\n', '', content)

    with open('main.go', 'w') as f:
        f.write(content)

    print("Fixed userID references in main.go")

if __name__ == '__main__':
    fix_main_go()

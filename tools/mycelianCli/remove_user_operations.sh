#!/bin/bash

# Script to remove user-related operations from mycelianCli
# This removes --user-id flags and references, using dev mode auth instead

echo "Removing user-related operations from mycelianCli..."

# Remove user-id from all commands systematically
sed -i '' 's/var userID, /var /g' main.go
sed -i '' 's/, userID string//g' main.go
sed -i '' 's/userID, //g' main.go
sed -i '' 's/--user-id[^"]*"[^"]*"[^)]*) *//g' main.go
sed -i '' 's/_ = cmd\.MarkFlagRequired("user-id")//g' main.go
sed -i '' 's/Str("user_id", userID)\.//g' main.go
sed -i '' 's/\.Str("user_id", userID)//g' main.go

# Update command descriptions
sed -i '' 's/for a user//g' main.go
sed -i '' 's/for users//g' main.go

echo "User operations removed. Please review the changes."

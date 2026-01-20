#!/bin/bash

# Navigate to repository directory
cd /Users/Ubermensch/development/test-solution/observer/internal/repository

# Remove the old corrupted file and the duplicate
rm -f mongodb_step.go mongodb_step_new.go

# Rename the clean file to the correct name
mv mongodb_step_clean.go mongodb_step.go

echo "Cleanup complete!"
echo "Deleted: mongodb_step.go (corrupted), mongodb_step_new.go (duplicate)"
echo "Renamed: mongodb_step_clean.go -> mongodb_step.go"

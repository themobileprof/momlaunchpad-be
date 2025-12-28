#!/bin/bash

# Load environment variables
set -a
source .env
set +a

# Run migrations
~/go/bin/migrate -path migrations -database "${DATABASE_URL}" "$@"

#!/bin/bash

# Define paths relative to the script's location
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SMART_SPEND_DIR="$ROOT_DIR/smart_spend"
SERVER_DIR="$ROOT_DIR/server"
DIST_DIR="$SMART_SPEND_DIR/dist"

echo "====================================================="
echo "       Starting FlowCollect Server Build             "
echo "====================================================="

# Record start time
START_TIME=$(date +%s)

# ====================================================
# Step 1: Build the frontend (smart_spend)
# ====================================================
echo "[1/3] Building frontend (smart_spend)..."
if [ ! -d "$SMART_SPEND_DIR" ]; then
    echo "Error: Directory '$SMART_SPEND_DIR' does not exist."
    exit 1
fi

cd "$SMART_SPEND_DIR" || exit 1
# Install dependencies if node_modules is missing, or just run install to be safe
npm install
npm run build
if [ $? -ne 0 ]; then
    echo "Error: Frontend build failed."
    exit 1
fi
echo "Frontend build completed successfully."
echo ""

# ====================================================
# Step 2: Build the backend (server) for Linux
# ====================================================
echo "[2/3] Building backend (server)..."
if [ ! -d "$SERVER_DIR" ]; then
    echo "Error: Directory '$SERVER_DIR' does not exist."
    exit 1
fi

cd "$SERVER_DIR" || exit 1

# Detect Operating System for informational purposes and specific build commands
OS_TYPE=$(uname -s)
echo "Detected Host OS: $OS_TYPE"
echo "Targeting Linux OS compilation..."

if [[ "$OS_TYPE" == "Linux"* ]]; then
    # Linux host
    GOOS=linux GOARCH=amd64 go build -o flow_collect_server .
elif [[ "$OS_TYPE" == "Darwin"* ]]; then
    # macOS host
    GOOS=linux GOARCH=amd64 go build -o flow_collect_server .
elif [[ "$OS_TYPE" == "MINGW"* || "$OS_TYPE" == "CYGWIN"* || "$OS_TYPE" == "MSYS"* ]]; then
    # Windows host (Git Bash, MSYS, Cygwin)
    export GOOS=linux
    export GOARCH=amd64
    go build -o flow_collect_server .
else
    # Fallback for any other OS
    GOOS=linux GOARCH=amd64 go build -o flow_collect_server .
fi

if [ $? -ne 0 ]; then
    echo "Error: Backend build failed."
    exit 1
fi
echo "Backend compiled successfully."
echo ""

# ====================================================
# Step 3: Move executable and print statistics
# ====================================================
echo "[3/3] Finalizing build..."
if [ ! -d "$DIST_DIR" ]; then
    echo "Error: Distribution directory '$DIST_DIR' does not exist."
    exit 1
fi

mv flow_collect_server "$DIST_DIR/"
if [ $? -ne 0 ]; then
    echo "Error: Failed to move the executable to $DIST_DIR"
    exit 1
fi
OUTPUT_FILE="$DIST_DIR/flow_collect_server"

# Calculate build duration
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Calculate File Hash (trying sha256sum, then shasum, then md5sum for cross-platform support)
if command -v sha256sum >/dev/null 2>&1; then
    FILE_HASH=$(sha256sum "$OUTPUT_FILE" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
    FILE_HASH=$(shasum -a 256 "$OUTPUT_FILE" | awk '{print $1}')
elif command -v md5sum >/dev/null 2>&1; then
    FILE_HASH=$(md5sum "$OUTPUT_FILE" | awk '{print $1}')
else
    FILE_HASH="N/A (No hashing utility found)"
fi

echo ""
echo "====================================================="
echo "                BUILD SUCCESSFUL!                    "
echo "====================================================="
echo "Output Path : $OUTPUT_FILE"
echo "Build Time  : ${DURATION} seconds"
echo "File Hash   : $FILE_HASH"
echo "====================================================="

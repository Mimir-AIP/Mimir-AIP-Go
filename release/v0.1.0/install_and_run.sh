#!/bin/bash
# Installer and Runner for Mimir-AIP PoC Pipeline
#
# This script automates the process of:
#   1. Cloning the Mimir-AIP repository
#   2. Setting up a Python virtual environment
#   3. Installing dependencies
#   4. Ensuring a minimal config.yaml for the PoC pipeline
#   5. Running the PoC pipeline via src/main.py
#
# Assumptions:
#   - Python 3 and git are installed and available in PATH
#   - The repository is public
#   - User has network access
#
# Usage:
#   cd release/v0.1.0
#   bash install_and_run.sh
#
# Design Decisions:
#   - Uses a virtual environment for isolation
#   - Creates config.yaml if missing, referencing src/pipelines/POC.yaml
#   - Provides clear error messages and logs actions
#   - Designed to be run from the release/v0.1.0 folder
#   - All paths are relative to the repository root

set -e  # Exit on error
set -u  # Treat unset variables as errors

REPO_URL="https://github.com/Mimir-AIP/Mimir-AIP"
REPO_DIR="Mimir-AIP"
CONFIG_FILE="config.yaml"
POC_PIPELINE_PATH="src/pipelines/POC.yaml"

# Function to print error and exit
fail() {
  echo "[ERROR] $1" >&2
  exit 1
}

# Check for required commands
type git >/dev/null 2>&1 || fail "git is required but not installed."
type python3 >/dev/null 2>&1 || fail "python3 is required but not installed."

# Clone repo if not already present
if [ ! -d "$REPO_DIR" ]; then
  echo "Cloning repository..."
  git clone "$REPO_URL" || fail "Failed to clone repository."
else
  echo "Repository already exists. Skipping clone."
fi
cd "$REPO_DIR"

# Set up virtual environment
if [ ! -d ".venv" ]; then
  echo "Creating Python virtual environment..."
  python3 -m venv .venv || fail "Failed to create virtual environment."
fi
source .venv/bin/activate

# Upgrade pip and install requirements
pip install --upgrade pip || fail "Failed to upgrade pip."
if [ -f "requirements.txt" ]; then
  pip install -r requirements.txt || fail "Failed to install requirements."
else
  fail "requirements.txt not found."
fi

# Create minimal config.yaml if missing
if [ ! -f "$CONFIG_FILE" ]; then
  echo "Creating minimal $CONFIG_FILE for PoC pipeline..."
  cat > "$CONFIG_FILE" <<EOF
settings:
  pipeline_directory: src/pipelines
  output_directory: output
  log_level: INFO
pipelines:
  - $POC_PIPELINE_PATH
EOF
else
  echo "$CONFIG_FILE already exists. Skipping creation."
fi

# Run the pipeline
echo "Running PoC pipeline..."
python3 src/main.py || fail "Pipeline execution failed."

echo "---"
echo "Mimir-AIP PoC pipeline completed. Logs are in mimir.log."
echo "Deactivate the virtual environment with 'deactivate' when done."

#!/bin/bash

# Extracts tool versions from Taskfile.yaml
# Usage: .github/scripts/extract-versions.sh

# Path to the Taskfile
TASKFILE="Taskfile.yaml"

# Check that the file exists
if [ ! -f "$TASKFILE" ]; then
  echo "Error: file $TASKFILE not found" >&2
  exit 1
fi

# Extract every variable from the vars section
echo "Extracting variables from Taskfile.yaml:"

# Find where the vars section starts and ends
VARS_START=$(grep -n "^vars:" "$TASKFILE" | cut -d: -f1)
if [ -z "$VARS_START" ]; then
  echo "Error: vars section not found in $TASKFILE" >&2
  exit 1
fi

VARS_START=$((VARS_START + 1))

# Look for the next section after vars, or the end of the file
NEXT_SECTION=$(tail -n +$VARS_START "$TASKFILE" | grep -n "^[a-z]" | head -1 | cut -d: -f1)
if [ -n "$NEXT_SECTION" ]; then
  VARS_END=$((VARS_START + NEXT_SECTION - 2))
else
  VARS_END=$(wc -l < "$TASKFILE")
fi

# Take every line of the vars section
VARS_SECTION=$(sed -n "${VARS_START},${VARS_END}p" "$TASKFILE")

# Associative array holding the variables
declare -A VARS

# Extract the name and value of each variable
while IFS= read -r line; do
  # Skip empty lines and comments
  if [[ "$line" =~ ^[[:space:]]*$ || "$line" =~ ^[[:space:]]*# ]]; then
    continue
  fi
  
  # Extract the name and the value
  if [[ "$line" =~ ^[[:space:]]*([A-Z_0-9]+):\ *\'([^\']*)\' ]]; then
    var_name=${BASH_REMATCH[1]}
    var_value=${BASH_REMATCH[2]}
    VARS["$var_name"]="$var_value"
    echo "- $var_name: ${VARS[$var_name]}"
  elif [[ "$line" =~ ^[[:space:]]*([A-Z_0-9]+):\ *\"([^\"]*)\" ]]; then
    var_name=${BASH_REMATCH[1]}
    var_value=${BASH_REMATCH[2]}
    VARS["$var_name"]="$var_value"
    echo "- $var_name: ${VARS[$var_name]}"
  elif [[ "$line" =~ ^[[:space:]]*([A-Z_0-9]+):\ *(.*) ]]; then
    var_name=${BASH_REMATCH[1]}
    var_value=${BASH_REMATCH[2]}
    VARS["$var_name"]="$var_value"
    echo "- $var_name: ${VARS[$var_name]}"
  fi
done <<< "$VARS_SECTION"

# Find the module list
if [ -n "${VARS[MODULES]}" ]; then
  MODULES="${VARS[MODULES]}"
  echo "- modules found: $MODULES"
else
  # If it is not in vars, look elsewhere (backwards compatibility)
  MODULES=$(sed -n 's/.*MODULES: \(.*\)/\1/p' "$TASKFILE" | head -1)
  echo "- modules (legacy format): $MODULES"
fi

# Export the GitHub Actions variables
if [ -n "$GITHUB_ENV" ]; then
  echo "Writing variables to GITHUB_ENV:"
  # Export every variable
  for var_name in "${!VARS[@]}"; do
    echo "$var_name=${VARS[$var_name]}" >> $GITHUB_ENV
    echo "  $var_name -> GITHUB_ENV"
  done
  # For compatibility add MODULES separately when it is not in vars
  if [ -z "${VARS[MODULES]}" ] && [ -n "$MODULES" ]; then
    echo "MODULES=$MODULES" >> $GITHUB_ENV
    echo "  MODULES -> GITHUB_ENV"
  fi
fi

if [ -n "$GITHUB_OUTPUT" ]; then
  echo "Writing variables to GITHUB_OUTPUT:"
  # Export every variable
  for var_name in "${!VARS[@]}"; do
    echo "$var_name=${VARS[$var_name]}" >> $GITHUB_OUTPUT
    echo "  $var_name -> GITHUB_OUTPUT"
  done
  # For compatibility add MODULES separately when it is not in vars
  if [ -z "${VARS[MODULES]}" ] && [ -n "$MODULES" ]; then
    echo "MODULES=$MODULES" >> $GITHUB_OUTPUT
    echo "  MODULES -> GITHUB_OUTPUT"
  fi
fi 
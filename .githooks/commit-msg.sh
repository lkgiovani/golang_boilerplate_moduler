#!/bin/bash

echo "Checking commit message format..."

# Get the commit message from the file
commit_msg_file=$1
commit_msg=$(cat "$commit_msg_file")

# Get the first line of the commit message
first_line=$(echo "$commit_msg" | head -n 1)

# For debugging
echo "Commit message first line: '$first_line'"

# Define the conventional commit format regex
# Format: type(scope): description
# Where type is one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert
conventional_format='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert)(\([a-zA-Z0-9_.-]*\))?: .+'

if ! [[ "$first_line" =~ $conventional_format ]]; then
  echo "[ERROR] Commit message does not follow conventional format."
  echo "Format should be: type(scope): description"
  echo "Where type is one of: feat, fix, docs, style, refactor, perf, test, build, ci, chore, revert"
  echo "Example: feat(auth): add login functionality"
  exit 1
fi

# Extract the keyword for emoji
KEYWORD=$(echo "$first_line" | awk '{print $1}' | sed -e 's/://')

# Add emoji based on commit type
case $KEYWORD in
  "feat"|"feat("*)
    EMOJI="✨"
    ;;
  "fix"|"fix("*)
    EMOJI="🐛"
    ;;
  "docs"|"docs("*)
    EMOJI="📚"
    ;;
  "style"|"style("*)
    EMOJI="💎"
    ;;
  "refactor"|"refactor("*)
    EMOJI="♻️"
    ;;
  "perf"|"perf("*)
    EMOJI="🚀"
    ;;
  "test"|"test("*)
    EMOJI="🧪"
    ;;
  "build"|"build("*)
    EMOJI="📦"
    ;;
  "ci"|"ci("*)
    EMOJI="👷"
    ;;
  "chore"|"chore("*)
    EMOJI="🔧"
    ;;
  "revert"|"revert("*)
    EMOJI="⏪"
    ;;
  *)
    EMOJI=""
    ;;
esac

# Only add emoji if one was selected
if [ -n "$EMOJI" ]; then
  # Prepend emoji to the first line
  new_first_line="$EMOJI $first_line"
  # Replace the first line in the commit message
  new_commit_msg=$(echo "$commit_msg" | sed "1s/.*/$new_first_line/")
  # Write the new commit message back to the file
  echo "$new_commit_msg" > "$commit_msg_file"
fi

echo "[PASS] Commit message format is valid."
exit 0

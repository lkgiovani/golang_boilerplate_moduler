#!/bin/bash

branch_name=$(git rev-parse --abbrev-ref HEAD)

if [[ ! "$branch_name" =~ ^(feature|fix|hotfix|docs|refactor|build|test)/.*$ ]]; then
  echo "Branch names must start with 'feature/', 'fix/', 'refactor/', 'docs/', 'test/' or 'hotfix/' followed by either a task id or feature name."
  exit 1
fi

exit 0

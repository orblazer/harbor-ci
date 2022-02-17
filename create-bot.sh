#!/bin/bash
USAGE="Usage: create-bot.sh <harbor-url> [project-id]"

if [[ -z $1 ]]; then # Check harbor url
  echo "[ERROR] Missing harbor url."
  echo $USAGE
  exit 1
elif [[ ! -z $2 ]]; then # Check project id
  if [[ ! $2 =~ ^-?[0-9]+$ || $2 -lt 0 ]]; then # Check if is num and is >= 0
    echo "[ERROR] Invalid project id."
    echo $USAGE
    exit 1
  fi
fi

projectId=$2
# Fix last slash
baseUrl=$1
if [[ $baseUrl != */ ]]; then
  baseUrl=$baseUrl/
fi

function buildPermissions {
  echo "{\"access\": [$(join_by , "${projectAccess[@]}")],\"kind\": \"project\",\"namespace\": \"$1\"}"
}

function buildPermission {
  local baseRes=""
  if [[ ! -z $projectId ]]; then
    baseRes="/project/$projectId/"
  fi

  echo "{\"action\":\"create\",\"resource\":\"$baseRes$1\",\"effect\":\"allow\"}"
}

function yesNo {
  # Prompts user with $1, returns true if response starts with y or Y or is empty string
  read -ep "$1 [Y/n] " YN

  [[ "$YN" == y* || "$YN" == Y* || "$YN" == "" ]]
}

function join_by {
  local IFS="$1"
  shift
  echo "$*"
}

projectAccess=()
permissions=()

# Ask credentials
echo "Enter credentials"
read -ep "  username: " username
read -sp "  password: " password
echo ""

# ask name
while :; do
  read -ep "Enter name: " name
  [[ -z $name ]] && continue

  break
done

# Ask duration
while :; do
    read -ep "Duration (in days, -1 = permanent): " duration
    [[ $duration =~ ^-?[0-9]+$ ]] || continue # Check if is num
    [[ $duration -gt -2 ]] || continue # Min -1

    break
done

# Ask project scope (when no project specified)
if [[ -z $projectId ]]; then
  while :; do
    read -ep "Project scope (# = all, for multiple: separate with comma (eg. project1,project2)):
> " project
    [[ -z $project ]] && continue

    break
  done
fi

# ask options
echo "Choose options:"
if yesNo "  allow scan"; then
  projectAccess+=("$(buildPermission scan create)")
  projectAccess+=("$(buildPermission artifact read)")
fi

# Build data
endpoint=""
data=""

if [[ -z $projectId ]]; then # No project specified
  endpoint="/robots"

  if [[ "$project" == *,*  ]]; then
    IFS=', ' read -ra projects <<< "$project"

    for item in "${projects[@]}"
    do
      permissions+=("$(buildPermissions $item)")
    done
  elif [[ "$project" == "#" ]]; then
    permissions+=("$(buildPermissions '*')")
  else
    permissions+=("$(buildPermissions $project)")
  fi

  data="{
  \"name\": \"$name\",
  \"level\": \"system\",
  \"duration\": $duration,
  \"permissions\": [$(join_by , "${permissions[@]}")]
}"
else
  endpoint="/projects/$2/robots"

  data="{
  \"access\": [$(join_by , "${projectAccess[@]}")],
  \"name\": \"$name\",
  \"expires_at\": $duration
}"
fi

echo ""

# Execute url
tempfile=$(mktemp)
code=$(curl -X 'POST' \
  -s "${baseUrl}api/v2.0$endpoint" \
  -u "$username:$password" \
  -H 'accept: application/json' \
  -H 'Content-Type: application/json' \
  -d "$data" \
  --write-out '%{http_code}' -o $tempfile)
content=$(cat $tempfile)
output="$(pwd)/robot.json"

if [[ $code != 200  ]]; then
  echo "[ERROR] Failed to create bot. Error:"
  echo "$content"
  rm -f $tempfile
  exit 1
else
  echo "[SUCESS] The bot as been created! Result is output in '$output'"
  mv $tempfile $output
fi

# Harbor CI

This project add some commands for include it in CI/CD.

## Features

- Scan artifact

## Setup the rebot

Due to missing GUI for some permissions (cf. [Harbor issue #8723](https://github.com/goharbor/harbor/issues/8723)), please use the follow step for create your bot.

**/!\ IMPORTANT**: The OIDC account is not supported.

1. Download script [create-bot.sh](./create-bot.sh)
2. execute script :

   ```sh
   ./create-bot.sh <harbor-url> [project-id]
   ```

   **Arguments** :

   - `harbor-url`: **(REQUIRED)** Define the harbor url
   - `project-id`: The project id for project bot

3. Fill the questions
4. Done!

## Commands usage

### Run in docker

```sh
docker run --rm orblazer/harbor-ci:latest [cmd arguments]
```

### Gitlab CI

```yml
include:
  - remote: 'https://raw.githubusercontent.com/orblazer/harbor-ci/main/utils/gitlab.yml'
```

**/!\\ Requirement**: Define CI/CD env variables `HARBOR_REGISTRY`, `HARBOR_REGISTRY_USER` and `HARBOR_REGISTRY_PASSWORD`

**Customization**:

- `FULL_IMAGE_NAME`: is the name of image want built and scan
- `SCAN_MAX_SEVERITY`: the maximum severity allowed in scan

## Scan artifact

This run artifact and return

### Usage

```sh
harbor-cli scan -username='<username>' -password='<password>' -url='<harbor-url>' <docker-image>
```

### Arguments

See [Common arguments](#common-arguments)

- `-severity=<severity>`: _(Default: `Critical`)_ The maximum severity level accepted.
  **Level**: `None`, `Low`, `Medium`, `High`, `Critical`

### Example

```sh
$ harbor-cli scan -username='robot$ci' -password='robot-password' -url='https://example.net/' -severity=High example/example-repo:latest

Scanning image...
+===============================================+
|                  Scan report                  |
+===============================================+
| Artifact url: https://example.net/harbor/projects/1/repositories/example-repo/artifacts/sha256:50d858e0985ecc7f60418aaf0cc5ab587f42c2570a884095a9e8ccacd0f6545c
|
| Vulnerability Severity: Critical
| Total: 3 (UNKNOWN: 0, LOW: 0, MEDIUM: 1, HIGH: 2, CRITICAL: 0)
| *Fixable: 3
|
| Scanned by: Trivy@v0.20.1
| Duration: 12s
+===============================================+
| /!\ The max severity level is reached !       |
|  Severity: High, Max severity: High           |
+===============================================+
exit status 1
```

## Common arguments

- `-username=<username>`: **(REQUIRED)** Define the harbor username
- `-password=<password>`: **(REQUIRED)** Define the harbor password
- `-url=<url>`: **(REQUIRED)** Define the harbor url

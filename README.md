# Pull Request Protector ⚔️

Blocks 3rd-party pull requests based on user and files changed.

⚠️ This plugin is in an alpha stage. It generally works 🤞 but needs some cleanup and testing. Happy to answer any questions in the [issues](https://github.com/nickofthyme/pull-request-protector-buildkite-plugin/issues).

## Context

The idea behind this plugin is that buildkite offers the option to enable builds to be triggered by pull requests from 3rd-party forks. This is great however, they provide minimal controls to guard what users can trigger builds, besides user-appened branch naming. Because buildkite runs every build from the local code in `.buildkite/`, this gives unfettered access to any user to run anything on your agents.

<img width="808" alt="image" src="https://user-images.githubusercontent.com/19007109/168648839-41119191-b394-4efd-a452-0b23150d4f86.png">

This plugin gives you the power to protect your builds given any of the following:

- **Users** - A list of specific usernames allowed, all others rejected.
- **Teams** - The user must belong to at least one team.
- **Membership** - The user must be a member of the base repo organization.
- **Collaborator** - The user must be a collaborator of the base repo.
- **Verified Commit** - The commit that triggered the build must be [verified](https://docs.github.com/en/authentication/managing-commit-signature-verification/displaying-verification-statuses-for-all-of-your-commits).

This plugin works by defining a command step prior to your buildkite pipeline, be it dynamic or not, that conditionally creates a block step before your pipeline. This block step much be approved by a buildkite user before continuing to run the pipeline. See example below.

## Example


```yml
steps:
  - label: "Check user"
    key: auth-check
    command: |
      # must have a command or step will fail but not required for this plugin
      echo "PR_PROTECTOR_BLOCKED: $$PR_PROTECTOR_BLOCKED" # use to run script based on result of conditions
    plugins:
      - nickofthyme/pull-request-protector#v0.1.0-alpha.12:
          verified: true # requires repo:read access
          collaborator: true # requires repo read access
          member: true # requires org:read access
          users: [nickofthyme]
          teams: ['my-team-slug'] # requires org:read access
          files: ['.buildkite/**', 'scripts/**/*.sh'] # files to trigger guard checks
          block_step: # The resulting block step if checks fail
            prompt: 'Pipeline is blocked for given user'
  - label: ":pipeline: Pipeline upload"
    depends_on: auth-check # must depend on auth-check step
    command: .buildkite/pipelines/pull_request/pipeline.sh
```

> All checks act as **AND** condition where all supplied checks must be true. Please open an issue if you desire an **OR** condition.

The above configuration would look this this when any user check fails.

<img width="959" alt="image" src="https://user-images.githubusercontent.com/19007109/168650879-71be28be-0ccf-416d-b121-7b336795e9f1.png">


## Configuration

### `users *[]string` (Optional) default: `[]`

A list of allowed usernames to match against the commit author. Supports glob patterns with wildcards (i.e. `*`) via https://github.com/gobwas/glob.

### `teams *[]string` (Optional) default: `[]`

A list of allowed teams to verify membership of commit author.

### `member bool` (Optional) default: `true`

Verifies the commit author is a member of the base repo where the job was triggered. Requires the github auth or token to have read access to the organization.

### `collaborator bool` (Optional) default: `true`

Verifies the commit author is a collaborator of the base repo where the job was triggered. Requires the github auth or token to have read access to the base repo.

### `files *[]string` (Optional) default: `[.buildkite/**]`

A list of protected files or patterns to trigger user checks. If none of these files were changed in the pull request, no checks will run and the pipeline script will not be blocked. Supports glob patterns with wildcards (i.e. `src/**/*.ts`) and `/` delimiter via https://github.com/gobwas/glob.

### `verified bool` (Optional) default: `true`

Verifies the latest commit is signed and the signature was successfully verified. See details [here](https://docs.github.com/en/authentication/managing-commit-signature-verification/displaying-verification-statuses-for-all-of-your-commits).

### `block_step interface{}` (Optional)

The block step used to block the pipeline, when any of the defined checks fail. This property is verified using the buildkite json schema definition for [`blockStep`](https://raw.githubusercontent.com/buildkite/pipeline-schema/master/schema.json#/definitions/blockStep), which is defined in their docs [here](https://buildkite.com/docs/pipelines/block-step). This property expects an object to define the block step and not the simplified string block step definition allowed by buildkite. This can be used to define `fields` and any other block step [attribute](https://buildkite.com/docs/pipelines/block-step#block-step-attributes), however be careful to not over constrain the step definition.

### `debug boolean` (Optional)

Enables verbose logging to help debug errors with plugin configuration.

### `gh_token_env string` (Optional)

The key of the environment variable containing the GitHub token. Defaults to `'GITHUB_TOKEN'`.

> Access should include `repo:read` and `org:read` permissions based on what checks are desired above.

### `gh_auth_env string` (Optional)

The key of the environment variable containing the GitHub App auth values as json. This is used only to authorize github app installations for making API requests.

```json
{
  "appId": 123456,
  "installationId": 12345678,
  "privateKey": "-----BEGIN RSA PRIVATE KEY-----\nhuiwehfriuehg...-----END RSA PRIVATE KEY-----"
}
```

> Access should include `repo:read` and `org:read` permissions based on what checks are desired above.

## Developing

To run plugin on local machine

```shell
go run *.go
```

To run plugin tests

```shell
docker compose build test-plugin && docker-compose run --rm test-plugin
```

To run plugin linting

```shell
docker compose build lint && docker-compose run --rm lint
```

## Contributing

1. Fork the repo
2. Make the changes
3. Run the tests
4. Commit and push your changes
5. Open a pull request

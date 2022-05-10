# Pull Request protector

Blocks 3rd-party pull requests based on user and files changed.

## Example

Add the following to your `pipeline.yml`:

```yml
steps:
  - label: ":pipeline: Pipeline upload"
    command: .buildkite/pipeline.sh
      - nickofthyme/pull-request-protector-buildkite-plugin#v0.1.0:
          files: ['.buildkite/**', 'scripts/**/*.sh']
          users: ['nickofthyme']
          teams: ['my-team-slug'] # requires org read access
          member: false # requires org read access
          collaborator: true # requires repo read access
          verified: true # requires repo read access
          block_step: # The resulting block step if checks fail
            prompt: 'Pipeline is blocked for given user'
```

> All checks act as **AND** condition where all supplied checks must be true. Please open an issue if you desire an **OR** condition.

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

### `verified interface{}` (Optional)

The block step used to block the pipeline, when any of the defined checks fail. This property is verified using the buildkite json schema definition for [`blockStep`](https://raw.githubusercontent.com/buildkite/pipeline-schema/master/schema.json#/definitions/blockStep), which is defined in their docs [here](https://buildkite.com/docs/pipelines/block-step). This property expects an object to define the block step and not the simplified string block step definition allowed by buildkite. This can be used to define `fields` and any other block step [attribute](https://buildkite.com/docs/pipelines/block-step#block-step-attributes), however be careful to not over constrain the step definition.

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

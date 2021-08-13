# Proposal: Parameterized Rules

Rules may define parameters that are resolved at runtime. These parameters are
then accessible in the rule definition and rule scripts, similar to how the
environment variable mechanism behaves today. Parameters may define a default
value or, if they do not, the parameter is understood to be required. When one
rule depends on another, arguments may be supplied to configure the dependee.
A mechanism is provided to resolve parameters from environment variables on the
host machine, where the environment variable name may or may not be the same as
the parameter name.

## Approach

 * The resolved parameters factor into the rule identity and their cache key.
 * Users should have the option to "connect" a parameter to an environment
   variable on the host system.
 * Parameter values may also be passed explicitly via the "with" keyword as
   part of a requires statement.
 * Users should have the option to resolve parameters at the project level
   and reuse the resolution across multiple components.
 * Extracting key-values from JSON environment variables could be helpful.
 * All environment variables should not be passed. Only those referenced as
   parameters should be passed.
 * Variables defined by a script should be resolved immediately before first use.
 * A derived component should be able to override variable value defaults.
 * Consider how conditional behavior should be introduced. Via a switch statement
   mechanism perhaps.
 * Parameters and their values should be shown to the user during execution by
   default. This behavior can be opted out of via `show_parameters: false`.
 * Parameters can be marked as sensitive with `sensitive: true` to avoid printing
   or otherwise capturing the parameter value in plaintext.

## Questions and Possible Answers

 * Do we use the existing `environment` map as part of this?
   * No, because the environment variables are passed to all rules today, while
     in this situation we want to control whether a rule uses a variable.
 * Default to "easy to use" mode, or default to "isolated builds" mode?
   * We need to maintain the current behavior as defaults. Perhaps opting into
     isolated builds on native executions via `isolated: true`.

## Example

```yaml
toolchain:
  items:
    - name: Go
      command: go version

# Pre-existing environment feature. These strings are passed to every rule in
# this component via environment variables. They are not dynamic.
environment:
  GO_BUILD_OPTS: -mod=readonly -ldflags="-s -w"

# New variables feature. These are resolved immediately before being used by a
# specific rule only. A rule references these by having a parameter defined that
# has `env: VAR_NAME` set. When VAR_NAME matches a variable named defined here,
# the variable is understood to be referenced. Consequently, these variables are
# effectively the same as Makefile variables, which act as environment variables
# when used in rules, and which may take on values from the host's shell.
variables:
  - name: KMS_KEY_ARN
    run: ./key-retrieval-script.sh # Run a script to resolve the variable
    global: true                   # Cache this variable across components
  - name: S3_BUCKET
    run: ./bucket-retrieval-script.sh
    global: true
  - name: AWS_PROFILE
    default: default              # Lowest precedence (if otherwise unspecified)
    env: AWS_PROFILE              # Higher precedence (if set in the environment)
  - name: S3_BUCKET_COMMENT
    value: The bucket is ${S3_BUCKET} # Example of referencing another variable

rules:
  build:
    docker:
      image: myimage/foo:1
    inputs:
      - go.mod
      - go.sum
      - "**/*.go"
    ignore:
      - "**/*_test.go"
    outputs:
      - ${NAME}.zip
    commands:
      - cleandir: build
      - run: GO111MODULE=on go build ${GO_BUILD_OPTS} -o build/${NAME}
      - zip:
          cd: build
          output: ../${OUTPUT}

  package:
    native: true    # Run on the host rather than in a container
    isolated: true  # Don't pass through env variables from the host shell (NEW)
    cached: false   # Don't cache the output file (NEW)
    local: true     # Output file goes into the same directory
    inputs:
      - cloudformation.yaml
    outputs:
      - cloudformation_deploy.yaml
    requires:
      - rule: build
    parameters:
      region:                     # Parameter with default values
        type: string
        default: us-east-1        # Lowest precedence (if otherwise unspecified)
        env: AWS_DEFAULT_REGION   # Higher precedence (if set in the environment)
      profile:
        type: string
        default: default
        env: AWS_PROFILE
      kms_key_arn:                # Required parameter: no default value
        type: string
      s3_bucket:                  # Required parameter: no default value
        type: string
    commands:
      - run: |
          aws cloudformation package \
            --region ${region} \
            --profile ${profile} \
            --kms-key-id ${kms_key_arn} \
            --s3-bucket ${s3_bucket} \
		    --template-file ${INPUT} \
		    --output-template-file ${OUTPUT}

  deploy:
    native: true
    isolated: true
    cached: false
    local: true
    inputs:
      - cloudformation_deploy.yaml
    requires:
      - rule: package
        with: # Example of directly passing arguments to another rule
          kms_key_arn: ${kms_key_arn}
          s3_bucket: ${s3_bucket}
    parameters:
      region:
        type: string
        default: us-east-1
        env: AWS_DEFAULT_REGION
      profile:
        type: string
        default: default
        env: AWS_PROFILE
      stack_name:
        type: string
        default: ${NAME}
      env_type:
        type: string
        default: dev
        env: ENV_TYPE
      kms_key_arn:
        type: string
      s3_bucket:
        type: string
    commands:
      - run: |
          aws cloudformation deploy \
            --region ${region} \
            --profile ${profile} \
            --s3-bucket ${s3_bucket} \
            --kms-key-id ${kms_key_arn} \
            --template-file cloudformation_deploy.yaml \
            --stack-name ${stack_name} \
            --capabilities CAPABILITY_NAMED_IAM \
            --no-fail-on-empty-changeset \
            --parameter-overrides \
                EnvType=${env_type}
```

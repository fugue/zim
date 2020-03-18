# Zim

Zim is a caching build system that is ideal for teams using monorepos containing
many components and dependencies. Its primary goal is fast incremental builds
across a team by leveraging a shared cache of rule outputs. It is entirely
language agnostic and has built-in support for cross-platform builds via Docker.

Components and rules are defined in a YAML definitions that are conceptually
similar to Makefiles. Each components may inherit from a base template, which
yields a simple mechanism to build many components in a consistent and
configurable manner.

## Why Zim?

Zim offers these advantages to teams developing in a monorepo:

 * Fast, parallel builds. Rules run only if inputs have changed and
   outputs are pulled from a shared cache if someone else built it already.

 * Trivially define how to build new component types. Define build steps
   for components in a few lines of YAML.

 * Gain the benefits of isolated build environments and cross-platform
   compilation via the built-in Docker support. Just specify the Docker
   image to be used when building a component.

 * Flexible input and output resource types. Currently Zim is able to work with
   both files and Docker images as natively supported resources.

 * Easy setup for a shared cache in S3 via an AWS CloudFormation stack.

 * Lightweight & easy to install. Zim is written in Go which means it
   consists of a single binary when built.

## Inspiration

This project draws inspiration from the core concepts of GNU Make along with the
caching strategy from [Buck](https://buck.build/),
[Bazel](https://bazel.build/), and [Please](https://please.build/index.html).

Like Make, Zim has a lightweight way to express new rules that define inputs,
outputs, and the commands needed to create the outputs.

Like Buck, Zim computes _Rule Keys_ which are used to determine whether the
output of a rule is already available in the cache, based on the combined
hashes of all the rule's inputs and configuration.

## Concepts

The following concepts are key to how Zim operates, although you can just skip
ahead to the *Getting Started* section below if you want to skate by for now.

 * **Project** - typically a Git repository that contains the source for
   multiple services.

 * **Component** - a directory in the monorepo which typically relates to a
   single library, binary, or microservice. In Zim, a Component is described
   by a `component.yaml` definition.

 * **Rule** - a definition that describes an action or build step. Each rule
   has optional input and output resources, dependencies, and associated
   commands.

 * **Key** - a key is computed for each rule which is unique to the current
   state of its inputs, dependencies, and rule configuration. This key is
   used when storing and retrieving output artifacts from the shared cache.

 * **Resource** - inputs and outputs from rules, which may be files or other
   types. Currently only files and Docker images are the two supported types.

 * **Provider** - new resource types may be added via providers. This consists
   of implementing a Go interface and recompiling Zim. Longer term, this could
   be changed to use IPC or another mechanism to make it easier to extend.

 * **Graph** - internally, Zim builds a directed acyclic graph (DAG) containing
   the rules the user asks to run, along with their transitive dependencies.
   This graph is created and processed by the scheduler in order to execute
   rules in order of their dependencies.

 * **Scheduler** - the scheduler is responsible for executing rules according
   to the DAG. The implementation uses Goroutines to parallelize execution of
   rules that are ready to run.

 * **Middleware** - rule execution is decorated and customized by middleware
   in Zim, similar to how middleware is used to customize HTTP handlers.
   Logging, caching, and uploads of artifacts are all accomplished via
   middleware.

## Getting Started

Install the Zim CLI by cloning this repo and running `go install` at the top level.
Run `zim -h` to see help regarding available commands and flags. Zim recognizes
when it is run within a Git repository and will automatically discover
`component.yaml` files within, which define components and their rules.

For each item in the repository that you would like to build with Zim, add
a `component.yaml` file in the corresponding directory. A simple example to
build a Go program is as follows.

```
name: myservice
rules:
  build:
    inputs:
    - "*.go"
    outputs:
    - ${NAME}
    command: go build -o ${OUTPUT}
```

With that definition saved, you can now enter `zim run build` to get it done.

```
$ zim run build
rule: myservice.build
cmd: go build -o ${OUTPUT}
rule: myservice.build in 1.347 sec [OK]
```

The outputs - an executable named `myservice` in this case - are stored in an
`artifacts` directory located at the root level of the repository.

## Creating the Shared Cache

Currently Zim supports using AWS infrastructure for its cache backend.
A CloudFormation stack containing an S3 bucket and a handful of other serverless
infrastructure is easily provisioned by using
[SAM](https://github.com/awslabs/aws-sam-cli).

Prerequisites:

 * Install the AWS and SAM CLIs: `pip install awscli aws-sam-cli`
 * [Download Go](https://golang.org/dl/) to build the Zim Lambdas

The following was tested with the following version of the SAM CLI:

```
$ sam --version
SAM CLI, version 0.45.0
```

With those dependencies installed, run the following to provision cache
infrastructure in AWS using a workflow guided by SAM:

```
$ make deploy
```

When the command completes, the URL of your Zim API is printed. This URL should
be saved to `~/.zim.yaml` as described in the following section.

## Developer Setup

Each developer should create the file `~/.zim.yaml` on their development
machine with two main variables:

  * The team API URL from `make deploy`
  * Your personal authentication token

With AWS credentials active for the account containing Zim, run the following
command to create an authentication token for each team member:

```
zim add token --email "joe@example.com" --name "Joe"
```

Each team member should now add the following to their `~/.zim.yaml`:

```
url: "TEAM_API_URL"
token: "MY_TOKEN"
```

Alternatively, you can use the environment variables `ZIM_URL` and `ZIM_TOKEN`.

## Cache Mode (Optional)

You may override the Zim CLI cache mode so that it only writes to the
cache and doesn't read. This may be useful for use in automated builds that
should populate the cache but not read from it.

To use this feature, set `cache-mode` in `~/.zim.yaml` as follows:

```
cache-mode: WRITE_ONLY
```

## Running Rules in Docker

To automatically run rules inside a Docker container, instead of on the host
directly, define a Docker image for each component and set the Docker option.

For example, to build a Go service in a container, you could use the following:

```
name: myservice
docker:
  image: circleci/golang:1.12.4
toolchain:
  items:
  - name: go
    command: go version
rules:
  build:
    inputs:
    - "*.go"
    outputs:
    - ${NAME}
    command: go build -o ${OUTPUT}
```

Be sure to provide the `--docker` flag or set it via one of the other
Viper mechanisms as discussed above.

With Docker enabled, Zim mounts the repository as a volume and sets the
component directory as the working directory when executing its rules.

Note the use of `toolchain` in the `component.yaml`. The above example includes
the output of `go version` in the *Rule Key* so that builds on different
architecture receive unique keys in the cache.

## Rule Keys

These keys are the basis for Zim caching. Zim uses SHA1 hashes to represent each
key. Specifically, the hash is computed on a JSON document containing the
following information for each rule:

* Project name
* Component name
* Rule name
* Docker image
* Input file relative paths and their SHA1 hashes
* Rule dependencies and their keys
* Environment variables set on the Component and Rule
* Toolchain
* Rule commands
* Zim version

This information uniquely identifies all the inputs and configuration used
by a rule. This means, prior to executing a rule, Zim can determine the current
rule key and check whether an output is stored with the that key in the cache.
If so, Zim downloads the output from the cache rather than executing the rule.

Zim assumes the rule commands are, in effect, a pure function. In practice
this isn't always the case, but is close enough. For example, when Python files
are compiled to `.pyc` a build timestamp is included, so the build will never
be exactly the same, even with the same file inputs.

If you would like to see a key for a given rule for debugging purposes, you
can use the following command:

```
zim key -r myservice.build
```

To retrieve the underlying information:

```
zim key -r myservice.build --detail
```

To learn more about Rule Keys, I recommend the
[Buck Docs](https://buck.build/concept/rule_keys.html) for now.

## Rule Dependencies

Zim supports dependencies between rules, both within a Component and across
Components. Collectively, rule dependencies form a directed acyclic graph that
Zim traverses when running rules.

To define a dependency, use the following syntax in a Component definition:

```
name: myservice
kind: go
rules:
  build:
    requires:
    - component: my_library_a
      rule: build
    - component: my_library_b
      rule: build
    inputs:
    - "*.go"
    outputs:
    - ${NAME}
    command: go build -o ${OUTPUT}
```

The above example declares two dependencies from `myservice.build` to
`my_library_a.build` and `my_library_b.build`. Consequently, if a user entered
`zim run build -c myservice`, Zim will first build the two libraries, and only
when those complete successfully will it build `myservice`.

When declaring a requirement, if the Component is omitted, then it is assumed
to be referring to another named Rule in the current Component.

## Variables

Rules are able to leverage environment variables from two sources. First,
developers may define an environment for each Component using the following
syntax, which results in these variables being available when all rules of
this Component are executed:

```
name: myservice
environment:
  RETRY_COUNT: 3
  FOO: "bar"
```

Second, a handful of environment variables are automatically inserted by Zim
in order to provide Rule commands some context:

 * `COMPONENT` - the Component name, e.g. "myservice"
 * `NAME` - the Component name, e.g. "myservice"
 * `KIND` - the Component kind, e.g. "go"
 * `RULE` - the Rule name, e.g. "build"
 * `NODE_ID` - ID in Graph for the Rule, e.g. "myservice.build"
 * `INPUT` - the relative path to the first input
 * `INPUTS` - relative paths to all inputs (space separated)
 * `OUTPUT` - the relative path to the first output
 * `OUTPUTS` - relative paths to all outputs (space separated)
 * `DEP` - the relative path to the first dependency
 * `DEPS` - relative paths to all dependencies (space separated)

The input, output, and dependency variables all follow the same order as
present in the Component YAML definition.

As a trivial example, if a Rule lists "*.go" as an input and the Component has
one Go file in the directory named "main.go", then `INPUT=main.go` is set in
the Rule environment.

These automatic variables are similar in purpose to those found in GNU Make.
For example, `INPUT` is equivalent to `$<` and `OUTPUT` is equivalent to `$@`.

Support for more dynamic variables will be added as needed for new use cases.

## Commands

Here are the most commonly used commands.

 * `zim run` - runs rules specified via args or flags
 * `zim key -r myservice.build` - shows the current key for a rule
 * `zim list rules` - shows Rules in the Project
 * `zim list components` - shows Components in the Project
 * `zim list inputs -c myservice` - shows inputs used by a Component
 * `zim list env` - shows Zim Viper configuration
 * `zim add token` - create a new authentication token

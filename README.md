# anypipe

Portable pipelines with minimal dependencies.

### Why?
After some (*a lot of*) frustrations trying to use Dagger in production environments with strict security policies, decided to try and make something with minimal dependencies and requirements.

### How?
An abstraction for pipeline/job/step definitions, integrated with a wrapper around Docker's Golang API. Meaning the only requirement/dependency to run a pipeline is to have a running docker daemon, and permissions to talk to it.

### Usage

`go get github.com/notmiguelalves/anypipe@vX.Y.Z` - Then you can leverage the pipeline abstractions and the docker utilities. To see an end-to-end example on how it is used I'd recommend having a looking at the integration tests :[pipeline abstractions](pkg/anypipe/pipeline_integration_test.go) and [docker utilities](pkg/dockerutils/docker_integration_test.go).

The `dockerutils` package is essentially a wrapper on top of Docker's Golang API, offering some higher level abstractions for container life-cycle management, file transfer between **host-container** and **container-container**, and command execution.

The `anypipe` package offers ways to define pipelines in code:

```go
pipeline := NewPipelineImpl(ctx, logger, "Test Pipeline")
pipeline.WithSequentialJobs(
	NewJobImpl("Example Job", "alpine:latest").
		WithStep("lint", lintStepImpl).
		WithStep("test", testStepImpl),
)
pipeline.Run(inputs)
```

Each step needs to implement the following signature
```go
func(du dockerutils.DockerUtils, c *dockerutils.Container, variables map[string]interface{}) error
```

A container with the user-defined image is created for each job (in the above snippet it would be `alpine:latest`. This container is then passed to each step as an argument, allowing you to execute operations in and/or out of the container.

A pipeline receives a set of input variables, which are passed to every job and step. These variables can be mutated, removed, added, etc. This means that if in step `lint` we wrote to the variables, those writes would be visible in step `test`.

When execution finishes (regardless if success or error) it outputs an overview of the steps, results and durations. If running in a GitHub environment, it will also generate a summary of the run in the job summary annotations:

# bad job
| Result | Step | Duration |
| --- | --- | --- |
| FAIL | step1 | 21.63µs |
| SKIP | step2 | 0s |
# test_job_1
| Result | Step | Duration |
| --- | --- | --- |
| PASS | step1 | 38.608751ms |
| PASS | step2 | 33.878416ms |


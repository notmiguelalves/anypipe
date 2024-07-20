# anypipe

Portable pipelines with minimal dependencies.

### Why?
After some (*a lot of*) frustrations trying to use Dagger.io in production environments with strict security policies, decided to try and make something with minimal dependencies and requirements.

`Let's see if it actually turns out that way :)`

### How?
Essentially a wrapper around Docker's Golang API. Meaning only dependency **should** be having a running docker/podman daemon, and permissions to talk to it.

### Roadmap
1. What I would consider base functionality in place
    - container creation
    - pipeline steps definition/API
    - interaction between host<->container
        - command execution
        - FS mounting
        - environment variables
    - nice CLI output
2. Unit and Integration test pipelines
3. Automated releases
4. Ensure podman compatibility
5. Open up the repo to public. Open source, very permissive license. Contributing guidelines.
6. Better integration with Github - different steps defined in the pipeline should also reflect as different steps in Github Workflows UI
7. ???

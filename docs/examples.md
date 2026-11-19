### Start builds and wait for them

Start two Workflows in parallel, share an env with them, and wait for both to finish. The step fails if any of them fails, and aborts the others.

```yaml
steps:
- build-router-start:
    inputs:
    - access_token: $BITRISE_TOKEN
    - workflows: |-
        unit_tests
        ui_tests
    - environment_key_list: MY_SHARED_ENV
    - wait_for_builds: "true"
    - abort_on_fail: "yes"
```

### Start builds, run other steps, then wait

Start a Workflow, run other steps of the parent Workflow while it runs, then wait for it with the **Bitrise Wait for Build** step.

```yaml
steps:
- build-router-start:
    inputs:
    - access_token: $BITRISE_TOKEN
    - workflows: ui_tests
- script:
    inputs:
    - content: echo "Running in parallel with ui_tests"
- build-router-wait:
    inputs:
    - access_token: $BITRISE_TOKEN
    - buildslugs: $ROUTER_STARTED_BUILD_SLUGS
```

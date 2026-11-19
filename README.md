# Bitrise Start Build

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/bitrise-step-build-router-start?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/bitrise-step-build-router-start/releases)

Starts a new build with the specified Workflow(s).

<details>
<summary>Description</summary>

The Step starts a new build of the same app with specified Workflow(s). On its own, the Step can be used to start multiple builds that run in parallel. In combination with the **Bitrise Wait for Build** Step, you can also add additional Steps to run in parallel while the triggered builds are running. For an example, check out our [Start builds from a parent Workflow](https://devcenter.bitrise.io/en/steps-and-workflows/generic-workflow-recipes/start--parallel--builds-from-the-workflow.html##) Workflow Recipe.
You can put this Step in a Workflow that is triggered by a code push or pull request and use this Step to trigger multiple other builds at the same time.

To do that, you need multiple concurrencies - at least two! You also need a Personal Access Token to use the Step: [check out our guide](https://devcenter.bitrise.io/getting-started/account-security/#generating-personal-access-tokens-manually) on how to generate the token.

If manual build approval is enabled for the project, this step won't be able to start new builds automatically.

### Configuring the Step

Before you run a build with this Step, make sure you have your Personal Access Token stored in a [Secret Environment Variable](https://devcenter.bitrise.io/builds/env-vars-secret-env-vars/#adding-a-secret-env-var).

Note that all builds you trigger using this Step will have the same build number!

   1. Set up at least two Workflows in the Workflow Editor: one will contain the Step, the other will be triggered by the Step.
     You can set up as many Workflows as you wish: your number of concurrencies limit the number of Workflows you can run simultaneously.
  2. Add the **Bitrise Start Build** Step to the part of the Workflow where you want to trigger another build.
  3. Add the Secret Env Var storing your Personal Access Token to the **Bitrise Access Token** input of the **Bitrise Start Build** Step: click the **Select secret variable** button, and choose the key you created.
  4. Find the **Workflows** input of the **Bitrise Start Build** Step, and add the Workflow(s) you want to run.
  5. In the **Environments to share** input, add Environment Variables (Env Vars) that you want to share between the builds triggered by the Step.
  6. (Optional) You can set the **Wait for builds** input to `true` if you would like to pause the current build until the triggered build(s) are finished.
  7. (Optional) You can add any Step you would like to run in parallel while the triggered Workflow(s) are running in the parent Workflow.
     Make sure to add the **Bitrise Wait for Build** Step to the end of Workflow.
  8. Add the Secret Env Var storing your Personal Access Token to the **Bitrise Access Token** input of the **Bitrise Wait for Build** Step: click the **Select secret variable** button, and choose the key you created.
  9. In the **Build slugs** input, define the builds for the Step.
     The build slugs you need are stored by the **Bitrise Start Build** Step in the `$ROUTER_STARTED_BUILD_SLUGS` Env Var. As long as the builds defined by the slugs are running, the Step will hold the build it is running in. The build will fail if any of the builds included in the Step fail.
  10. Optionally, you can save the build artifacts from the builds and configure the Step to abort all builds if any of the builds fail:
     - In **The path of the build artifacts** input, set where you'd like to save the artifacts.
       The value of the input must be an actual, existing path, ending with `/`.
     - Set the **Abort all builds if one fails** input to either `yes` or `no`.
     Please note that the build artifacts of child Workflows - Workflows triggered by this Step - will be only available if the child Workflow contains a **Deploy to Bitrise.io** Step.

### Troubleshooting

- Make sure you have both the **Bitrise Start Build** and the **Bitrise Wait for Build** Steps in the right place in your initial Workflow.
- Always check all your builds: if your concurrencies are taken up by other builds, this Step cannot start new ones.
- Your Personal Access Token can expire! Make sure it's still valid - without it, you can't start new builds with this Step.

### Useful links

- [Starting parallel builds with a single Workflow](https://devcenter.bitrise.io/builds/triggering-builds/trigger-multiple-workflows/)
- [Start builds from a parent Workflow](https://devcenter.bitrise.io/en/steps-and-workflows/generic-workflow-recipes/start--parallel--builds-from-the-workflow.html##)

### Related Steps

- [Bitrise Wait for Build](https://www.bitrise.io/integrations/steps/build-router-wait)
- [Bitrise Run](https://www.bitrise.io/integrations/steps/bitrise-run)
- [Build Status Change](https://www.bitrise.io/integrations/steps/build-status-change)
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

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


## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `access_token` | Your access token  To acquire a `Personal Access Token` for your user, sign in with that user on [bitrise.io](https://bitrise.io), go to your `Account Settings` page, and select the [Security tab](https://www.bitrise.io/me/profile#/security) on the left side.  | required, sensitive |  |
| `workflows` | The Workflow(s) to start. One Workflow per line. | required |  |
| `environment_key_list` | The keys of the envs which will be shared with the triggered Workflows.  **FORMAT** Separate the keys with new line. E.g:  `ENV_1`  `ENV_2`  `ENV_3` |  |  |
| `wait_for_builds` | Should the Step wait for the builds to be finished? |  | `false` |
| `build_artifacts_save_path` | The path where build artifacts will be saved to if the **Wait for builds** input is set to `true`.  Note that when this Step triggers a Workflow, the artifacts of the triggered Workflow are only available to the Workflow that contains this Step, and not other Workflows. The triggered Workflow MUST have a **Deploy to Bitrise.io** Step to deploy build artifacts!  |  |  |
| `abort_on_fail` | Abort all started builds if any build fails | required | `no` |
| `verbose` | You can enable the verbose log for easier debugging. | required | `no` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `ROUTER_STARTED_BUILD_SLUGS` | Newline separated list of started build slugs. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/bitrise-step-build-router-start/pulls) and [issues](https://github.com/bitrise-steplib/bitrise-step-build-router-start/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Note: this step's end-to-end tests (defined in e2e/bitrise.yml) are working with secrets which are intentionally not stored in this repo. External contributors won't be able to run those tests. Don't worry, if you open a PR with your contribution, we will help with running tests and make sure that they pass.


Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)

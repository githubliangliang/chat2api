# Setup In-Process Restart Design

## Context

On first launch, `sub2api` starts a setup-only HTTP server. After the web setup
wizard writes the configuration, the setup handler currently calls
`sysutil.RestartServiceAsync`. On Linux this exits the process and assumes a
systemd or container restart policy will start it again. A user who launches
the application directly with `./sub2api` is therefore left with a stopped
process.

## Goal

After a successful web setup, transition the directly launched process from
setup mode to normal server mode without requiring an external supervisor.
The browser must receive the successful installation response before the setup
server stops, and the existing frontend readiness polling must continue to
redirect the user to the login page when the normal server is ready.

## Scope

- Change only the browser-based first-run setup flow.
- Preserve `./sub2api -setup` behavior.
- Preserve the authenticated admin restart endpoint behavior.
- Continue to support systemd and container deployments.
- Do not introduce a general-purpose process supervisor or Unix-specific
  process replacement.

## Design

The setup route registration accepts an installation-complete callback owned by
the server entry point. The install handler invokes that callback asynchronously
after it has completed installation and returned a success response.

`runSetupServer` owns a completion channel and the setup HTTP server lifecycle.
It starts the HTTP server, then waits for either an installation-complete signal
or a server error. On completion, it gracefully shuts down the setup server and
returns a result indicating that setup finished successfully.

The main entry point checks that result. When setup completed, it calls
`runMainServer`, which reloads the generated `config.yaml` through the existing
bootstrap path and starts the fully initialized application in the same process.
No state from the setup server is reused beyond the persisted configuration and
database contents.

The completion signal is non-blocking and idempotent so a slow shutdown or a
duplicate callback cannot block the HTTP handler. The setup HTTP response is
written before shutdown begins. The existing frontend polls `/setup/status`;
once the normal server responds with `needs_setup: false`, it redirects to
`/login` as it does today.

## Error Handling

- Installation validation or persistence failures return the existing error
  response and do not signal completion.
- A setup-server listen failure remains fatal and does not start normal mode.
- A graceful shutdown failure is logged; startup does not overlap the old
  listener. The process exits rather than attempting to run two HTTP servers.
- A normal-server initialization failure follows the existing fatal startup
  path and reports the configuration or dependency error in logs.

## Testing

- A handler test verifies that successful installation schedules exactly one
  completion notification after returning success.
- A handler test verifies that failed installation does not notify completion.
- A server lifecycle test verifies that an installation-complete signal causes
  the setup server path to return a completed result.
- Existing setup package tests and the backend test suite must remain green.
- A manual smoke test launches `./sub2api`, completes the web wizard, observes
  the same PID continue into normal mode, and confirms the browser reaches the
  login page without manually restarting the binary.

## Success Criteria

With no systemd or container supervisor, completing the browser setup wizard
automatically makes the normal Sub2API service available and redirects the user
to login. No manual second invocation of `./sub2api` is required.

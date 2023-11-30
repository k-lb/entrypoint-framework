# entrypoint-framework

[![Go Reference](https://pkg.go.dev/badge/github.com/k-lb/entrypoint-framework.svg)](https://pkg.go.dev/github.com/k-lb/entrypoint-framework)

entrypoint-framework provides building blocks for container entrypoints that needs not only to run, but also to manage application. This is valid especially for environments where restarting containers or pods (on events like process crash or configuration change) doesn't meet time constrains.

## Design decisions

### Event based

Containers run in environments where frequent changes occurs. It means that entrypoint should react on such change as fast as possible. Another important factor is CPU usage. As entrypoint is running and managing main application then it should use as little resources as possible. Thus instead of actively monitoring for changes logic should wait for an event (like changing a file or stopping a process).

### Consistent state

There may be situations where event notification occurs but from the business perspective action isn't finished. The most prominent example is updating many configuration files. In such case only part of configuration may be applied. Thus all actions that entrypoint does must be atomic to keep consistent state.

## Handlers

entrypoint-framework defines handlers. Currently following handlers are defined:
- process handler,
- configuration handler,
- activation handler.

These are interfaces but concrete implementation is also provided. Thanks to that we achieve flexibility, as entrypoint logic can be written once and appropriate handlers can be swapped to better fit use case changes (e.g. configuration format change).

### Process Handler

It is used to spawn required processes. It notifies when the managed process starts, ends and allows to stop, kill or send signal to managed process.

### Configuration Handler

Its duty is to watch configuration files and react on changes. In case of a change user is notified via channel. If change is valid entrypoint may call `Update()` method that will trigger the update. Results of the update are sent on a separate channel. Currently two implementations are provided:
1. single file configuration handler;
2. tarred configuration handler.

*Single file configuration handler* should be used when there is only one configuration file. If there are multiple files then those should be provided as a tar archive and *tarred configuration handler* should be used. It will extract and compare new configuration with the existing one informing about the changes via channel.

### Activation Handler

This handler can be used in situations where container may be running (ready) but not performing its tasks. For example there may be active and backup pods. In such case on every state change handler will send notification with current state via a channel. Currently there is one handler implemented. It watches if file that denotes if container should be active exists.

## Creating entrypoints

Developers are provided with standard godoc API documentation. Additionally there is an exemplary entrypoint under [test directory](https://github.com/k-lb/entrypoint-framework/tree/main/test). It may be used as a model when creating an entrypoint.

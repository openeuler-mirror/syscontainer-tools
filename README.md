# isulad-tools

## Introduction

**isulad-tools** is a fully customized tool,
it is a small auxiliary tool which is expected to work with isulad with hook support,
and provides enhanced functions which is inappropriate to be included in isulad itself.

The project includes two main parts: `isulad-tools` and `hooks`.
`isulad-tools` is used for dynamically operating on running containers,
and `hooks` is used for executing user defined program at some special timepoint of container's lifecycle.

## Hooks

We provide isulad hooks function.

* isulad-hooks:
 1. insert block devices added by isulad-tools into container when container restarts(prestart state).
 2. insert network interfaces and route rules added by isulad-tools into container when container restarts(prestart state).
 3. remove udev rules and leaking network interfaces when container stops(post-stop state).
 4. handling oci relabel for container in prestart and post stop state.

You could use hook spec to customise your hooks.
For detailed information, See [introduction of isulad-hooks](hooks/isulad-hooks/README.md)

## isulad-tools

Basic usage of `isulad-tools`:

```
NAME:
   isulad-tools - Enhanced tools for IT isulad

USAGE:
   isulad-tools [global options] command [command options] [arguments...]

VERSION:
   v0.9
commit: e39c47b1d0403fd133c49db13ab6df7e5d53a21b

COMMANDS:
    add-device          add one or more host devices to container
    add-nic             create a new network interfaces for container
    add-path            add one or more host paths to container
    add-route           add a new network route rule into container
    relabel             relabel rootfs for running SELinux in system container
    remove-device       remove one or more devices from container
    remove-nic          remove a network interface from container
    remove-path         remove one or more paths from container
    remove-route        remove a network route rule from container

GLOBAL OPTIONS:
   --debug                              enable debug output for logging
   --log "/dev/null"                    set the log file path where internal debug information is written
   --log-format "text"                  set the format used by logs ('text' (default), or 'json')
   --syslog-service "unix:///dev/log"   set syslog service
   --help, -h                           show help
   --version, -v                        print the version
```

For usage of each command, you can check with `--help`, e.g. `isulad-tools add-device --help`

## Contributions

As this is a fully customized tool, I don't think anyone will be interested in contributing to this project,
but we welcome your contributions. Before contributing, please make sure you understand our needs and
make a communication with us.

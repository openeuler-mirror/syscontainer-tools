# syscontainer-tools

## Introduction

**syscontainer-tools** is a fully customized tool.
It is a small auxiliary tool which is expected to work with iSulad with hook support,
and provides enhanced functions which are inappropriate to be included in iSulad itself.

The project includes two main parts: `syscontainer-tools` and `hooks`.
`syscontainer-tools` is used for dynamically operating on running containers,
and `hooks` is used for executing user-defined programs at some special time points of the container's lifecycle.

## Hooks

We provide syscontainer hooks function.

* syscontainer-hooks can:
 1. Insert block devices added by syscontainer-tools into a container when the container restarts (pre-start state).
 2. Insert network interfaces and route rules added by syscontainer-tools into a container when the container restarts (pre-start state).
 3. Remove udev rules and leaked network interfaces when a container stops (post-stop state).
 4. Handle OCI relabeling for containers in pre-start and post-stop states.

You can use hook-spec to customize your hooks.
For detailed information, see [syscontainer-hooks](hooks/syscontainer-hooks/README.md).

## syscontainer-tools

Basic usage of `syscontainer-tools`:

```
NAME:
   syscontainer-tools - Enhanced tools for IT iSulad

USAGE:
   syscontainer-tools [global options] command [command options] [arguments...]

VERSION:
   v0.9
commit: e39c47b1d0403fd133c49db13ab6df7e5d53a21b

COMMANDS:
    add-device          add one or more host devices to the container
    add-nic             create network interfaces for the container
    add-path            add one or more host paths to the container
    add-route           add a new network route rule to the container
    relabel             relabel rootfs for running SELinux in the system container
    remove-device       remove one or more devices from the container
    remove-nic          remove a network interface from the container
    remove-path         remove one or more paths from the container
    remove-route        remove a network route rule from the container

GLOBAL OPTIONS:
   --debug                              enable debug output for logging
   --log "/dev/null"                    set the log file path where internal debug information is written
   --log-format "text"                  set the format used by logs ('text' (default), or 'json')
   --syslog-service "unix:///dev/log"   set the syslog service
   --help, -h                           show help information
   --version, -v                        print the version
```

For usage of each command, you can check with `--help`, e.g. `syscontainer-tools add-device --help`

## Contributions

As this is a fully customized tool, we don't think anyone will be interested in contributing to this project,
but we welcome your contributions. Before contributing, please make sure you understand our needs and
make a communication with us: isulad@openeuler.org.

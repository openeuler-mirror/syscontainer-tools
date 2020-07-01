# syscontainer-hooks

This is a simple custom syscontainer hook for our own need,
it interacts with isulad as a multifunctional hook.

 1. allow user to add your own devices or binds into the container and update device Qos for container(device hook in prestart state).
 2. allow user to remove udev rule which added by syscontainer-tools when container is exiting(device hook in post-stop state).
 3. allow user to add network interface and route rule to container(network hook in prestart state).
 4. allow user to remove network interface on host when container is exiting(network hook in post-stop state).
 5. allow user to do oci relabel for container in both prestart and post-stop state for container.

Actually, this hook only handles the container restart process, we use syscontainer-tools to
add device/binds/network interface/route rule to container. And syscontainer-tools will save the device/network config to disk.
And the hook will make sure the resources you added to container will be persistent after restart.

Rename it to your favourite name afterwards.

## build

To build the binary, you need to download it then run 

```
# make
# sudo make install
```

Note: make install will install the binary into your "/usr/bin",
it's not a mandatory step, make your own choice for your convenience :)


## customise hook service

We could use `syscontainer-hooks` to customise the hook service.
```
Usage of syscontainer-hooks:
  -log string
        set output log file
  -state string
        set syscontainer hook state mode: prestart or poststop
  -with-relabel
        syscontainer hook enable oci relabel hook function
```

As block device and network interface are both in our requirement, so these two function are mandantory.
We could use `--with-relabel=true` to add oci-relabel hook service for container.
We could use `--state` to specify which state the hook will be running in.

Full hook config:
[hook spec example of syscontainer-hooks](hooks/syscontainer-hooks/example/hookspec.json)

## Try it!

First you need an enhanced `isula` with newly added `--hook-spec` flag,
after that, you can run it like this:

1.run isulad container with hook spec in `example` directory

```
$ isula run -d --name test_device --hook-spec $PWD/example/hookspec.json busybox sleep 20000
```
2.use syscontainer-tools to add device or binds to container

```
syscontainer-tools add-device test_device /dev/zero:/dev/test_zero:rwm /dev/zero:/dev/test_zero2:rwm
```

3.restart the container. to check the device is still in container.

```
isula restart test_device
```

Let's check the [`hookspec.json`](example/hookspec.json) file:

```
{
        "prestart": [
            {
                "path": "/var/lib/isulad/hooks/device-hook",
                "args": ["device-hook"],
                "env": []
            }
        ],
        "poststart":[],
        "poststop":[]
}
```

# Contact me

If you have any question or suggestion, please contact: isulad@openeuler.org.
Also welcome for any issue or MR! Thanks!

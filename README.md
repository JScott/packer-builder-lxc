packer-builder-lxc
==================

Lxc builder for [Packer](https://www.packer.io/), with working provisioning.


# OS Support

As building platform only Linux is supported, at this point Ubuntu and Debian is tested. Please consult your lxc templates so you can pass the right parameters. All OSes should be buildable if they are supported by your lxc templates.

## Debian

Wheezy is supported with lxc 1.0.0. An older version is shipped, with a broken lxc-attach command, so it's not supported. A backport deb can be created easily following [this guide](https://wiki.debian.org/SimpleBackportCreation). Debian does not ship network support by default for lxc, you have to configure it manually. See the [vagrant-lxc wiki](https://github.com/fgrehm/vagrant-lxc/wiki/Usage-on-debian-hosts#network-configuration) for a detailed howto.

If you want to do anything during provisioning that needs network access, you'll need to edit `/etc/lxc/default.conf` to bring the network up on new containers:

```
lxc.network.type=veth
lxc.network.link=lxcbr0
lxc.network.flags=up
```

If your containers do not get an ip address from dhcp you need to turn off checksum offloading on the bridge:

```bash
/sbin/ethtool -K lxcbr0 tx off
```

If you're done with all of these you are ready to build containers on wheezy!

## Ubuntu

Everything above saucy should be supported (saucy is tested). The default configuration is good to go!

If your containers do not get an ip address from dhcp you need to turn off checksum offloading on the bridge:

```bash
/sbin/ethtool -K lxcbr0 tx off
```


Building from source
====================
Install golang-go: https://golang.org/doc/install#install

Building will require Go 1.6 (maybe 1.7?) or higher.

Install dependencies:
* [gox](https://github.com/mitchellh/gox)
* [go-fs](https://github.com/mitchellh/go-fs)
* [multistep](https://github.com/mitchellh/multistep)
* [this package!](https://github.com/JScott/packer-builder-lxc)

```bash
go get github.com/mitchellh/gox
go get github.com/mitchellh/go-fs
go get github.com/mitchellh/multistep
go get github.com/JScott/packer-builder-lxc
```

Remove a few vendors from Packer's new structure that will break packer-builder-lxc:
```bash
rm -rf ~/gopath/src/github.com/hashicorp/packer/vendor/github.com/mitchellh/multistep
rm -rf ~/gopath/src/github.com/hashicorp/packer/vendor/github.com/mitchellh/mapstructure
```

Go to the source directory, usually it is in `~/gopath/src/github.com/ustream/packer-builder-lxc`
```bash
cd ~/gopath/src/github.com/ustream/packer-builder-lxc
```

Build binary file with `gox` for desired platform:
```bash
gox -os=linux -arch=amd64 -output=pkg/{{.OS}}_{{.Arch}}/packer-builder-lxc
```

Copy output binary file to desired location, for example to `/vagrant/packer`:
```bash
cp pkg/linux_amd64/packer-builder-lxc /vagrant/packer
```

Installation
============

Add the executable to your `.packerconfig` [Core Config](https://www.packer.io/docs/other/core-configuration.html), if you use custom path like `/vagrant/packer/packer-builder-lxc`:
```json
{
  "builders": {
    "lxc": "/vagrant/packer/packer-builder-lxc"
  }
}
```
From now you should be able to use `lxc` in packer builders.

Example packer templates
========================

Building wheezy on saucy:

```json
{
  "builders": [
    {
      "type": "lxc",
      "config_file": "lxc/config",
      "template_name": "debian",
      "template_environment_vars": [
        "MIRROR=http://http.debian.net/debian/",
        "SUITE=wheezy"
      ],
      "target_runlevel": 3
    }
  ]
}
```

The `config_file` is an lxc config file which will be bundled with the machine. You can create your own or just grab the `debian` or `ubuntu` from [vagrant-lxc-base-boxes](https://github.com/fgrehm/vagrant-lxc-base-boxes/tree/master/conf).


Building wheezy on wheezy:

```json
{
  "builders": [
    {
      "type": "lxc",
      "config_file": "lxc/config",
      "template_name": "debian",
      "template_parameters": ["--arch", "amd64", "--release", "wheezy"],
      "template_environment_vars": [
        "MIRROR=http://http.debian.net/debian/"
      ],
      "target_runlevel": 3
    }
  ],
  "provisioners": [
    {
      "type": "shell",
      "only": ["lxc"],
      "environment_vars": [
        "DISTRIBUTION=debian",
        "RELEASE=wheezy"
      ],
      "scripts": [
        "scripts/lxc/base.sh",
        "scripts/lxc/vagrant-lxc-fixes.sh"
      ]
    }
  ],
  "post-processors": [
    {
      "type": "compress",
      "output": "output-vagrant/wheezy64-lxc.box"
    }
  ],
}
```

Note the differences in template parameters/envvars!

Creating and cloning a build:

```json
{
  "builders": [
    {
      "type": "lxc",
      "target_runlevel": 2,
      "container_name": "base",
      "config_file": "lxc.config",
      "init_timeout": "120s",
      "template_parameters": [
        "-d",
        "ubuntu",
        "-r",
        "trusty",
        "-a",
        "amd64"
      ],
      "template_name": "ubuntu",
      "cleanup_first": true
    }
  ]
}
```

```json
{
  "builders": [
    {
      "type": "lxc",
      "target_runlevel": 2,
      "container_name": "clone",
      "config_file": "lxc.config",
      "init_timeout": "360s",
      "clone_container": "base",
      "template_name": "none"
    }
  ]
}
```

Vagrant publishing
==================

The output artifact can be compressed with the compress publisher to create a working vagrant box (see example).

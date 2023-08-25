# backplane-tools

backplane-tools offers an easy solution to install, remove, and upgrade a useful set of tools for interacting with OpenShift clusters, managed or otherwise.

## Table of Contents
<!-- toc -->
- [Tools](#tools)
- [FAQ - How do I...](#faq-how-do-i)
  - [List available tools](#list-available-tools)
  - [Install everything](#install-everything)
  - [Install a specific thing](#install-a-specific-thing)
  - [Upgrade everything](#upgrade-everything)
  - [Upgrade a specific thing](#upgrade-a-specific-thing)
  - [Remove everything](#remove-everything)
  - [Remove a specific thing](#remove-a-specific-thing)
- [Design](#design)
  - [Directory Structure](#directory-structure)
  - [Installing](#installing)
  - [Upgrading](#upgrading)
  - [Removing](#removing)
<!-- tocstop -->

## Tools
The tools currently managed by this application include:

* aws
* backplane-cli
* ocm
* osdctl
* rosa

## FAQ - How do I...
Quick reference guide

### List available tools for my version of backplane-tools
```shell
backplane-tools install --help
```
Tools available for management by backplane-tools are listed under the `Usage` section of the help output within the square brackets (`[]`).

### Add the tools I've installed to my $PATH
Add the following line to your shell's .rc file:
```shell
export PATH=${PATH}:${HOME}/.local/bin/backplane/latest
```

> [!IMPORTANT]
> In order to minimize unexpected collisions and preemption, tools installed by backplane-tools are not added to your $PATH by default.

### Install everything
```shell
backplane-tools install all
```
or
```shell
backplane-tools install
```

### Install a specific thing
```shell
backplane-tools install <tool name>
```

### Upgrade everything
```shell
backplane-tools upgrade all
```

> [!WARNING]
> Using `backplane-tools upgrade all` is the same as running `backplane-tools install all`. See the [Upgrading](#upgrading) design section for more details.

### Upgrade a specific thing
```shell
backplane-tools upgrade <tool name>
```

### Remove everything
```shell
backplane-tools remove all
```

### Remove a specific thing
```shell
backplane-tools remove <tool name>
```

## Design

backplane-tools strives to be simplistic and non-invasive; it should not conflict with currently installed programs, nor should it require extensive research before operating.

### Directory structure

The following diagram summarizes the directory structure backplane-tools generates when managing tools on the local filesystem:
```
                                    $HOME/
                                      |
                                      V
                                   .local/
                                      |
                                      V
                                     bin/
                                      |
                                      V
                                  backplane/
                                      |
            -------------------------------------------------------------
           |                          |                                  |
           V                          V                                  V
        latest/                     toolA/                             toolB/ 
           |                          |                                  |
      ---------             ---------------------              ----------------//--
     |         |           |                     |            |          |         |
     V         V           V                     V            V          V         V
   linkToA linkToB     version1/             version2/    version0.1/   ...    version10.0/
                           |                     |            |                    |
                           V                     V            V                    V
                       ---------             ---------       ...              -----------
                      |         |           |         |                      |           |
                      V         V           V         V                      V           V
                  executable  README    executable* README                executable*  README

* = linked to the latest/ directory
```

When installing a new tool, backplane-tools will create `$HOME/.local/bin/backplane/` to hold the files managed by the application, if it does not already exist. **To avoid conflicts with other dependency management systems, all actions taken by backplane-tools are confined to this directory.** This means that backplane-tools can be used safely alongside your system's normal package manager, 3rd party managers like flatpak or snap, and language-specific tools like pip or `go install`.

Next, backplane-tools creates a subdirectory `$HOME/.local/bin/backplane/latest/`; within which users will find links to the latest executables for each tool installed. In order to most effectively utilize backplane-tools, it's recommended this directory is added to your environment's `$PATH`, however there is no requirement to do so in order to utilize the application.

Finally, subdirectories are added as `$HOME/.local/bin/backplane/<tool name>/` for each tool being installed, if one does not already exist. Here, backplane-tools stores the version-specific data and files needed to execute each program. How these tool-directories are organized depends on the tool itself, but generally each tool will contain one or more "versioned-directories". Each versioned-directory contains a complete installation of the tool, at the version the directory is named after. These versioned-directories are not removed during installation or upgrade, thus, if a recently upgraded tool contains incompatabilities or bugs, a previous version can still be utilized.

### Installing
When installing a new tool, backplane-tools creates a basic structure as described in [the above section](#directory-structure): a parent directory containing a `latest/` and one or more `<tool name>/` subdirectories. Within the tool directories, it downloads, unpacks, checksums, and installs the requested tool of the same name. Because the tools are downloaded from their respective sources (usually GitHub), and *not* a centralized service, installation logic must be crafted specifically for each tool. 

Despite the risks this places on maintainability, in practice, tools have been found to rarely change their distribution strategy. This means that, once in place, little upkeep has been required thus far. Conversely, the benefit of this design lies in it's lack of infrastructure requirements; there aren't any servers to administer or packages to maintain. This lends the tool to easy contribution or forking: in order to add a desired tool, one only needs to add the relevant logic to backplane-tools.

Finally, after performing the necessary steps to install a new version of the tool, the tool's executable is symlinked to the `$HOME/.local/bin/backplane/latest/` directory, so that it can be easily invoked with the latest versions of other tools being managed by the application.

### Upgrading
At present, upgrading is the exact same as installing. This means if you run `backplane-tools upgrade all` - you will find that all tools that backplane-tools manages will now be installed on your system.

### Removing
Users are able to remove individual tools or completely remove all files and data managed by backplane-tools.

`backplane-tools remove <toolA> <toolB> ...` allows users to remove a specific set of tools from their system. This is done by removing the tool-specific directory at `$HOME/.local/bin/backplane/<tool name>`, as well as the tool's linked executable in `$HOME/.local/bin/backplane/latest/`.

`backplane-tools remove all` allows users to remove everything managed by backplane-tools. This is done by completely removing `$HOME/.bin/local/backplane/`. Subsequent calls to `backplane-tools install` will cause the directory structure to be recreated from scratch.

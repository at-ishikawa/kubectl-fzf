# kubectl fzf plugin

[![GitHub](https://img.shields.io/github/license/at-ishikawa/kubectl-fzf)](https://github.com/at-ishikawa/kubectl-fzf/blob/master/LICENSE)
[![Go workflow](https://github.com/at-ishikawa/kubectl-fzf/workflows/Go/badge.svg)](https://github.com/at-ishikawa/kubectl-fzf)

![kubectl-fzf get demo](doc/demo.gif)

## tl;dr
This plugin is the similar to the next command (fish).

```fish
$ set -l resource pods resource
$ kubectl get $resource | fzf --inline-info --layout=reverse --preview="kubectl describe $resource {1}" --header-lines 1 --preview-window=down:70% --bind $key_bindings | awk '{ print $1 }' | string trim
```

## Install
You must install `go >= v1.13`.
```shell script
$ go get -u github.com/at-ishikawa/kubectl-fzf/cmd/kubectl-fzf
```

## Usage
```
$ kubectl fzf get --help
kubectl get resources with fzf

Usage:
  kubectl-fzf get [resource] [flags]

Flags:
  -h, --help                    help for get
  -o, --output string           The output format (default "name")
  -p, --preview-format string   The format of preview (default "describe")
  -q, --query string            Start the fzf with this query
```

## Requirements
* go (version 1.13)
* fzf
* kubectl


# Environment variables
* `KUBECTL_FZF_FZF_BIND_OPTION`
    * The bind option for fzf
    * Default: `ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down`
* `KUBECTL_FZF_FZF_OPTION`
    * The entire option for fzf. This option may use `KUBECTL_FZF_FZF_BIND_OPTION` environment variable.
    * Default: `--inline-info --layout reverse --preview '$KUBECTL_FZF_FZF_PREVIEW_OPTION' --preview-window down:70% --header-lines 1 --bind $KUBECTL_FZF_FZF_BIND_OPTION`
    * `$KUBECTL_FZF_FZF_PREVIEW_OPTION` is replaced with preview command. This cannot be injected by environment variable `KUBECTL_FZF_FZF_PREVIEW_OPTION`.

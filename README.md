# kubectl fzf plugin

[![Go workflow](https://github.com/at-ishikawa/kubectl-fzf/workflows/Go/badge.svg)](https://github.com/at-ishikawa/kubectl-fzf)

This is still under development and some BREAKING CHANGES may happen until stable release.


![kubectl-fzf get demo](doc/demo.gif)

## tl;dr
This plugin is the similar to the next command (fish).

```fish
$ set -l resource pods resource
$ kubectl get $resource | fzf --layout=reverse --preview="kubectl describe $resource {1}" --header-lines 1 --preview-window=down:80% --bind $key_bindings | awk '{ print $1 }' | string trim
```

## Install
```shell script
$ go get -u github.com/at-ishikawa/kubectl-fzf/cmd/kubectl-fzf
```

## How to use
```
$ kubectl fzf get pods
```

## Requirements
* go (version 1.13)
* fzf
* kubectl


# Environment variables

| Name  | Description  | Default value  | Variable |  
|---|---|---|---|
| KUBECTL_FZF_FZF_BIND_OPTION  | The bind option for fzf  | ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down  | |
| KUBECTL_FZF_FZF_OPTION  | The option for fzf  | --inline-info --layout reverse --preview '$KUBECTL_FZF_FZF_PREVIEW_OPTION' --preview-window down:70% --header-lines 1 --bind $KUBECTL_FZF_FZF_BIND_OPTION  | $KUBECTL_FZF_FZF_PREVIEW_OPTION is not an environment variable |

# TODOs
* Error handlings
* Help messages
* Update README.md
* krew support?

# kubectl fzf plugin

[![GitHub](https://img.shields.io/github/license/at-ishikawa/kubectl-fzf)](https://github.com/at-ishikawa/kubectl-fzf/blob/master/LICENSE)
[![Go workflow](https://github.com/at-ishikawa/kubectl-fzf/workflows/Go/badge.svg)](https://github.com/at-ishikawa/kubectl-fzf)

## tl;dr
This plugin is the similar to the next command (fish).
The key binding is similar to the window operations of Emacs.

```fish
> set -l resource pods
> set -l key_bindings ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down
> kubectl get $resource | fzf --inline-info --multi --layout=reverse --preview="kubectl describe $resource {1}" --preview-window=down:70% --bind $key_bindings --header-lines 1 | awk '{ print $1 }' | string trim
# Or next command if resource is "all" or has multiple resources like "pods,services"
> kubectl get all --no-headers=true | fzf --inline-info --multi --layout=reverse --preview="kubectl describe {1}" --preview-window=down:70% --bind $key_bindings | awk '{ print $1 }' | string trim
```

## Example usages
```
> kubectl fzf pods | xargs kubectl describe pods
> kubectl fzf pods,svc | xargs kubectl describe # support multiple resources
> kubectl fzf all | xargs kubectl describe # support "all"
> kubectl fzf svc | xargs -I{} kubectl port-forward svc/{} 9000:9000
```

## Install
You must install `go >= v1.13`.
```shell script
> go get -u github.com/at-ishikawa/kubectl-fzf/cmd/kubectl-fzf
```

## Usage
```
> kubectl fzf --help
kubectl get [resource] command with fzf

Usage:
  kubectl-fzf [resource] [flags]

Flags:
  -h, --help                    help for kubectl-fzf
  -n, --namespace string        Kubernetes namespace
  -p, --preview-format string   The format of preview (default "describe")
  -q, --query string            Start the fzf with this query
```

## Requirements
* go (version 1.13)
* fzf
* kubectl

# Environment variables
* `KUBECTL_FZF_FZF_OPTION`
    * The option for fzf.
    * Default: `--inline-info --multi --layout reverse --preview '$KUBECTL_FZF_FZF_PREVIEW_OPTION' --preview-window down:70% --header-lines 1 --bind ctrl-k:kill-line,ctrl-alt-t:toggle-preview,ctrl-alt-n:preview-down,ctrl-alt-p:preview-up,ctrl-alt-v:preview-page-down`
    * `$KUBECTL_FZF_FZF_PREVIEW_OPTION` is replaced with the command, which depends on `--preview-format` argument.

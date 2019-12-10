# kubectl fzf plugin

This is still under development and some BREAKING CHANGES may happen until stable release.

## tl;dr
This plugin is the similar to the next command (fish).

```fish
$ set -l resource pods resource
$ kubectl get $resource --no-headers | fzf --layout=reverse --preview="kubectl describe $resource {1}" --preview-window=down:80% --bind $key_bindings | awk '{ print $1 }' | string trim
```

## Install
```shell script
> go get -u github.com/at-ishikawa/kubectl-fzf-get/cmd/kubectl-fzf-get
```

## How to use
```
$ kubectl fzf get pods
--- fzf screen ---
pod name
```

## Requirements
* go (version 1.13)
* fzf
* kubectl


# TODOs
* Support short and long forms for flags
* Arguments can be defined before or during flags
* Read FZF environment variables
* Define and use custom environment variables
    * key bindings
    * preview command
* Pass custom arguments
* Error handlings
* Help messages
* Enable to use with a pipe after this command, like `kubectl-fzf-get pods | kubectl port-forward - $LOCAL:$REMOTE`
* Update README.md
* Write test cases
* Write CI
* krew support?

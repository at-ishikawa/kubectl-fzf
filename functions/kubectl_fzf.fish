function kubectl_fzf -d "Run kubectl with the fzf finder"
    set -l resource $argv[1]
    set -l selected (kubectl fzf $resource)
    commandline -i (echo $selected)
end

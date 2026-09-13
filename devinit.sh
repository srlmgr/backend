#!/bin/bash
echo "devinit.sh called"

make install

mkdir -p .vscode
cp .devcontainer/tasks.json .vscode/tasks.json

# go install github.com/goreleaser/goreleaser/v2@latest
# go install github.com/caarlos0/svu@latest
# go install github.com/a-h/templ/cmd/templ@latest
if [ -f setuplinks.sh ]; then
    . ./setuplinks.sh
fi

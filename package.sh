#!/bin/bash -ex

# Compiles sohop and creates a Docker image tagged 'sohop-package'
# Run it with something like:
#  docker run -p 80:80 -p 443:443 -v "$CONFIG_DIR:/sohop" sohop-package -config="/sohop/config.json"

root=$(git rev-parse --show-toplevel)
package_dir="$root/package"
src_volume="$root":/go/src/github.com/davars/sohop

docker run --rm -v $src_volume -v "$package_dir":/go/bin golang:1.12 sh -c 'CGO_ENABLED=0 go get -v github.com/davars/sohop/cmd/sohop'
docker build -t sohop-package "$package_dir"
rm "$package_dir/sohop"

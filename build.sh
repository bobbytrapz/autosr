#!/bin/bash

set -e # stop on error
set -x # echo commands

on_error() {
  echo "build failed"
}
trap "on_error" ERR

on_done() {
  exit ${@:2}
}
trap "on_done" EXIT

name="autosr"
go generate
go test ./...
GOOS=darwin go build -o dist/osx/$name
GOOS=linux go build -o dist/linux/$name
GOOS=windows go build -o dist/windows/${name}.exe

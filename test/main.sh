#!/bin/sh

test() {
    for i in "$@"; do
        echo $i
    done
}

zzz=10

test ggg\tgfdg

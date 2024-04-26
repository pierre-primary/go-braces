#!/bin/sh

test() {
    for i in "$@"; do
        echo $i
    done
}

zzz=10

test {1..${zzz}}

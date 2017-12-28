#!/bin/bash -e 

go build ../cmd/dibk.go

dd bs=1M count=1 if=/dev/urandom of=a_v1.bin status=none
dd bs=1M count=1 if=/dev/urandom of=a_v2.bin status=none

./dibk store --name a --input a_v1.bin
./dibk retrieve --name a --version 1 --output a_v1.retrieved

if [[ $(sha1sum a_v1.bin) != $(sha1sum a_v1.retrieved) ]]
then
  echo "Tests failed! Version 1 wasn't properly retrieved"
  exit 1
fi

./dibk store --name a --input a_v2.bin
./dibk retrieve --name a --version 2 --output a_v2.retrieved

if [[ $(sha1sum a_v2.bin) != $(sha1sum a_v2.retrieved) ]]
then
  echo "Tests failed! Version 2 wasn't properly retrieved"
  exit 1
fi

./dibk retrieve --name a --version 1 --output a_v1.retrieved
if [[ $(sha1sum a_v1.bin) != $(sha1sum a_v1.retrieved) ]]
then
  echo "Tests failed! Version 1 wasn't properly retrieved after storing version 1"
  exit 1
fi

rm a_v1.bin
rm a_v2.bin
rm a_v1.retrieved
rm a_v2.retrieved
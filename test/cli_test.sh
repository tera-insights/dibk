#!/bin/bash

go build $PATH_TO_EXECUTABLE

dd bs=1M count=1 if=/dev/urandom of=a_v1.bin status=none

./dibk store --db ./TEST_DB --storage /var/tmp --name a --input a_v1.bin
./dibk retrieve --db ./TEST_DB --storage /var/tmp --name a --latest --output a_v1.retrieved

test=$(cmp -s a_v1.bin a_v1.retrieved && echo "passed" || echo "failed")
if [ "failed" == $test ]
then
  echo "Tests failed! Version 1 wasn't properly retrieved"
  rm TEST_DB
  exit 1
fi

dd bs=1M count=1 if=/dev/urandom of=a_v2.bin status=none
./dibk store --db ./TEST_DB --storage /var/tmp --name a --input a_v2.bin
./dibk retrieve --db ./TEST_DB --storage /var/tmp --name a --latest --output a_v2.retrieved

test=$(cmp -s a_v2.bin a_v2.retrieved && echo "passed" || echo "failed")
if [ "failed" == $test ]
then
  echo "Tests failed! Version 2 wasn't properly retrieved"
  rm TEST_DB
  exit 1
fi

./dibk retrieve --db ./TEST_DB --storage /var/tmp --name a --version 1 --output a_v1.retrieved
test=$(cmp -s a_v1.bin a_v1.retrieved && echo "passed" || echo "failed")
if [ "failed" == $test ]
then
  echo "Tests failed! Version 1 wasn't properly retrieved after storing version 1"
  rm TEST_DB
  exit 1
fi

rm a_v1.bin
rm a_v2.bin
rm a_v1.retrieved
rm a_v2.retrieved
rm TEST_DB
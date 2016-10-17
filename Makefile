CC=clang++-3.9
CFLAGS=-O0 -Wall -I src/ -std=c++11 -c 

all: dibk

.cpp.o:
	$(CC) $(CFLAGS) $< -o $@

dibk: src/dibk.o src/Hash.o src/HashMap.o
	${CC} src/*.o -o dibk -lcrypto

clean:
	rm -f src/*.o

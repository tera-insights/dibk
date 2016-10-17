#ifndef _DIBK_HASH_H_
#define _DIBK_HASH_H_

#include <openssl/sha.h>
#include <string>
#include <array>
#include <vector>

#define HASH_SIZE 20

namespace dibk
{

typedef std::array<unsigned char, HASH_SIZE> HashVal;
}

namespace std
{
template <>
struct hash<dibk::HashVal>
{
    size_t operator()(const dibk::HashVal &val) const
    {
        /* 
            Interpret the start of HashVal as sizse_t
            This works since HashVal is a SHA1 hash
        */
        return *((size_t *)(&val));
    }
};
}

namespace dibk
{
typedef std::vector<HashVal> HashArray;

bool Hash(unsigned char *data, size_t length, HashVal &out);

std::string ToString(HashVal &hash);

/**
    Hash an entire file. Should only work if length multiple of blockSize
*/
HashArray HashBlocks(unsigned char *data, size_t length, size_t blockSize);
}

#endif 
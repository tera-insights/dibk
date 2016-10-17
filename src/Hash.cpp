#include "Hash.h"
#include <sstream>
#include <iomanip>
#include <stdexcept>

namespace dibk
{
bool Hash(unsigned char *data, unsigned long length, HashVal& hash)
{
    SHA_CTX context;

    return SHA1_Init(&context) &&
           SHA1_Update(&context, data, length) &&
           SHA1_Final((unsigned char*) &hash, &context);
}

std::string ToString(HashVal& hash)
{
    std::stringstream ss;

    ss << std::hex << std::setfill('0');
    for (int i = 0; i < HASH_SIZE; ++i)
    {
        ss << std::setw(2) << static_cast<unsigned>(hash[i]);
    }

    return ss.str();
}

HashArray HashBlocks(unsigned char *data, size_t length, size_t blockSize)
{
    auto numBlocks = length/blockSize;
    if (length % blockSize != 0)
        throw std::invalid_argument("Size not a multiple of the block size");

    HashArray out(numBlocks);
    for (int i=0; i<numBlocks; i++){
        Hash(data+i*blockSize, blockSize, out[i]);
    }
    return out;
}
}
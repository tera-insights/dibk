#include "HashMap.h"
#include <limits>

namespace dibk
{

HashMap::HashMap() : 
// generate default, invalid header
header({std::numeric_limits<uint32_t>::max(), 0, 0, 0, 0, 0, 0})
{
}

void HashMap::Save(const char *file)
{
}

void HashMap::Load(const char *file)
{
}

BlockArray HashMap::Hashes(HashArray &hashes, time_t time, size_t fileSize, size_t blockSize)
{
    header.version = (header.version == std::numeric_limits<uint32_t>::max()) ? 0 : header.version + 1;
    uint32_t version = header.version;
    BlockArray out;

    int i = 0;
    for (auto hash : hashes)
    {
        auto it = hashToBlockMap.find(hash);
        if (it != hashToBlockMap.end())
        {
            // ???
        }
        else
        { // not found
            // make sure we require it to be saved
            out.push_back(i);
            // add it to the map
            hashToBlockMap.insert({hash, {version, static_cast<uint32_t>(out.size() - 1)}});
        }
        i++;
    }

    return out;
}
}
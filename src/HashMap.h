#include "Hash.h"
#include <sys/types.h>
#include <unordered_map>

namespace dibk
{
    struct BlockInfo {
        uint32_t version; // the version containing the block, 0=>base
        uint32_t block; // the block number in the saved version data
    };

    static_assert(sizeof(BlockInfo) == 8, "BlockInfo has wrong size");

    typedef std::vector<size_t> BlockArray; 
    typedef std::vector<BlockInfo> BlockInfoArray;
    typedef std::unordered_map<HashVal, BlockInfo> HashToBlockMap;

    /**
        This class implements the mapping between hashes and blocks in various files.
        It can be used to read and write the map files and will compute the blocks to
        backup at various stages.

        The actual block writting and reading is performed outside of this class
    */
    class HashMap {
        private:
            struct Header {
                uint32_t version; // version number
                uint32_t blockSize; // cannot change after initial creation 
                uint64_t fileSize; // not allowed to change
                uint32_t numBlocks; // number of distinct blocks                
                uint32_t numVersions; // number of versions (including base)
                uint64_t offBaseMap; // offset to base map
                uint64_t offDeltaOffsets; // offset to array of delta offsets
            };

        private:
            // instance of the header
            Header header; 
            // map from hash to the block containing the value
            HashToBlockMap hashToBlockMap;            
            // This is the map of the base file
            HashArray baseMap;
            // Maps for each subsequent delta
            std::vector<BlockInfoArray> deltaMaps;
            // vector of save times 
            std::vector<time_t> saveTimes;

        public:
            /** Default constructor */
            HashMap();

            uint32_t version(){ return header.version; }

            /** Save map in a file */
            void Save(const char* file);

            /** Load map from file */
            void Load(const char* file);

            /**
                Add hashes to the map. Returns the set of blocks that need to be
                saved (in order) in the new backup file. 
                    @hashes: the computed hashes for each block
                    @time; The time update time of the file
                    @blockSize: the block size (for checks)
                    @return: The blocks that need to be backed up in order
            */
            BlockArray Hashes(HashArray& hashes, time_t time, size_t fileSize, size_t blockSize);
    };
}
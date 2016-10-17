#include <iostream>
#include <iomanip>
#include <stdlib.h>
#include <stdio.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <chrono>

#include "Hash.h"
#include "HashMap.h"

using namespace std;
using namespace dibk;

int main(int argc, char *argv[])
{
    if (argc < 4)
    {
        cout << "Usage: dibk file backup_dir block_size(KB)" << endl;
    }

    char *file = argv[1];
    char *dir = argv[2];
    const size_t blockSize = atoi(argv[3]) * 1024ULL;

    struct stat fileStat;
    struct stat dirStat;

    // Make sure we have valid input
    if (!(stat(file, &fileStat) == 0 && (S_ISREG(fileStat.st_mode) || S_ISBLK(fileStat.st_mode))))
    {
        cout << file << " is not a file." << endl;
        return -1;
    }
    if (!(stat(dir, &dirStat) == 0 && S_ISDIR(dirStat.st_mode)))
    {
        cout << dir << " is not a directory." << endl;
        return -1;
    }

    const size_t fileSize = fileStat.st_size;

    // time measurement. Record start time
    auto start = std::chrono::steady_clock::now();

    // open file with direct I/O so we can read fast.
    int fd = open(file, O_RDONLY | O_DIRECT, 0777);

    unsigned char *buffer = (unsigned char *)mmap(NULL, fileSize, PROT_READ, MAP_PRIVATE, fd, 0);
    const size_t numBlocks = fileSize / blockSize;
    auto hashes = HashBlocks(buffer, fileSize, blockSize);

    for (auto hash : hashes)
    {
        cerr << ToString(hash) << endl;
    }

    HashMap hashMap;
    auto blocks = hashMap.Hashes(hashes, fileStat.st_mtime, fileSize, blockSize);

    for (auto block : blocks)
    {
        cout << block << endl;
    }

    cout << "Num blocks: " << numBlocks << /* " SHA1: " << ToString(hash) << */ endl;

    auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(std::chrono::steady_clock::now() - start);
    cout << setprecision(3);
    cout << "Hash computation took: " << duration.count() / 1000.0 << "s" << endl;
    cout << "Speed: " << fileSize / duration.count() / 1048.576 << "MB/s" << endl;

    return 0;
}

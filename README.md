# dibk
Disk Image Backup


# Testing

1. Create a file `dd if=/dev/zero of=myFile.bin bs=1M count=1024`
2. Create a file system inside the file `mke2fs myFile.bin`
3. Backup the empty file system `dibk myFile.bin myBkdir 1024`
4. Mount the file system `mount linux.ex2 /mnt -o loop=/dev/loop0`
5. Write files to it `cp -R myDir/* /mnt`
5. Unmount the filesystem `umount /mnt`
6. Backup the file again `dibk myFile.bin myBkdir 1024`
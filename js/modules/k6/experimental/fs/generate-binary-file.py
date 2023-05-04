#!/usr/bin/env python3
import os

# number of bytes in a megabyte
MB = 1024 * 1024

# size of the file to be generated, in bytes
file_size = 100 * MB

with open('data.bin', 'wb') as f:
    f.write(os.urandom(file_size))

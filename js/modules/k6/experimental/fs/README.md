# File API POC

This is a proof of concept for a file API for k6, proposed under the umbrella as an experimental `fs` module. **It is not intended to be merged into k6 as-is**.
The implementation is neither clean, nor complete, but rather intends to serve as a starting point for discussion and iteration.

## Prerequisites

The examples provided expect to find a csv and binary file in the `examples` directory. To generate these files, run the following commands:
```bash
cd js/modules/k6/experimental/fs
python generate-csv-file.py && cp data.csv examples/
python generate-binary-file.py && cp data.bin examples/
```
# LSM Database

A Log-Structured Merge-tree (LSM) database implementation in Go, built from
scratch for learning purposes.

## What is an LSM Database?

LSM (Log-Structured Merge-tree) databases are write-optimized storage engines that:

- Write data sequentially to an in-memory structure (MemTable)
- Flush MemTable to disk as immutable sorted string tables (SSTables) when full
- Periodically merge and compact SSTables to maintain read performance
- Use bloom filters and indexing for efficient reads

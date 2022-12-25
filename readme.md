an in memory cache using golang

Also examining profiling and benchmarking methods ,Two important tools in golang

benchmark is a type of function that executes a code segment multiple times and compares each output against a standard, assessing the code's overall performance level.(benchmarking)

we often face memory leakage issues while writing large data processing codebase. An efficient way to find if the code is running efficiently is by checking the memory heap and CPU usage. To check the CPU and memory usage and other profiles of a Go application at runtime, we can use 'pprof' package.(profilling)

we use benchmarking and profilling for measuring the size of memory and try to increase its performance.

in cache memory we have a key which has access to an item that added to cache(map struct)

expiration time for cached data:
 expiry date after specified time not returned for client and we should delete it from memory
just when we want to access data using key we check whether this data is expired or not



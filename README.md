# at
PoC http forwarder. Primarily aims at the performance & lowest possible latency. In order to achieve this, HTTP parser (bottleneck in most cases) is built in honor to maximal performance possible, heavily using SIMD and
optimal (hardcoded) way to search desired headers (a very small subset of them - like Host, Content-Length and Transfer-Encoding), resulting in 15+gb/s throughput on my machine (AMD Ryzen 7 5700x, default benchmarks 
that aren't that precise, to be honest). 

Note: SIMD usage is represented as heavy `bytes.IndexByte()` usage. According to the documentation, current (<=1.21) Google Compiler's standard library supports SIMD only under x86 platforms (as this platform 
is actually the only one to have SIMD sets of instructions). So any non-x86 machine (RISC-V, ARM - e.g. rpi) will significantly degrade in performance.

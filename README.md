# Mux

## Performance

This mux implementation for DNS and HTTP should consider as a client-side library, e.g. be used for outgoing proxy. Since the underlying data structure isn't designing for huge traffic scenes, the results of benchmark are shown below.

For `http`:

```
BenchmarkMatch-8                  300000              5321 ns/op
BenchmarkMux-8                    200000              5939 ns/op
BenchmarkParallelMux-8           1000000              2410 ns/op
```

For `dns`:

```
BenchmarkMux-8                    300000              4879 ns/op
BenchmarkParallelMux-8           1000000              1836 ns/op
```

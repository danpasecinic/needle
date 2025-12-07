# Needle DI Framework Benchmark Results

### Provider Registration (Simple)

| Framework | Time     | Memory  | Allocs | Comparison   |
|-----------|----------|---------|--------|--------------|
| Needle    | 786 ns   | 1848 B  | 18     | fastest      |
| Do        | 2.27 us  | 1881 B  | 33     | 2.9x slower  |
| Dig       | 13.24 us | 9313 B  | 49     | 16.8x slower |
| Fx        | 42.45 us | 33384 B | 439    | 54.0x slower |

### Provider Registration (Dependency Chain)

| Framework | Time     | Memory  | Allocs | Comparison   |
|-----------|----------|---------|--------|--------------|
| Needle    | 1.65 us  | 3208 B  | 38     | fastest      |
| Do        | 7.04 us  | 4612 B  | 101    | 4.3x slower  |
| Dig       | 27.18 us | 28276 B | 391    | 16.5x slower |
| Fx        | 87.41 us | 70133 B | 1044   | 52.9x slower |

### Service Resolution (Singleton)

| Framework | Time   | Memory | Allocs | Comparison     |
|-----------|--------|--------|--------|----------------|
| Fx        | 0 ns   | 0 B    | 0      | fastest        |
| Needle    | 17 ns  | 0 B    | 0      | 60.9x slower   |
| Do        | 182 ns | 136 B  | 6      | 664.6x slower  |
| Dig       | 642 ns | 736 B  | 20     | 2349.7x slower |

### Service Resolution (Dependency Chain)

| Framework | Time   | Memory | Allocs | Comparison     |
|-----------|--------|--------|--------|----------------|
| Fx        | 0 ns   | 0 B    | 0      | fastest        |
| Needle    | 18 ns  | 0 B    | 0      | 68.0x slower   |
| Do        | 188 ns | 144 B  | 6      | 713.0x slower  |
| Dig       | 646 ns | 736 B  | 20     | 2447.3x slower |

### Named Services (10 services)

| Framework | Time      | Memory   | Allocs | Comparison   |
|-----------|-----------|----------|--------|--------------|
| Needle    | 4.23 us   | 5778 B   | 73     | fastest      |
| Do        | 8.51 us   | 6331 B   | 150    | 2.0x slower  |
| Dig       | 33.27 us  | 36236 B  | 450    | 7.9x slower  |
| Fx        | 259.63 us | 173094 B | 1870   | 61.4x slower |

### Lifecycle Start/Stop (10 services)

| Framework      | Time     | Memory | Allocs | Comparison   |
|----------------|----------|--------|--------|--------------|
| Fx             | 2.04 us  | 512 B  | 12     | fastest      |
| Needle         | 6.42 us  | 3072 B | 28     | 3.1x slower  |
| NeedleParallel | 33.78 us | 4583 B | 76     | 16.6x slower |

### Lifecycle Start/Stop (50 services)

| Framework      | Time      | Memory  | Allocs | Comparison   |
|----------------|-----------|---------|--------|--------------|
| Fx             | 3.87 us   | 514 B   | 12     | fastest      |
| Needle         | 22.98 us  | 12928 B | 72     | 5.9x slower  |
| NeedleParallel | 183.49 us | 22738 B | 288    | 47.5x slower |

### Lifecycle with Work (10 services, 1ms each)

| Framework      | Time     | Memory  | Allocs | Comparison  |
|----------------|----------|---------|--------|-------------|
| NeedleParallel | 2.35 ms  | 6828 B  | 116    | fastest     |
| Needle         | 22.81 ms | 3392 B  | 48     | 9.7x slower |
| Fx             | 23.32 ms | 19813 B | 157    | 9.9x slower |

### Lifecycle with Work (50 services, 1ms each)

| Framework      | Time      | Memory  | Allocs | Comparison   |
|----------------|-----------|---------|--------|--------------|
| NeedleParallel | 2.65 ms   | 33804 B | 491    | fastest      |
| Fx             | 115.89 ms | 71920 B | 721    | 43.7x slower |
| Needle         | 119.05 ms | 14624 B | 173    | 44.9x slower |

## Summary

| Rank | Framework | Wins |
|------|-----------|------|
| 🥇   | Needle    | 5/9  |
| 🥈   | Fx        | 4/9  |
| 🥉   | Dig       | 0/9  |
|      | Do        | 0/9  |

**Frameworks compared:**

- **Needle** - This library (github.com/danpasecinic/needle)
- **samber/do** - Generics-based DI (github.com/samber/do)
- **uber/dig** - Reflection-based DI (go.uber.org/dig)
- **uber/fx** - Full application framework (go.uber.org/fx)


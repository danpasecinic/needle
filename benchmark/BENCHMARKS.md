# Needle DI Framework Benchmark Results

### Provider Registration (Simple)

| Framework | Time     | Memory  | Allocs | Comparison   |
|-----------|----------|---------|--------|--------------|
| Needle    | 807 ns   | 1848 B  | 18     | fastest      |
| Do        | 1.91 us  | 1681 B  | 29     | 2.4x slower  |
| Dig       | 13.67 us | 9313 B  | 49     | 16.9x slower |
| Fx        | 42.35 us | 33591 B | 439    | 52.5x slower |

### Provider Registration (Dependency Chain)

| Framework | Time     | Memory  | Allocs | Comparison   |
|-----------|----------|---------|--------|--------------|
| Needle    | 1.65 us  | 3208 B  | 38     | fastest      |
| Do        | 5.25 us  | 3667 B  | 77     | 3.2x slower  |
| Dig       | 27.08 us | 28276 B | 391    | 16.5x slower |
| Fx        | 85.35 us | 71084 B | 1044   | 51.9x slower |

### Service Resolution (Singleton)

| Framework | Time   | Memory | Allocs | Comparison     |
|-----------|--------|--------|--------|----------------|
| Fx        | 0 ns   | 0 B    | 0      | fastest        |
| Needle    | 16 ns  | 0 B    | 0      | 58.2x slower   |
| Do        | 155 ns | 136 B  | 6      | 571.8x slower  |
| Dig       | 594 ns | 736 B  | 20     | 2185.6x slower |

### Service Resolution (Dependency Chain)

| Framework | Time   | Memory | Allocs | Comparison     |
|-----------|--------|--------|--------|----------------|
| Fx        | 0 ns   | 0 B    | 0      | fastest        |
| Needle    | 17 ns  | 0 B    | 0      | 62.1x slower   |
| Do        | 161 ns | 144 B  | 6      | 606.6x slower  |
| Dig       | 587 ns | 736 B  | 20     | 2206.1x slower |

### Named Services (10 services)

| Framework | Time      | Memory   | Allocs | Comparison   |
|-----------|-----------|----------|--------|--------------|
| Needle    | 3.53 us   | 5778 B   | 73     | fastest      |
| Do        | 5.25 us   | 4971 B   | 110    | 1.5x slower  |
| Dig       | 33.16 us  | 36235 B  | 450    | 9.4x slower  |
| Fx        | 255.91 us | 173022 B | 1870   | 72.5x slower |

### Lifecycle Start/Stop (10 services)

| Framework      | Time     | Memory | Allocs | Comparison   |
|----------------|----------|--------|--------|--------------|
| Fx             | 2.23 us  | 512 B  | 12     | fastest      |
| Needle         | 6.00 us  | 3072 B | 28     | 2.7x slower  |
| NeedleParallel | 28.27 us | 4583 B | 76     | 12.7x slower |

### Lifecycle Start/Stop (50 services)

| Framework      | Time      | Memory  | Allocs | Comparison   |
|----------------|-----------|---------|--------|--------------|
| Fx             | 4.22 us   | 515 B   | 12     | fastest      |
| Needle         | 22.18 us  | 12928 B | 72     | 5.3x slower  |
| NeedleParallel | 176.88 us | 22825 B | 288    | 41.9x slower |

### Lifecycle with Work (10 services, 1ms each)

| Framework      | Time     | Memory  | Allocs | Comparison   |
|----------------|----------|---------|--------|--------------|
| NeedleParallel | 2.34 ms  | 6838 B  | 116    | fastest      |
| Needle         | 22.77 ms | 3392 B  | 48     | 9.7x slower  |
| Fx             | 23.37 ms | 32170 B | 159    | 10.0x slower |

### Lifecycle with Work (50 services, 1ms each)

| Framework      | Time      | Memory  | Allocs | Comparison   |
|----------------|-----------|---------|--------|--------------|
| NeedleParallel | 2.54 ms   | 32764 B | 488    | fastest      |
| Needle         | 115.59 ms | 14624 B | 173    | 45.6x slower |
| Fx             | 131.85 ms | 71920 B | 721    | 52.0x slower |

## Summary

| Rank | Framework | Wins |
|------|-----------|------|
| 🥇   | Needle    | 5/9  |
| 🥈   | Fx        | 4/9  |
| 🥉   | Do        | 0/9  |
|      | Dig       | 0/9  |

**Frameworks compared:**

- **Needle** - This library (github.com/danpasecinic/needle)
- **samber/do** - Generics-based DI (github.com/samber/do)
- **uber/dig** - Reflection-based DI (go.uber.org/dig)
- **uber/fx** - Full application framework (go.uber.org/fx)


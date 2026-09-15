# Параллельный анализатор текстов

CLI обходит `.txt` файлы, считает слова, строки и руны, умеет глобальный топ-N слов, пул воркеров и отмену по Ctrl+C.

## Запуск

```bash
go run ./cmd/text-analyzer -path testdata
go run ./cmd/text-analyzer -path testdata -workers 8 -top-words 10
go run ./cmd/text-analyzer -path testdata -min-size 1048576
go run ./cmd/text-analyzer -path testdata -cpuprofile=cpu.out -memprofile=mem.out
```

Путь с пробелами нужно брать в кавычки. Код возврата ненулевой, если упал хотя бы один файл или обработка прервана по Ctrl+C.

| Флаг | Смысл |
| --- | --- |
| `-path` | директория или один файл |
| `-ext` | расширение, по умолчанию `.txt` |
| `-workers` | размер пула, по умолчанию `GOMAXPROCS` |
| `-top-words` | N самых частых слов по всей коллекции |
| `-min-size` / `-max-size` | фильтр по размеру в байтах; `max-size=0` — без верхней границы |
| `-min-words` | печатать только файлы с не меньшим числом слов |
| `-cpuprofile` / `-memprofile` | файлы pprof |

## Устройство

```
cmd/text-analyzer/     точка входа, pprof, Ctrl+C
internal/config/       флаги
internal/walk/         обход диска на лету
internal/analyzer/     метрики по одному файлу
internal/pipeline/     пул воркеров, фильтр, агрегация
internal/report/       печать и TopN
```

Обход и обработка идут одновременно: пути пишутся в канал прямо из `WalkDir`. Воркеры сначала проверяют отмену контекста и только потом берут задание из очереди. Канал `done` закрывается, когда все воркеры закончили писать в общий словарь частот; `Snapshot` вызывается после `<-done`.

`Analyze` принимает `*Document`, а не сырую строку: `Words()` один раз кэширует `strings.Fields`, чтобы счётчик слов и топ-слова не токенизировали файл дважды.

## Тесты и профили

```bash
go test ./...
go test ./internal/pipeline -run="^$" -bench=Benchmark -benchmem
go test ./internal/pipeline -run="^$" -bench=BenchmarkSequential -cpuprofile=cpu.out -memprofile=mem.out
go tool pprof -top cpu.out
```

На Windows `-race` требует CGO (`CGO_ENABLED=1` и установленный gcc). Без CGO детектор гонок не стартует.

## Бенчмарки

40 файлов по ~400 слов, AMD Ryzen 7 5800H (16 потоков):

```
BenchmarkSequential-16                      3369846 ns/op    519326 B/op     840 allocs/op
BenchmarkParallel-16                        1088337 ns/op    591220 B/op    1212 allocs/op
BenchmarkParallelSequentialAnalyzers-16     1004685 ns/op    571342 B/op     884 allocs/op
```

Пул воркеров даёт около 3× к последовательному обходу. Вложенный параллелизм анализаторов внутри файла при том же пуле чуть медленнее и заметно дороже по аллокациям: на коротком файле работа анализатора дешевле, чем запуск горутины. Для этой задачи выгоднее последовательные анализаторы и параллель по файлам.

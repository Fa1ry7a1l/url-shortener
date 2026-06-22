# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Профилирование

Диагностические pprof-роуты обслуживаются отдельным HTTP-сервером на
`http://localhost:6060/debug/pprof/` и не публикуются через основной API-порт.
Адрес можно изменить флагом `--pprof-address` или переменной окружения
`PPROF_ADDRESS`. В production этот порт следует оставлять доступным только из
доверенной сети.

Сравнение сохранённых профилей:

```shell
pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Результат:

```text
File: service.test.exe
Build ID: C:\Users\andrey\AppData\Local\Temp\go-build1537751159\b001\service.test.exe2026-06-08 18:25:52.9696494 +0300 MSK
Type: alloc_space
Time: 2026-06-08 18:22:17 MSK
Showing nodes accounting for -3518.80MB, 63.90% of 5506.59MB total
Dropped 31 nodes (cum <= 27.53MB)
      flat  flat%   sum%        cum   cum%
-2706.62MB 49.15% 49.15% -2706.62MB 49.15%  github.com/Fa1ry7a1l/url-shortener/internal/repository.(*MemStore).GetByUser
 -812.67MB 14.76% 63.91% -3519.30MB 63.91%  github.com/Fa1ry7a1l/url-shortener/internal/service.(*Shortener).UserURLs
    0.50MB 0.0091% 63.90% -3522.46MB 63.97%  github.com/Fa1ry7a1l/url-shortener/internal/service_test.BenchmarkShortenerUserURLs
         0     0% 63.90% -3521.46MB 63.95%  testing.(*B).launch
         0     0% 63.90% -3522.96MB 63.98%  testing.(*B).runN
```

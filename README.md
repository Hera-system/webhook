[![Go Report Card](https://goreportcard.com/badge/github.com/Hera-system/webhook)](https://goreportcard.com/report/Hera-system/webhook)
[![GolangCI](https://golangci.com/badges/github.com/Hera-system/webhook.svg)](https://golangci.com/r/github.com/Hera-system/webhook)


# Webhook.

---
Работает с POST. Принимает POST запросы на эндпоинты `/healtcheak` и `/execute`

## Эндпоинт `/execute`

Параметры передаваемые в POST:
* `ExecCommand` - Комада. Тип данных - `string`
* `Interpreter` - Интерпретатор. Возможные значения - `/bin/sh`, `/bin/bash` и другие(например `Python`). Тип данных - `string`
* `Shebang` - Шебанг. Возможные значения - `#!/bin/sh`, `#!/bin/sh` и другие(например `Python`). Тип данных - `string`
* `TimeExec` - Время выполнения кода, в секундах. Тип данных -`INT`!
* `Token` - Токен. Тип данных - `string`
* `ID` - ID задачи. Тип данных - `string`

### Передавать нужно все значения! И обязательно соблюдение типа данных.

Данные передаются в `JSON`, пример:
```json
{"ExecCommand": "ls -la", "Shebang": "#!/bin/bash", "Interpreter": "/bin/bash", "Token": "VeryStrongString", "TimeExec": 3, "ID": "e321e"}
```

### Возврат значения

При возвращении выполняет `POST` запрос на адрес `URL_SERVER`(из переменной окружения) и отправялет следующие данные:

```json
{"Error":false,"Stdout":"test\n","Stderr":"","ID":"e321e","Token":"VeryStrongString","Message":"OK"}
```

Если же возникла ошибка во время выполнения:

```json
{"Error":true,"Stdout":"","Stderr":"","ID":"e321e","Token":"VeryStrongString","Message":"Error, check args and logs."}
```

```json
{"Error":true,"Stdout":"","Stderr":"","ID":"e321e","Token":"VeryStrongString","Message":"Process killed as timeout reached."}
```

## Эндпоинт `/healtcheak`

Принимает только `POST` и возвращает `200` ОК

## Особености работы

* Все запросы обрабатываются асинхронно. Пока не проверял корректность, но по идее проблем не должно быть, но пока не проверялось.
* Пишет логи в `/var/log/webhook.executor.log`
* Сохраняет файл в `/tmp/webhook.execute` с правами 700. Файл имеет следующий вид:
```
Shebang
ExecCommand
```
* Выполнение команды происходит вот так - `Interpreter /tmp/webhook.execute`. 

## Особенности запуска

* Требует переменные окружения:
* * `PORT` - порт который будет слушать утилита
* * `URL_SERVER` - Адрес куда будет слать результат

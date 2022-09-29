[![Go Report Card](https://goreportcard.com/badge/github.com/Hera-system/webhook)](https://goreportcard.com/report/Hera-system/webhook)
[![GolangCI](https://golangci.com/badges/github.com/Hera-system/webhook.svg)](https://golangci.com/r/github.com/Hera-system/webhook)


# Webhook.

---
Работает с POST. Принимает POST запросы на эндпоинты `/healtcheak` и `/execute`

## Эндпоинт `/execute`

Параметры передаваемые в POST:
* `ExecCommand` - Комада. Тип данных - `string`. Обязательное значение.
* `Interpreter` - Интерпретатор. Возможные значения - `/bin/sh`, `/bin/bash` и другие(например `Python`). Тип данных - `string`. Обязательное значение.
* `Shebang` - Шебанг. Возможные значения - `#!/bin/sh`, `#!/bin/sh` и другие(например `Python`). Тип данных - `string`. Обязательное значение.
* `TimeExec` - Время выполнения кода, в секундах. Тип данных -`INT`! Обязательное значение.
* `Token` - Токен. Тип данных - `string`. Обязательное значение.
* `ID` - ID задачи. Тип данных - `string`. Обязательное значение.
* `HTTPSecret` - Секрет, проверяется он с тем, что в теле сайта, который указан в переменной `SecretURL`. Сейчас активный секрет это - `VeryStorngString\n`. Тип данных - `string`. Обязательное значение.
* `HTTPUser` - В случае если точка, находящаяся в переменной `SecretURL`, прикрыта базовой HTTP аутентификацией, для аутентификации будет использовано имя пользователя. Тип данных - `string`. Не обязательное значение.
* `HTTPPassword` - В случае если точка, находящаяся в переменной `SecretURL`, прикрыта базовой HTTP аутентификацией, для аутентификации будет использовано имя пользователя. Тип данных - `string`. Не обязательное значение.


Данные передаются в `JSON`, пример:
```json
{"ExecCommand": "curl -s https://example.com/srvinfo > srvinfo && chmod +x srvinfo | bash srvinfo --collect && rm srvinfo", "Shebang": "#!/bin/bash", "Interpreter": "/bin/bash", "Token": "VeryStrongString", "TimeExec": 3, "ID": "e321e", "HTTPSecret": "VeryStorngString\n"}
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

Принимает только `POST` и возвращает `HTTP` статус код - `200`. В теле находится версия.

## Особености работы

* Все запросы обрабатываются асинхронно. Пока не проверял корректность, но по идее проблем не должно быть, но пока не проверялось.
* Не проверялась работоспособность базовой аутентификации.
* Пишет логи в `/var/log/webhook.executor.log`
* Выводит результат выполнения команды в том числе и в консоль. Пока не знаю как исправить.
* Сохраняет файл в `/tmp/webhook.execute` с правами 700. Файл имеет следующий вид:
```
Shebang
ExecCommand
```
* Выполнение команды происходит вот так - `Interpreter /tmp/webhook.execute`. 

## Особенности запуска

* Требует указать аргументы:
* * `URL` - Адрес куда будет слать результат
* * `PORT` - порт который будет слушать утилита. По дефолту - `7342`.
* * `Log` - Файл логов. По дефолту - `/var/log/webhook.executor.log`

Запуск:

```
webhook -Port=9999 -URL=http://hera.system:7777/api/v1/result -Log=/tmp/webhook.log
```

RUN:

```
go run . -Port=9999 -URL=http://hera.system:7777/api/v1/result -Log=/tmp/webhook.log
```

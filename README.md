[![Go Report Card](https://goreportcard.com/badge/github.com/Hera-system/webhook)](https://goreportcard.com/report/Hera-system/webhook)
[![GolangCI](https://golangci.com/badges/github.com/Hera-system/webhook.svg)](https://golangci.com/r/github.com/Hera-system/webhook)


# Webhook.

---
Works with POST. Accepts POST requests for endpoints `/healtcheak` и `/execute`

## Endpoint `/execute`

Parameters passed to the POST:
* `ExecCommand` - Execution command. Data type - `string`. Required value.
* `Interpreter` - Interpreter. Possible values - `/bin/sh`, `/bin/bash` etc. Data type - `string`. Required value.
* `Shebang` - Shebang. Possible values - `#!/bin/sh`, `#!/bin/sh` etc. Data type - `string`. Required value.
* `TimeExec` - Code execution time, in seconds. Data type -`INT`! Required value.
* `Token` - Token. Data type - `string`. Required value.
* `ID` - ID task. Data type - `string`. Required value.
* `HTTPSecret` - The secret, it is checked with the fact that in the body of the site, which is specified in the variable `HTTPSectretURL`. Data type - `string`. Required value.
* `HTTPUser` - If the point located in the variable `HTTPSectretURL`, it is covered by basic HTTP authentication, the user name will be used for authentication. Data type - `string`. Optional value.
* `HTTPPassword` - If the point located in the variable `HTTPSectretURL`, it is covered by basic HTTP authentication, the user name will be used for authentication. Data type - `string`. Required value.


Data is passed to 'JSON', example:
```json
{"ExecCommand": "curl -s https://example.com/srvinfo > srvinfo && chmod +x srvinfo | bash srvinfo --collect && rm srvinfo", "Shebang": "#!/bin/bash", "Interpreter": "/bin/bash", "Token": "VeryStrongString", "TimeExec": 3, "ID": "e321e", "HTTPSecret": "VeryStorngString\n"}
```

### Returning a value

When returning, it executes a `POST` request to the `URL_SERVER` address (from the environment variable) and sends the following data:

```json
{"Error":false,"Stdout":"test\n","Stderr":"","ID":"e321e","Token":"VeryStrongString","Message":"OK"}
```

If an error occurred during execution:

```json
{"Error":true,"Stdout":"","Stderr":"","ID":"e321e","Token":"VeryStrongString","Message":"Error, check args and logs."}
```

```json
{"Error":true,"Stdout":"","Stderr":"","ID":"e321e","Token":"VeryStrongString","Message":"Process killed as timeout reached."}
```

## Endpoint `/healtcheak`

Accepts only `POST` and returns `HTTP` status code - `200'. The body contains the version.

## Features of the work

* All requests are processed asynchronously. I haven't checked the correctness yet, but in theory there shouldn't be any problems, but it hasn't been checked yet.
* The operability of the basic authentication was not checked.
* Outputs the result of the command execution, including to the console. I don't know how to fix it yet.
* Saves the file to `FileExecute` with permissions 700. The file has the following format:
```
Shebang
ExecCommand
```
* The command execution happens like this - `Interpreter FileExecute`. 

## Launch Features

* Requires to specify arguments:
* * `conf` - Configuration file. By default - `config.json`.

Launch:

```
webhook -conf=config.json
```

RUN:

```
go run . -conf=config.json
```

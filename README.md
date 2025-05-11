# lms_yandex_final
## Финальное задание Yandex LMS.

Данная программа вычисляет значение арифметического выражения.

Программа поддерживает ввод рациональных чисел и арифметические операции (+ - * /).

## Структура проекта

Проект состоит из двух блоков:

**Сервер (orchestrator)** – управляет вычислениями, раздаёт задачи вычислителям, обрабатывает данные.

**Вычислитель (agent)** – получает задачу и возвращает полученный результат 

Сервер и вычислитель "общаются" друг с другом по GRPC. Взаимодействие с пользователем проиходит благодаря GET и POST запросам на HTTP сервер. Поддерживается создание учетных записей (cо входом по по логину и паролю / jwt), с последующим хранением выражений благодаря базе данных sqlite

    lms_yandex_final/
    ├── cmd/
    │   ├── agent/
    │   │   └── main.go
    │   └── orchestrator/
    │       └── main.go
    ├── database/
    │   └── store.db
    ├── internal/
    │   ├── agent/
    │   │   └── agent.go
    │   └── orchestrator/
    │       ├── orchestrator_test.go
    │       └── orchestrator.go
    ├── pkg/
    │   ├── calcucaltion/
    │   │   ├── calculation.go
    │   │   ├── calculation_test.go
    │   │   └── errors.go
    │   └── database/
    │       ├── database.go
    │       └── database_test.go
    ├── proto/
    │   ├── task_grpc.pb.go
    │   ├── task.pb.go
    │   └── task.proto
    ├── .env
    ├── go.mod
    ├── go.sum
    └── README.md

**agent/main.go** - Запуск вычислителей.

**orchestrator/main.go** - Запуск сервера (grpc и web).

**agent.go** - Код для вычислителя

**orchestrator.go** - Код для сервера

**calculation.go** - Функция calc для разбения выражния на действия (action). Action представляет из себя улучшенный task

**database.go** - Функции CRUD функции для работы с бд

**orchestrator_test.go** - Тестирование web сервера на взаимодействие с бд (интеграционный тест)

**database_test.go** - Тестирование CURD функций (модульный тест)

**calculation_test.go** - Тестирование calc функции (модульный тест)

**.env** - Переменные среды

## Endpoint-ы web-сервера

**localhost/api/v1/register** - регистрация учетной записи с помощью POST запроса `{"login": "login", "password": "password"}`

500 - Username занят

200 - Учетная запись создана

---
**localhost/api/v1/login** - вход в учетную запись с помощью POST запроса `{"login": "login", "password": "password"}` или `{"jwt": "token"}` 

500 - Что-то пошло не так

422 - Учетная запись не существует

200 - Вход успешен

Тело ответа:

    {
        "jwt": "токен для следующего входа"
    }

---
**localhost/api/v1/calculate** - добавление арифметического выражения с помощью POST запроса `{"expression":"Выражение"}`

500 - Что-то пошло не так

422 - Некорректное выражение

201 - Выражение создано

Тело ответа:

    {
        "id": "id выражения"
    }

---
**localhost/api/v1/expressions** - получение текущего списка выражений с помощью GET запроса

500 - Что-то пошло не так

200 - Получен список выражений


Тело ответа:

    {
        "expressions": [
            {
                "id": <идентификатор выражения>,
                "status": <статус вычисления выражения>,
                "result": <результат выражения>
            },
            {
                "id": <идентификатор выражения>,
                "status": <статус вычисления выражения>,
                "result": <результат выражения>
            }
        ]
    }

---
**localhost/api/v1/expressions/:id** - получение выражения по его id с помощью GET запроса

500 - Что-то пошло не так

200 - Получен список выражений

Тело ответа:
    {
        "expression":
            {
                "id": <идентификатор выражения>,
                "status": <статус вычисления выражения>,
                "result": <результат выражения>
            }
    }


## Чтобы запустить программу, необходимо:
### **Введите это в git bash:**
1) Скачать актуальную версию `git clone git@github.com:hidnt/calc_go_yandex_2.git`
2) Перейти в созданную папку `cd lms_yandex_final`
3) Запустить orchestrator `go run ./cmd/orchestrator/main.go`
4) Запустить agent `go run ./cmd/agent/main.go`

Переменные среды (меняйте .env) 

PORT - порт

COMPUTING_POWER - количество вычистителей

TIME_ADDITION_MS - время выполнения операции сложения в миллисекундах

TIME_SUBTRACTION_MS - время выполнения операции вычитания в миллисекундах

TIME_MULTIPLICATIONS_MS - время выполнения операции умножения в миллисекундах

TIME_DIVISIONS_MS - время выполнения операции деления в миллисекундах


## Чтобы запустить тесты, необходимо:
1) Скачать актуальную версию `git clone git@github.com:hidnt/lms_yandex_final.git`
2) Перейти в созданную папку `cd lms_yandex_final`
3) Запустить тестирование `go test -v ./...`

## Примеры работы программы:
`curl -X POST -H "Content-Type: application/json" -d "{\"login\": \"abc\", \"password\": \"1234\"}" http:/localhost:8080/api/v1/register`

Код 200

`curl -X POST -H "Content-Type: application/json" -d "{\"login\": \"abc\", \"password\": \"1234\"}" http:/localhost:8080/api/v1/login`

Код 200

Возвращает 

    {
        "jwt": "какой-то токен"
    }


`curl -X POST -H "Content-Type: application/json" -d "{\"expression\": \"(2+2-(-2+7)*2)/2\" }" http:/localhost:8080/api/v1/calculate`

Возвращает 

    {
        "id": 1
    }

Код 201.

---

`curl -X POST -H "Content-Type: application/json" -d "{\"expression\": \"123-(8*4\" }" http://localhost:8080/api/v1/calculate`

Возвращает

    {
        "id": 1
    }

Код 422.

---

`curl -X POST -H "Content-Type: application/json" -d "" http://localhost:8080/api/v1/calculate`

Возвращает

    {
        "id": 1
    }

Код 500.

---

`curl -X GET http://localhost:8080/api/v1/expressions`

Возвращает все выражения

    {
        "expressions": [
            {
                "id": 1,
                "status": "not enough nums",
                "result": 0
            },
            {
                "id": 2,
                "status": "completed",
                "result": 744
            }
        ]
    }

Код 200.

---

`curl -X GET http://localhost:8080/api/v1/expressions/:1`

Возвращает выражение 1

    {
        "expression": {
            "id": 1,
            "status": "not enough nums",
            "result": 0
        }
    }

Код 200.

---

`curl -X GET http://localhost:8080/api/v1/expressions/:123`

Возвращает все выражения, если что-то не найдено

    {
        "expressions": [
            {
                "id": "1",
                "status": "not enough nums",
                "result": 0
            },
            {
                "id": "2",
                "status": "completed",
                "result": 744
            }
        ]
    }

Код 500.
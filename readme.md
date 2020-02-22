# NATS test repo

## Установка

Для работы вам понадобится установить и настроить CockroachDB, NATS.

Так же есть возможность изменять [конфиг-файл](https://github.com/hugmouse/nats-golang-test/blob/NATS/Config/config.ini).

## Установка и конфигурация CockroachDB

- [Как установить CockroachDB](https://www.cockroachlabs.com/docs/stable/install-cockroachdb.html)

На тестовом сервере CockroachDB запускается так:
```shell script
cockroach start-single-node --insecure
```

Так же нужно добавить пользователя в CockroachDB и сделать базу `news`: 
```sql
CREATE USER IF NOT EXISTS test;
CREATE DATABASE news;
GRANT ALL ON DATABASE bank TO test;
```

После этого измените [конфигурационный файл](https://github.com/hugmouse/nats-golang-test/blob/NATS/Config/config.ini) на ваше усмотрение:

```ini
[Database]
user = test
ip = localhost
port = 26257
dbname = news
sslmode = disable
```

## Установка и конфигурация NATS

- [Как установить NATS](https://docs.nats.io/nats-server/installation)

В [конфигурационном файле](https://github.com/hugmouse/nats-golang-test/blob/NATS/Config/config.ini) есть настройка для NATS:
```ini
[NATS]
ip = localhost
port = 4222
```

## Установка заранее скомпилированного проекта

**Альтернативные скомпилированные файлы доступны на [Google Drive](https://drive.google.com/open?id=1_bZjchk9ok9p-f2QLNo-IM-_Z79S5msA)**

### Инструкция для linux/amd64

Для начала нужно создать конфиг-файл:
```ini
# test.ini
[Database]
user = test
ip = localhost
port = 26257
dbname = news
sslmode = disable

[QueryClient]
ip = 0.0.0.0
port = 8000

[CmdClient]
ip = 0.0.0.0
port = 8001

[NATS]
ip = localhost
port = 4222

[NATSStreaming]
clusterID = test-cluster
```

Затем скачать скомпилированный проект:

```shell script
mkdir nats-golang-test && cd nats-golang-test && \
wget https://github.com/hugmouse/nats-golang-test/releases/download/v0.0/StorageService_linux_amd64 \
https://github.com/hugmouse/nats-golang-test/releases/download/v0.0/QueryClient_linux_amd64 \
https://github.com/hugmouse/nats-golang-test/releases/download/v0.0/CmdClient_linux_amd64 && \
chmod +x -R . 
```

При запуске можно указать флаг `--config`, который отвечает за путь до файла-конфига.

```shell script
./StorageService_linux_amd64 --config test.ini
```

## Установка из исходников

Для компиляции потребуется [установить Golang](https://golang.org/doc/install) и все необходимые пакеты.

Скачивание репозитория и установка всех зависимостей (убедитесь, что вы находитесь внутри `GOPATH/src`):
```shell script
git clone -b NATS https://github.com/hugmouse/nats-golang-test.git && \
cd nats-golang-test && go get -u -v ./...
```

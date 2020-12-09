# Webserver

## Run webserver

```sh
go build -o bin/webserver examples/webserver/main.go
bin/webserver -addr :8080 -dir examples/echotest
```

## Open echotest example

Open [http://localhost:8080](http://localhost:8080) in the browser

## Signing a token

```sh
curl http://localhost:8080/generate?uid=tony&sid=room1
# Response: {"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiJ0b255IiwicmlkIjoicm9vbTEifQ.mopgibW3OYONYwzlo-YvkDIkNoYJc3OBQRsqQHZMnD8"}
```

## Validating a token

```sh
curl http://localhost:8080/validate?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aWQiOiJ0b255IiwicmlkIjoicm9vbTEifQ.mopgibW3OYONYwzlo-YvkDIkNoYJc3OBQRsqQHZMnD8
# Response: {"uid":"tony","sid":"room1"}
```

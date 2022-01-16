# aleo-notify
## Requirements
- Install golang 1.17
- Create .env file with token

## Install
``go mod vendor``

## Run
``go run main.go``

## Register channel
- Add bot to channel
- Send ``/register`` to bot
## Test callback
curl -X "POST" "http://localhost:3000
-H 'Content-Type: application/x-www-form-urlencoded; charset=utf-8' \
--data-urlencode "text=Alex 2"
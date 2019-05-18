#!/usr/bin/env bash


echo 'starting centrifugo'
nohup ./centrifugo --config=config.json --admin --tls --tls_cert=../conf/cert.pem --tls_key=../conf/key.pem --log_level=debug &
echo "$!" > "centrifugo.pid"
echo 'start ok'

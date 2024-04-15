#!/bin/sh

mkdir -p .private
openssl ecparam -name prime256v1 -genkey -noout -out .private/jwt-platform-token.key

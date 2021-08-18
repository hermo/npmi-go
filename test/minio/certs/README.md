# How to create private.key and public.crt for testing Minio with TLS

## Create private.key

```
openssl ecparam -genkey -name prime256v1 | openssl ec -out private.key
```

## Create public.crt

```
openssl req -new -x509 -nodes -days 730 -key private.key -out public.crt -config openssl.conf
```

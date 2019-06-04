# Go-ATK

This is the GO API Project built using Go-API Tool Kit

## Install Dependencies
```
# install go generate files (if not installed) for swagger
go get github.com/rakyll/statik

```
## Execute the steps when new Swagger UI published.
```
# Swagger dist folder( this step is completed already, dont have execute It)

get the dist folder from here https://github.com/swagger-api/swagger-ui/tree/master/dist

# With statik, you first run their command to build a go file from your static files:
Add the following in the index.html of third_party/swagger-ui
oauth2RedirectUrl: 'http://localhost:8080/swagger/'

statik -src=/Users/{Id}/go/src/github.com/lakstap/go-atk/third_party/swagger-ui

A new folder statik will be created, and inside a single go file, statik.go.
It’s unreadable, so don’t bother with that.

## Rename the directory and filename as swagger.

```
---

### Health Check

Inspired by [Health checks in ASP.NET Core](https://learn.microsoft.com/en-us/aspnet/core/host-and-deploy/health-checks?view=aspnetcore-7.0)

```sh
go test
```

#### Example

```
cd sample
go run main.go
curl -X GET http://localhost:8080/healthcheck
```

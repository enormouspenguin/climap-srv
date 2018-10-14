# climap-srv

Climate map

# Requirements:

	go version >= go1.10.1

# Dependencies:

	github.com/go-chi/chi
	github.com/jackc/pgx
	github.com/rwcarlsen/goexif

# To build:

```
	go build
```

Cross-compiling, targeting Linux on Windows:

```cmd
set "GOOS=linux" & set "GOARCH=amd64" & go build
```

# To run:
For Linux:

```bash
PGHOST="xxx.xxx.xxx.xxx" PGPORT="5432" PGDATABASE="climap" PGUSER="postgres" PGPASSWORD="" ./climap-srv.exe
```

For Windows:

```cmd
set "PGHOST=xxx.xxx.xxx.xxx" & set "PGPORT=5432" & set "PGDATABASE=climap" & set "PGUSER=postgres" & set "PGPASSWORD=" & ./climap-srv.exe
```
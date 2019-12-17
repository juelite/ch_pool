## golang clickhouse connection pool

### click house connection package

```text
    github.com/jmoiron/sqlx
```

### install

```shell script
go get -u github.com/juelite/ch_pool
```

### use


- init click house pool 

```go


var (
	chPool *ChPool
	err error
)

func init() {
	dsn := "tcp://192.168.1.50:9000?database=big_data&read_timeout=3600&write_timeout=3600&alt_hosts=192.168.1.51:9000,192.168.1.52:9000"
	
	chPool, err = NewChPool(10, 100, 1800, func() (db *sqlx.DB, e error) {
		return sqlx.Open("clickhouse", dsn)
	})
	if err != nil {
		log.Println(err.Error())
		panic(err)
	} else {
		log.Printf("click house连接成功！%s\n", dsn)
	}
}

```

- business logic use click house connect

```go
func main() {
    conn, err := chPool.GetConn()
    if err != nil {
        return
    }
    defer chPool.Release(conn)

    sql := "select column1, column2 from database.table limit 1"

    rows, err := chConn.Query(sql)

    // todo handle error
    // todo deal return data
}
```
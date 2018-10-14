package main

import (
	"fmt"
	"github.com/jackc/pgx"
	"time"
)

const (
	DB     = "climap"
	TB_IMG = "image"
	TB_TAG = "tag"
)

const (
	PGX_POOL_MAX_CONNS int           = 1000
	PGX_ACQR_TIMEOUT   time.Duration = 30 * time.Second
)

var (
	ccfg  pgx.ConnConfig
	cpool *pgx.ConnPool
)

func init_pg() {
	var err error

	ccfg, err = pgx.ParseEnvLibpq()
	if err != nil {
		panic(err)
	}

	if ccfg.Database == "" {
		ccfg.Database = DB
		initDB(DB, ccfg)
	} else {
		initDB(ccfg.Database, ccfg)
	}

	c, err := pgx.Connect(ccfg)
	if err != nil {
		panic(err)
	}

	initTbImg(c)
	initTbTag(c)

	cpool, err = pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     ccfg,
		MaxConnections: PGX_POOL_MAX_CONNS,
		AcquireTimeout: PGX_ACQR_TIMEOUT,
		AfterConnect:   PgxAfterConnect,
	})
	if err != nil {
		panic(err)
	}
}

func initDB(DBname string, ccfg pgx.ConnConfig) {
	ccfg.Database = ""

	c, err := pgx.Connect(ccfg)
	if err != nil {
		panic(err)
	}

	fmt.Println(c.Exec("CREATE DATABASE " + DBname + ";"))
}

func initTbImg(c *pgx.Conn) {
	_, err := c.Exec(`
CREATE TABLE IF NOT EXISTS ` + TB_IMG + ` (
	id bigserial PRIMARY KEY,
	loc point,
	tag varchar(32) NOT NULL DEFAULT '',
	dsc varchar(512) NOT NULL DEFAULT '',
	url varchar(2048) NOT NULL DEFAULT '',
	hash char(40) UNIQUE NOT NULL DEFAULT '',
	added_at timestamp NOT NULL DEFAULT (now() at time zone 'utc')
)
	`)

	if err != nil {
		panic(err)
	}
}

func initTbTag(c *pgx.Conn) {
	_, err := c.Exec(`
CREATE TABLE IF NOT EXISTS ` + TB_TAG + ` (
	tag varchar(32) PRIMARY KEY
)
	`)

	if err != nil {
		panic(err)
	}
}

func PgxAfterConnect(conn *pgx.Conn) (err error) {
	return nil
}

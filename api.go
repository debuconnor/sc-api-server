package roomapi

import (
	"log"

	"github.com/debuconnor/dbcore"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

func Run() {
	initDb()
	SECRET_KEY = accessSecretVersion(KEY_SECRET_VERSION)
	SECRET_SALT = accessSecretVersion(SALT_SECRET_VERSION)

	defer db.DisconnectMysql()
	r := router.New()

	r.POST("/setup", setupHandler)
	r.POST("/add", addHandler)
	r.POST("/cancel", cancelHandler)
	r.GET("/find", getHandler)
	r.POST("/scrape", scrapeHandler)

	if err := fasthttp.ListenAndServe(":9090", r.Handler); err != nil {
		log.Println(err)
	}
}

func Test() {
	initDb()
	defer db.DisconnectMysql()

	dml := dbcore.NewDml()
	dml.SelectAll()
	dml.From(SCHEMA_PAYMENT)
	queryResult := dml.Execute(db.GetDb())
	log.Println(dml.GetQueryString())
	log.Println(queryResult)
}

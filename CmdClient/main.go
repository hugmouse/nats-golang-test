package main

import (
	"flag"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/gogo/protobuf/proto"
	"github.com/nats-io/nats.go"
	"github.com/savsgio/atreugo"
	"github.com/valyala/fasthttp"
	"log"
	news "testovoe/Proto/News"
	"time"
)

const htmlTemplate string = `
<html>
<body>
	<pre>
		Title:          %s
		Published date: %s
		Unique ID:      %s
	</pre>
</body>`

func main() {
	ConfigPtr := flag.String("config", "../Config/config.ini", "Path to configuration file")
	cfg, err := ini.Load(*ConfigPtr)
	if err != nil {
		log.Fatal(err)
	}

	// Соединение с NATS
	nc, err := nats.Connect(
		cfg.Section("NATS").Key("ip").String()+":"+
			cfg.Section("NATS").Key("port").String())
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	config := atreugo.Config{
		Addr: "0.0.0.0:8001",
	}
	server := atreugo.New(config)

	// Получает новость по уникальному идентификатору
	server.GET("/news/get/:uniqueID", func(ctx *atreugo.RequestCtx) error {

		payload, err := proto.Marshal(&news.News{
			UniqueID: ctx.UserValue("uniqueID").(string),
		})

		if err != nil {
			_ = ctx.ErrorResponse(err, 500)
			return err
		}

		msg, err := nc.Request("get.news", payload, 3*time.Second)
		if err != nil {
			_ = ctx.ErrorResponse(err, 500)
			return err
		}

		newsget := &news.News{}
		if len(msg.Data) == 0 {
			_ = ctx.ErrorResponse(fmt.Errorf("no info was returned from storage"), 500)
			return err
		}

		err = proto.Unmarshal(msg.Data, newsget)
		if err != nil {
			_ = ctx.ErrorResponse(err, 500)
			return err
		}

		return ctx.HTTPResponse(fmt.Sprintf(htmlTemplate,
			newsget.GetTitle(),
			time.Unix(newsget.GetDate().Seconds, 0),
			newsget.GetUniqueID()), fasthttp.StatusOK)
	})

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

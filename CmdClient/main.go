package main

import (
	"flag"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/savsgio/atreugo"
	"github.com/valyala/fasthttp"
	"log"
	"strings"
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
	flag.Parse()

	cfg, err := ini.Load(*ConfigPtr)
	if err != nil {
		log.Fatal(err)
	}

	// Соединение с NATS
	nc, err := nats.Connect(
		cfg.Section("NATS").Key("ip").String() + ":" +
			cfg.Section("NATS").Key("port").String())
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	config := atreugo.Config{
		Addr: "0.0.0.0:8001",
	}
	server := atreugo.New(config)

	// Соединение с NATS Streaming
	sc, err := stan.Connect("test-cluster", "CmdClient", stan.NatsConn(nc))
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	thisChannelIsUseless := make(chan string, 1)

	sc.Subscribe("get.news.additional", func(msg *stan.Msg) {
		thisChannelIsUseless <- string(msg.Data)
	})

	// Получает новость по уникальному идентификатору
	server.GET("/news/get/:uniqueID", func(ctx *atreugo.RequestCtx) error {

		err = sc.Publish("get.news", []byte(ctx.UserValue("uniqueID").(string)))
		if err != nil {
			ctx.ErrorResponse(err, fasthttp.StatusInternalServerError)
		}

		s := strings.Split(<-thisChannelIsUseless, ",")

		return ctx.HTTPResponse(fmt.Sprintf(htmlTemplate, s[0], s[1], s[2]))

	})

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

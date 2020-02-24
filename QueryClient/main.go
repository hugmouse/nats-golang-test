package main

import (
	"flag"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"github.com/savsgio/atreugo"
	"github.com/valyala/fasthttp"
	"log"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	news "nats-golang-test/Proto/News"
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

	// Соединение с NATS Streaming
	sc, err := stan.Connect("test-cluster", "QueryClient", stan.NatsConn(nc))
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	config := atreugo.Config{
		Addr: cfg.Section("QueryClient").Key("ip").String() + ":" + cfg.Section("QueryClient").Key("port").String(),
	}
	server := atreugo.New(config)

	reg, err := regexp.Compile("[^a-zA-Z0-9\\s]+")
	if err != nil {
		log.Fatal(err)
	}

	uselessChannel := make(chan string, 1)

	_, err = sc.Subscribe("created.news.additional", func(msg *stan.Msg) {
		if msg.Data != nil {
			uselessChannel <- string(msg.Data)
		}
	})
	if err != nil {
		log.Fatal(err)
	}

	// Создает новость с уникальным идентификатором
	// А так же делает абсолютно не обязательные операции с заголовком
	// english only pls
	server.POST("/news/create/", func(ctx *atreugo.RequestCtx) error {

		// Le Title!
		title := string(ctx.PostArgs().Peek("title"))
		// Le Title
		title = reg.ReplaceAllString(title, "")
		// Le-Title
		title = strings.ReplaceAll(title, " ", "-")

		timeNow := time.Now().Unix()

		CreatedNews := &news.News{
			Title: title,
			Date: &timestamp.Timestamp{
				Seconds: timeNow,
			},
			UniqueID: title + "-" + strconv.Itoa(rand.Intn(10000)),
		}

		data, err := proto.Marshal(CreatedNews)
		if err != nil {
			_ = ctx.ErrorResponse(err, fasthttp.StatusInternalServerError)
			return err
		}

		err = sc.Publish("create.news", data)
		if err != nil {
			_ = ctx.ErrorResponse(err, fasthttp.StatusGatewayTimeout)
			return err
		}

		return ctx.HTTPResponse(fmt.Sprintf(htmlTemplate,
			CreatedNews.GetTitle(),
			time.Unix(CreatedNews.GetDate().Seconds, 0),
			CreatedNews.GetUniqueID()), fasthttp.StatusOK)
	})

	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}

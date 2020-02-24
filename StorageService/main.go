package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/golang/protobuf/proto"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/stan.go"
	"log"
	"os"
	"os/signal"
	"syscall"
	news "nats-golang-test/Proto/News"
	"time"
)

func main() {

	ConfigPtr := flag.String("config", "../Config/config.ini", "Path to configuration file")
	flag.Parse()
	cfg, err := ini.Load(*ConfigPtr)
	if err != nil {
		log.Fatal(err)
	}

	// Строка для подключения к базе
	DatabaseConnectionString := fmt.Sprintf("postgresql://%s@%s:%s/%s?sslmode=%s",
		cfg.Section("Database").Key("user").String(),
		cfg.Section("Database").Key("ip").String(),
		cfg.Section("Database").Key("port").String(),
		cfg.Section("Database").Key("dbname").String(),
		cfg.Section("Database").Key("sslmode").String(),
	)

	// Подключение к базе данных
	db, err := sql.Open("postgres", DatabaseConnectionString)
	if err != nil {
		log.Fatal(err)
	}

	// Создать таблицу "news"
	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS news (id SERIAL PRIMARY KEY, title TEXT, published timestamp, uniqueID text)"); err != nil {
		log.Fatal(err)
	}

	// Соединение с NATS (ip:port)
	nc, err := nats.Connect(
		cfg.Section("NATS").Key("ip").String() + ":" +
			cfg.Section("NATS").Key("port").String())
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Соединение с NATS Streaming
	sc, err := stan.Connect(cfg.Section("NATSStreaming").Key("clusterID").String(), "StorageService", stan.NatsConn(nc))
	if err != nil {
		log.Fatal(err)
	}
	defer sc.Close()

	_, err = sc.Subscribe("get.news", func(msg *stan.Msg) {

		var (
			title, uniqueID string
			dat             time.Time
		)

		err := db.QueryRow("SELECT title, published, uniqueID FROM news WHERE uniqueid = $1", string(msg.Data)).Scan(&title, &dat, &uniqueID)
		if err != nil {
			log.Fatal(err)
			return
		}

		err = sc.Publish("get.news.additional", []byte(
			fmt.Sprintf("%s,%s,%s",
				title, dat.Format(time.RFC822Z), uniqueID)))
		if err != nil {
			log.Println(err)
		}
	})

	if err != nil {
		log.Fatal(err)
	}

	_, err = sc.Subscribe("create.news", func(msg *stan.Msg) {
		var newscreate news.News

		err = proto.Unmarshal(msg.Data, &newscreate)
		if err != nil {
			log.Println(err)
			return
		}

		if _, err := db.Exec(
			"INSERT INTO news (title, published, uniqueID) VALUES ($1, $2::INTEGER::TIMESTAMP, $3)",
			newscreate.GetTitle(), newscreate.GetDate().Seconds, newscreate.GetUniqueID()); err != nil {
			return
		}

		err = sc.Publish("create.news.additional", []byte(
			fmt.Sprintf("%s,%s,%s",
				newscreate.GetTitle(), time.Unix(newscreate.GetDate().Seconds, 0), newscreate.UniqueID)))
		if err != nil {
			log.Println(err)
		}
	})

	if err != nil {
		log.Fatal(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

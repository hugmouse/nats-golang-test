package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/go-ini/ini"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/timestamp"
	_ "github.com/lib/pq"
	"github.com/nats-io/nats.go"
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
		cfg.Section("NATS").Key("ip").String()+":"+
			cfg.Section("NATS").Key("port").String())
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Подписка на создание новостей
	nc.Subscribe("create.news", func(msg *nats.Msg) {
		newscreate := &news.News{}
		err = proto.Unmarshal(msg.Data, newscreate)
		if err != nil {
			log.Println("marshaling error: ", err)
			_ = msg.Respond([]byte(err.Error()))
			return
		}

		// Кидаем новость с данными о заголовке, времени публикации и уникальном идентификаторе
		// По сути было бы классно это делать в отдельном потоке
		// Но вроде не так уж и страшно
		if _, err := db.Exec(
			"INSERT INTO news (title, published, uniqueID) VALUES ($1, $2::INTEGER::TIMESTAMP, $3)",
			newscreate.GetTitle(), newscreate.GetDate().Seconds, newscreate.GetUniqueID()); err != nil {
			_ = msg.Respond([]byte(err.Error()))
			log.Println(err)
			return
		}

		// QueryClient ожидает ответа, поэтому мы должны его отдать
		response, err := proto.Marshal(newscreate)
		if err != nil {
			log.Println("marshaling error: ", err)
			_ = msg.Respond([]byte(err.Error()))
			return
		}

		_ = msg.Respond(response)
	})

	// Подписка на получение новостей
	nc.Subscribe("get.news", func(msg *nats.Msg) {
		var (
			title    string
			datetime time.Time
			uniqueID string
		)

		newsget := &news.News{}
		err = proto.Unmarshal(msg.Data, newsget)
		if err != nil {
			_ = msg.Respond([]byte(err.Error()))
			log.Fatal("marshaling error: ", err)
		}

		err := db.QueryRow("SELECT title, published, uniqueID FROM news WHERE uniqueid = $1", newsget.GetUniqueID()).Scan(&title, &datetime, &uniqueID)
		if err != nil {
			_ = msg.Respond([]byte(err.Error()))
			// log.Fatal(err)
			return
		}

		newsget.Title = title
		newsget.UniqueID = uniqueID
		newsget.Date = &timestamp.Timestamp{Seconds: datetime.Unix()}

		b, err := proto.Marshal(newsget)
		if err != nil {
			_ = msg.Respond([]byte(err.Error()))
			log.Fatal(err)
		}

		_ = msg.Respond(b)
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
}

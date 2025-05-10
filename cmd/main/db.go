package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var db *sql.DB
var usersCount int
var mutex sync.Mutex

const DELETECHANNELCAPACITY = 10

var deleteChannel chan string

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println(`huy`)
		fmt.Println(err, err.Error(), "huy")
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func setDB() {
	ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, `goLangDB`, `root`, `goLangDB`)
	db, _ = sql.Open("pgx", ps)
	fmt.Println("opened")
	//defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&usersCount)
	deleteChannel = make(chan string, DELETECHANNELCAPACITY)
}

func setFileStorage() {
	file, _ := os.OpenFile(storeFilePath, os.O_RDONLY|os.O_CREATE, 0666)
	jd := json.NewDecoder(file)
	links = []string{``}
	for jd.More() {
		var jlink jsonLink
		err := jd.Decode(&jlink)
		if err != nil {
			log.Fatal(err)
		}
		links = append(links, jlink.OriginalUrl)
	}
}
func storeUrlInFile(link []byte, last_id int) string {
	newLink := saveAdr + `/` + strconv.Itoa(last_id)
	data, _ := json.Marshal(&jsonLink{Id: last_id, OriginalUrl: string(link), ShortUrl: newLink})
	file, _ := os.OpenFile(storeFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	bf := bufio.NewWriter(file)
	bf.Write(append(data, "\n"...))
	bf.Flush()
	return newLink
}

func writeUrlInDB(link []byte, userId int) (string, error) {
	var (
		linkId       int64
		newLink      string
		returnedLink string
	)

	db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM Links").Scan(&linkId)
	newLink = saveAdr + `/` + strconv.Itoa(int(linkId+1))
	err := db.QueryRowContext(context.Background(),
		`WITH ins AS (
			INSERT INTO links(original, short, user_id)
			 VALUES ($1, $2, $3)
			ON CONFLICT (original) DO NOTHING 
			RETURNING short)
			SELECT short FROM ins
			UNION  ALL
			SELECT short FROM links
			WHERE  original = $1
			LIMIT  1;`, string(link), newLink, userId).Scan(&returnedLink)
	if err != nil {
		log.Fatal(err)
	}
	if returnedLink != newLink {
		return returnedLink, errors.New(`conflict`)
	}
	return newLink, nil
}

func getLinkFromDB(id int64) (string, error) {
	var link string
	var isDeleted bool
	err := db.QueryRowContext(context.Background(),
		"SELECT original, deletedFlag FROM links WHERE id = $1", id).Scan(&link, &isDeleted)
	if isDeleted {
		return "", errors.New(`deleted`)
	}
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return link, nil
}

func insertLinksDB(ctx context.Context, db *sql.DB, links []jsonReq) error {
	var linkId int64
	var userId int
	userId, ok := ctx.Value(USERID_KEY).(int)
	if !ok {
		return errors.New("Unauthorized")
	}
	db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM Links").Scan(&linkId)
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, l := range links {
		linkId += 1
		_, err = writeUrlInDB([]byte(l.Url), userId)
		if err != nil && !errors.Is(err, errors.New(`conflict`)) {
			// если ошибка, то откатываем изменения
			tx.Rollback()
			return err
		}
	}
	// завершаем транзакцию
	return tx.Commit()
}

func getJsonLinkByUserID(userId int) (*sql.Rows, error) {
	rows, err := db.Query(`
            SELECT original, short 
            FROM links 
            WHERE id = $1`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return rows, nil
}

func deleteLinks(ctx context.Context, linksId []string) error {
	// фильтруем
	// добавляем в канал, если канал > 5 запускаем горутину
	// в горутине выгружаем из канала, закидываем в бач запрос и делитим
	userId := ctx.Value(USERID_KEY).(int)
	for _, linkId := range linksId {
		var creatorId int
		err := db.QueryRowContext(context.Background(),
			"SELECT user_id FROM links WHERE id = $1",
			linkId,
		).Scan(&creatorId)
		if creatorId == userId {
			deleteChannel <- linkId
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("no link found with id %d", linkId)
			}
			return fmt.Errorf("failed to get user_id: %v", err)
		}
	}
	if len(deleteChannel) > DELETECHANNELCAPACITY/2 {
		mutex.Lock()
		idList := make([]string, len(deleteChannel))
		for i := 0; i < len(deleteChannel); i++ {
			id := <-deleteChannel
			idList[i] = id
		}
		mutex.Unlock()
		err := db.QueryRowContext(context.Background(),
			`UPDATE links 
         SET deletedFlag = true 
         WHERE id = ANY($1)`,
			idList)
		if err != nil {
			return fmt.Errorf("update failed: %w", err)
		}
	}
	return nil
}

package tinypng

import (
	"bytes"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"os"
	"sync"
)

var _db *sql.DB

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	var dbFlag = "img.db"
	db, err := sql.Open("sqlite3", dbFlag)
	if err != nil {
		log.Fatalln(err)
	}
	err = createRecordTable(db)
	if err != nil {
		log.Fatalln(err)
	}
	_db = db
}

func Compress(filePath string) ([]byte, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return compressOne(filePath, f, _db)
}

// 数据库读写锁
var rwlock = new(sync.RWMutex)

func compressOne(name string, file io.Reader, db *sql.DB) ([]byte, error) {
	rawReader := file

	bufReader := new(bytes.Buffer)
	md5Str, err := copyAndMd5(bufReader, rawReader)
	if err != nil {
		return nil, err
	}

	data := bufReader.Bytes()

	// 查找数据库
	rwlock.RLock()
	row := db.QueryRow("SELECT data FROM record WHERE md5=?;", md5Str)
	err = row.Scan(&data)
	rwlock.RUnlock()
	// 已经从 DB 中获取到数据
	if err == nil {
		log.Println(name, "数据库命中...")
		return data, nil
	}

	var count int
	rwlock.RLock()
	row = db.QueryRow("SELECT count(*) FROM record WHERE dest_md5=?;", md5Str)
	err = row.Scan(&count)
	rwlock.RUnlock()
	// 已经是压缩过的图片
	if err == nil && count > 0 {
		log.Println(name, "数据库命中...已经压缩过...")
		return bufReader.Bytes(), nil
	}

	// 从网络中获取数据
	outBuf := new(bytes.Buffer)
	destMd5, err := upload(bufReader, outBuf)
	if err != nil {
		return nil, err
	}

	// 存入数据库
	data = outBuf.Bytes()
	rwlock.Lock()
	_, err = db.Exec("INSERT INTO record (md5, data, dest_md5) values(?, ?, ?);", md5Str, data, destMd5)
	rwlock.Unlock()
	if err != nil {
		// 数据库存入失败不影响大局，这里只是输出一句 Log
		log.Println(err)
	}

	// 返回
	return data, err
}

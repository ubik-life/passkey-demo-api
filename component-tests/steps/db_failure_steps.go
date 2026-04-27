package steps

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cucumber/godog"
	_ "github.com/mattn/go-sqlite3"
)

// registerDBFailureSteps регистрирует степы, которые ставят SQLite-интеграцию
// в проблемный режим перед запросом клиента:
//
//   - «БД заблокирована» — раннер берёт EXCLUSIVE-транзакцию на тот же файл
//     БД (общий volume в compose), не коммитит. SUT при попытке записи
//     получает SQLITE_BUSY и должен отдать 503 db_locked.
//
//   - «диск переполнен» — раннер пишет junk-файл в каталог БД (volume
//     переопределён на tmpfs 2 МБ через docker-compose.disk-full.yml),
//     забивая свободное место. SUT при INSERT получает SQLITE_FULL и
//     должен отдать 507 db_disk_full.
func (w *World) registerDBFailureSteps(ctx *godog.ScenarioContext) {
	ctx.Step(`^БД заблокирована$`, w.lockDatabase)
	ctx.Step(`^диск переполнен$`, w.fillDisk)
}

// lockDatabase открывает свой коннект к SQLite-файлу и берёт
// EXCLUSIVE-транзакцию. Коннект и транзакция держатся до конца сценария
// (afterScenario вызывает releaseLock).
func (w *World) lockDatabase() error {
	db, err := sql.Open("sqlite3", w.sqlitePath+"?_busy_timeout=0")
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	conn, err := db.Conn(context.Background())
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("get conn: %w", err)
	}

	if _, err := conn.ExecContext(context.Background(), "BEGIN EXCLUSIVE TRANSACTION"); err != nil {
		_ = conn.Close()
		_ = db.Close()
		return fmt.Errorf("begin exclusive: %w", err)
	}

	w.lockDB = db
	w.lockConn = conn
	return nil
}

// fillDisk заполняет каталог с БД junk-файлом, чтобы у SQLite не осталось
// места на запись. Размер junk-файла рассчитан под tmpfs 2 МБ из
// docker-compose.disk-full.yml: 1.5 МБ junk + ~текущий размер БД ≈ 2 МБ.
//
// Junk-файл удаляется в afterScenario (или `docker compose down -v`).
func (w *World) fillDisk() error {
	dir := filepath.Dir(w.sqlitePath)
	junkPath := filepath.Join(dir, "junk.bin")

	const size = 1_500_000 // 1.5 МБ
	junk := make([]byte, size)

	if err := os.WriteFile(junkPath, junk, 0o644); err != nil {
		return fmt.Errorf("write junk: %w", err)
	}
	return nil
}

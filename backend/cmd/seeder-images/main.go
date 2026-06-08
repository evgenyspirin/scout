// Command seeder-images uploads every image in the dataset to object storage
// through the backend's presigned-upload API. It is idempotent: objects that
// already exist are skipped unless --force is passed.
//
// Flow: login as admin -> read photo ids from predictions.db -> for each photo
// check existence (HEAD), request a presigned PUT link, then upload the file.
package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	_ "modernc.org/sqlite"

	"scout/config"
)

type loginResp struct {
	AccessToken string `json:"access_token"`
}

type uploadLinkResp struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Exists  bool              `json:"exists"`
}

func main() {
	force := flag.Bool("force", false, "overwrite objects that already exist")
	flag.Parse()

	_ = godotenv.Load()
	cfg := config.Load()
	base := cfg.Seeder.APIBaseURL
	imagesDir := cfg.Seeder.ImagesDir
	client := &http.Client{Timeout: 60 * time.Second}

	token, err := login(client, base, cfg.Seeder.AdminLogin, cfg.Seeder.AdminPass)
	if err != nil {
		log.Fatalf("login failed: %v", err)
	}
	log.Printf("authenticated as %q", cfg.Seeder.AdminLogin)

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", cfg.SQLite.Path))
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ids, err := photoIDs(db)
	if err != nil {
		log.Fatalf("read photo ids: %v", err)
	}
	log.Printf("found %d photos in dataset", len(ids))

	var uploaded, skipped, failed int
	for _, id := range ids {
		path := filepath.Join(imagesDir, id+".jpg")
		if _, statErr := os.Stat(path); statErr != nil {
			log.Printf("[SKIP] %s: image file not found at %s", id, path)
			skipped++
			continue
		}

		if !*force {
			exists, headErr := objectExists(client, base, token, id)
			if headErr != nil {
				log.Printf("[FAIL] %s: existence check error: %v", id, headErr)
				failed++
				continue
			}
			if exists {
				log.Printf("[SKIP] %s: object already exists", id)
				skipped++
				continue
			}
		}

		link, linkErr := uploadLink(client, base, token, id)
		if linkErr != nil {
			log.Printf("[FAIL] %s: upload-link error: %v", id, linkErr)
			failed++
			continue
		}

		if uploadErr := uploadFile(client, link, path); uploadErr != nil {
			log.Printf("[FAIL] %s: upload error: %v", id, uploadErr)
			failed++
			continue
		}
		log.Printf("[OK]   %s: uploaded", id)
		uploaded++
	}

	log.Printf("done: uploaded=%d skipped=%d failed=%d total=%d", uploaded, skipped, failed, len(ids))
	if failed > 0 {
		os.Exit(1)
	}
}

func login(client *http.Client, base, login, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"login": login, "password": password})
	resp, err := client.Post(base+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	var lr loginResp
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return "", err
	}
	return lr.AccessToken, nil
}

func photoIDs(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT id FROM photos ORDER BY id ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func objectExists(client *http.Client, base, token, id string) (bool, error) {
	req, _ := http.NewRequest(http.MethodHead, base+"/api/v1/photos/"+id+"/object", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
}

func uploadLink(client *http.Client, base, token, id string) (uploadLinkResp, error) {
	body, _ := json.Marshal(map[string]string{"contentType": "image/jpeg"})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/photos/"+id+"/upload-link", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return uploadLinkResp{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return uploadLinkResp{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	var lr uploadLinkResp
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return uploadLinkResp{}, err
	}
	return lr, nil
}

func uploadFile(client *http.Client, link uploadLinkResp, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPut, link.URL, f)
	if err != nil {
		return err
	}
	req.ContentLength = info.Size()
	for k, v := range link.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "image/jpeg")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PUT status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

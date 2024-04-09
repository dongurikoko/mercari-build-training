package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
)

const (
	ImgDir              = "images"
	ItemsSchemaPath      = "../db/items.db"
	CategoriesSchemaPath = "../db/categories.db"
	DBPath              = "../db/mercari.sqlite3"
)

type Response struct {
	Message string `json:"message"`
}

func root(c echo.Context) error {
	return echo.NewHTTPError(http.StatusOK, "Hello, World!")
}

type Item struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	ImageName string `json:"image_name"`
}

type Items struct {
	Items []Item `json:"items"`
}

type ServerImpl struct {
	db *sql.DB
}

// DBにテーブルを作成する
func (s ServerImpl) createTables() error {
	// スキーマを読み込む
	itemsSchema, err := os.ReadFile(ItemsSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to read items schema: %w", err)
	}
	categoriesSchema, err := os.ReadFile(CategoriesSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to read categories schema: %w", err)
	}

	// テーブルがない場合は作成
	if _, err := s.db.Exec(string(categoriesSchema)); err != nil {
		return fmt.Errorf("failed to create categories table: %w", err)
	}
	if _, err := s.db.Exec(string(itemsSchema)); err != nil {
		return fmt.Errorf("failed to create items table: %w", err)
	}

	return nil
}

func (s ServerImpl) addItem(c echo.Context) error {
	// リクエストボディからデータを取得
	name := c.FormValue("name")
	category := c.FormValue("category")

	// 画像ファイルを取得
	imageFile, err := c.FormFile("image")
	if err != nil {
		c.Logger().Errorf("Failed to get image file in addItem: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "failed to get image file")
	}
	src, err := imageFile.Open()
	if err != nil {
		c.Logger().Errorf("Failed to open image file in addItem: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to open image file")
	}
	defer src.Close()

	// 画像ファイルをハッシュ化
	hash := sha256.New()
	if _, err := io.Copy(hash, src); err != nil {
		c.Logger().Errorf("Failed to hash image file in addItem: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash image file")
	}
	hashedImageName := fmt.Sprintf("%x.jpeg", hash.Sum(nil))

	// DBへの保存
	if err := s.addItemToDB(name, category, hashedImageName); err != nil {
		c.Logger().Errorf("Failed to add item to DB in addItem: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to add item to DB")
	}

	// 画像ファイルを保存
	dst, err := os.Create(fmt.Sprintf("images/%s", hashedImageName))
	if err != nil {
		c.Logger().Errorf("Failed to create image file in addItem: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create image file")
	}
	defer dst.Close()
	src.Seek(0, 0) // ファイルポインタを先頭に戻す
	//srcからdstへ内容をコピー
	if _, err := io.Copy(dst, src); err != nil {
		c.Logger().Errorf("Failed to copy image file in addItem: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to copy image file")
	}

	c.Logger().Infof("Receive item: %s", name)

	message := fmt.Sprintf("item received: name=%s,category=%s,images=%s", name, category, hashedImageName)
	res := Response{Message: message}

	return c.JSON(http.StatusOK, res)

}

func (s ServerImpl) addItemToDB(name, category, imageName string) error {
	// トランザクションを開始
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction in addItemToDB: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil {
			log.Printf("Failed to rollback: %v", err)
		}
	}()

	var id int64
	err = tx.QueryRow("SELECT id FROM categories WHERE name = ?", category).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { // カテゴリが存在しない場合
			stmt1, err := tx.Prepare("INSERT INTO categories (name) VALUES (?)")
			if err != nil {
				return fmt.Errorf("failed to prepare SQL statement1 in addItemToDB: %w", err)
			}
			defer stmt1.Close()

			result, err := stmt1.Exec(category)
			if err != nil {
				return fmt.Errorf("failed to execute SQL statement1 in addItemToDB: %w", err)
			}
			// 新しく挿入された行のIDを取得
			id, err = result.LastInsertId()
			if err != nil {
				return fmt.Errorf("failed to get last insert ID in addItemToDB: %w", err)
			}
		} else {
			return fmt.Errorf("failed to select id from categories in addItemToDB: %w", err)
		}
	}

	// itemsテーブルに商品を追加
	stmt2, err := tx.Prepare("INSERT INTO items (name, category_id, image_name) VALUES (?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare SQL statement2 in addItemToDB: %w", err)
	}
	defer stmt2.Close()
	if _, err := stmt2.Exec(name, id, imageName); err != nil {
		return fmt.Errorf("failed to execute SQL statement2 in addItemToDB: %w", err)
	}

	// トランザクションをコミット
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction in addItemToDB: %w", err)
	}

	return nil
}

func (s ServerImpl) getAllItems(c echo.Context) error {
	// itemsテーブルとcategoriesテーブルをJOINして全てのアイテムを取得
	rows, err := s.db.Query("SELECT items.id, items.name, categories.name, items.image_name FROM items JOIN categories ON items.category_id = categories.id")
	if err != nil {
		c.Logger().Errorf("Failed to search items from DB in getAllItems: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to search items from DB")
	}
	defer rows.Close()

	var allItems Items
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Id, &item.Name, &item.Category, &item.ImageName); err != nil {
			c.Logger().Errorf("Failed to scan items from DB in getAllItems: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to scan items from DB")
		}
		allItems.Items = append(allItems.Items, item)
	}
	return c.JSON(http.StatusOK, allItems)
}

func (s ServerImpl) getItemsByKeyword(c echo.Context) error {
	// クエリパラメータからキーワードを取得
	keyword := c.QueryParam("keyword")

	// DBから名前にキーワードを含む商品一覧を返す
	rows, err := s.db.Query(`
			SELECT items.id, items.name, categories.name, items.image_name 
			FROM items JOIN categories ON items.category_id = categories.id 
			WHERE items.name LIKE '%' || ? || '%'`, keyword)
	if err != nil {
		c.Logger().Errorf("Failed to search items from DB in getItemsByKeyword: %v,keyword: %v", err, keyword)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to search items from DB")
	}
	defer rows.Close()

	var keywordItems Items
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.Id, &item.Name, &item.Category, &item.ImageName); err != nil {
			c.Logger().Errorf("Failed to scan items from DB in getItemsByKeyword: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to scan items from DB")
		}
		keywordItems.Items = append(keywordItems.Items, item)
	}
	return c.JSON(http.StatusOK, keywordItems)
}

func (s ServerImpl) getItemById(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.Logger().Errorf("Failed to convert id to int in getItemById: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "failed to convert id to int")
	}

	// DBからIDに対応する商品を取得
	row := s.db.QueryRow(`
			SELECT items.id, items.name, categories.name, items.image_name 
			FROM items JOIN categories ON items.category_id = categories.id 
			WHERE items.id = ?`, id)

	var item Item
	if err := row.Scan(&item.Id, &item.Name, &item.Category, &item.ImageName); err != nil {
		if errors.Is(err, sql.ErrNoRows) { // IDに対応する商品がない場合
			c.Logger().Errorf("Item not found in DB: id=%d", id)
			return echo.NewHTTPError(http.StatusNotFound, "item not found")
		}
		c.Logger().Errorf("Failed to search item from DB in getItemById: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to search item from DB")
	}
	return c.JSON(http.StatusOK, item)
}

func LoadItemsFromJSON() (*Items, error) {
	jsonFile, err := os.Open("items.json")
	if err != nil {
		return nil, fmt.Errorf("failed to open items.json: %w", err)
	}
	defer jsonFile.Close()

	var allItems Items
	decoder := json.NewDecoder(jsonFile)
	if err := decoder.Decode(&allItems); err != nil {
		return nil, fmt.Errorf("failed to decode items.json: %w", err)
	}
	return &allItems, nil
}

func (s ServerImpl) getImg(c echo.Context) error {
	// itemのidを取得
	id, err := strconv.Atoi(c.Param("imageFilename"))
	if err != nil {
		c.Logger().Errorf("Failed to convert id to int in getImg: %v,", err)
		return echo.NewHTTPError(http.StatusBadRequest, "failed to convert id to int")
	}

	stmt, err := s.db.Prepare("SELECT items.image_name FROM items WHERE items.id=?")
	if err != nil {
		c.Logger().Errorf("Failed to prepare SQL statement in getImg: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to prepare SQL statement")
	}

	var imgName string
	if err := stmt.QueryRow(id).Scan(&imgName); err != nil {
		c.Logger().Errorf("Failed to get image name in getImg: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get image name")
	}

	// imageのパスを作る
	imgPath := path.Join(ImgDir, imgName)

	// imageが存在しない場合はデフォルトの画像を返す
	if _, err := os.Stat(imgPath); err != nil {
		c.Logger().Infof("Image not found: %s", imgPath)
		imgPath = path.Join(ImgDir, "default.jpg")
	}
	return c.File(imgPath)
}

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.SetLevel(log.INFO)

	frontURL := os.Getenv("FRONT_URL")
	if frontURL == "" {
		frontURL = "http://localhost:3000"
	}
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{frontURL},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))

	// DBへの接続
	db, err := sql.Open("sqlite3", DBPath)
	if err != nil {
		e.Logger.Errorf("Failed to open DB: %v", err)
		return
	}
	defer db.Close()

	serverImpl := ServerImpl{db: db}

	// テーブルの作成
	if err := serverImpl.createTables(); err != nil {
		e.Logger.Errorf("Failed to create tables: %v", err)
	}

	// Routes
	e.GET("/", root)
	e.POST("/items", serverImpl.addItem)
	e.GET("/items", serverImpl.getAllItems)
	e.GET("/search", serverImpl.getItemsByKeyword)
	e.GET("/items/:id", serverImpl.getItemById)
	e.GET("/image/:imageFilename", serverImpl.getImg)

	// Start server
	e.Logger.Fatal(e.Start(":9000"))
}

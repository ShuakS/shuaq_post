package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Package struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Timestamp   string `json:"timestamp"`
}

type StatusHistory struct {
	ID        string `json:"id"`
	PackageID string `json:"package_id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

var users = map[string]string{
	"user@example.com": "password123",
	"":                 "", // Example user for demonstration
}

func main() {
	r := gin.Default()
	db, err := sql.Open("sqlite3", "./logistics.db")
	if err != nil {
		log.Fatal(err)
	}

	// Создание таблицы посылок
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS packages (
		id TEXT PRIMARY KEY, 
		status TEXT, 
		description TEXT, 
		timestamp DATETIME
    )`)
	if err != nil {
		log.Fatal(err)
	}

	// Создание таблицы истории статусов
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS status_history (
		id TEXT PRIMARY KEY,
		package_id TEXT,
		status TEXT,
		timestamp DATETIME,
		FOREIGN KEY(package_id) REFERENCES packages(id)
	)`)
	if err != nil {
		log.Fatal(err)
	}

	r.POST("/login", func(c *gin.Context) {
		var user User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if storedPassword, exists := users[user.Email]; exists && storedPassword == user.Password {
			user.Token = uuid.New().String()
			c.JSON(http.StatusOK, user)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		}
	})

	r.GET("/packages", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, status, description, timestamp FROM packages")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var packages []Package
		for rows.Next() {
			var pkg Package
			if err := rows.Scan(&pkg.ID, &pkg.Status, &pkg.Description, &pkg.Timestamp); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			packages = append(packages, pkg)
		}
		c.JSON(http.StatusOK, packages)
	})

	r.GET("/packages/:id/history", func(c *gin.Context) {
		packageID := c.Param("id")
		rows, err := db.Query("SELECT id, package_id, status, timestamp FROM status_history WHERE package_id = ?", packageID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var history []StatusHistory
		for rows.Next() {
			var entry StatusHistory
			if err := rows.Scan(&entry.ID, &entry.PackageID, &entry.Status, &entry.Timestamp); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			history = append(history, entry)
		}
		c.JSON(http.StatusOK, history)
	})

	r.POST("/register", func(c *gin.Context) {
		var newPackage Package
		if err := c.BindJSON(&newPackage); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		newPackage.ID = uuid.New().String()
		newPackage.Status = "registered"
		newPackage.Timestamp = time.Now().Format(time.RFC3339)

		_, err := db.Exec("INSERT INTO packages (id, status, description, timestamp) VALUES (?, ?, ?, ?)",
			newPackage.ID, newPackage.Status, newPackage.Description, newPackage.Timestamp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = db.Exec("INSERT INTO status_history (id, package_id, status, timestamp) VALUES (?, ?, ?, ?)",
			uuid.New().String(), newPackage.ID, newPackage.Status, newPackage.Timestamp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, newPackage)
	})

	r.POST("/update", func(c *gin.Context) {
		var updateRequest struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := c.BindJSON(&updateRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		timestamp := time.Now().Format(time.RFC3339)

		_, err := db.Exec("UPDATE packages SET status = ?, timestamp = ? WHERE id = ?", updateRequest.Status, timestamp, updateRequest.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		_, err = db.Exec("INSERT INTO status_history (id, package_id, status, timestamp) VALUES (?, ?, ?, ?)",
			uuid.New().String(), updateRequest.ID, updateRequest.Status, timestamp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusOK)
	})

	r.Run(":8080")
}

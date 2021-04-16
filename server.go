package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID       string `gorm:"primarykey" json:"id"`
	Username string `gorm:"size:255" json:"username"`

	Password []byte `json:"-"`

	Created int `gorm:"autoCreateTime" json:"created"`
	Updated int `gorm:"autoUpdateTime" json:"updated"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = "usr_" + xid.New().String()
	return nil
}

func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	u.Password, err = bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	return
}

// Run runs the infra server
func Server() error {
	// Load the config file
	// raw, _ := ioutil.ReadFile(options.ConfigPath)
	// if err != nil {
	// 	panic(err)
	// }

	// Extract YAML configuration
	// config := config{}
	// _ = yaml.Unmarshal(raw, &config)
	// if err != nil {
	// 	panic(err)
	// }

	db, err := gorm.Open(sqlite.Open("infra.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Migrate the schema
	db.AutoMigrate(&User{})

	var admin User

	if err = db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		db.Create(&User{Username: "admin", Password: []byte("admin")})
	}

	// Create the admin user if it doesn't exist yet
	router := httprouter.New()

	// User endpoints
	router.GET("/v1/users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var users []User
		db.Find(&users)

		ret, err := json.Marshal(users)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal error"))
		}

		w.Write(ret)
	})

	// Get token or return SSO login flow url to open in a browser
	router.POST("/v1/users/:id/token", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Extract auth information

		// Verify auth or token or jwt
	})

	// Get token or return SSO login flow url to open in a browser
	router.POST("/v1/users/:id/cert", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		// Extract auth information

		// Verify auth or token or jwt
	})

	// Proxy handler for accessing upstream infrastructure
	proxyHandler := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// Extract auth information

		// Create a token
		fmt.Printf("%+v\n", r)
	}

	// Access proxy endpoints
	router.GET("/v1/proxy/*all", proxyHandler)
	router.POST("/v1/proxy/*all", proxyHandler)
	router.PUT("/v1/proxy/*all", proxyHandler)
	router.PATCH("/v1/proxy/*all", proxyHandler)
	router.DELETE("/v1/proxy/*all", proxyHandler)

	// SCIM endpoints
	router.GET("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		fmt.Printf("%+v\n", r)
	})
	router.POST("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		fmt.Printf("%+v\n", r)
	})
	router.PUT("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		fmt.Printf("%+v\n", r)
	})
	router.PATCH("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		fmt.Printf("%+v\n", r)
	})

	fmt.Printf("Listening on port %v\n", 3001)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 3001), router))

	return nil
}

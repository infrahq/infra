package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Run runs the infra server
func Server() error {
	// Initialize the database
	db, err := gorm.Open(sqlite.Open("infra.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Apply schema migrations
	db.AutoMigrate(&User{})

	// Create the admin user if they don't exist yet
	var admin User
	if err = db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		password, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			panic("could not hash password")
		}
		db.Create(&User{Username: "admin", HashedPassword: password})
	}

	// Create the admin user if it doesn't exist yet
	router := gin.Default()
	router.Use(gin.Recovery())

	// User endpoints
	router.GET("/v1/users", func(c *gin.Context) {
		var users []User
		db.Find(&users)
		c.JSON(http.StatusOK, gin.H{"object": "list", "url": "/v1/users", "has_more": false, "data": users})
	})

	router.POST("/v1/users", func(c *gin.Context) {
		type binding struct {
			Username string `form:"username" binding:"required"`
			Password string `form:"password" binding:"required"`
		}

		var form binding
		if err := c.Bind(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(form.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user := &User{Username: form.Username, HashedPassword: hashedPassword}
		if err = db.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, user)
	})

	// Generate credentials for user
	router.POST("/v1/login", func(c *gin.Context) {
		type binding struct {
			Username string `form:"username" binding:"required"`
			Password string `form:"password" binding:"required"`
		}

		var params binding
		if err := c.ShouldBind(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.First(&user, "username = ?", params.Username).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "incorrect credentials"})
			return
		}

		if err = bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(params.Password)); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "incorrect credentials"})
			return
		}

		// Creating Access Token
		claims := jwt.MapClaims{}
		claims["user"] = user.Username
		claims["exp"] = time.Now().Add(time.Minute * 5).Unix()
		unsigned := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err := unsigned.SignedString([]byte("secret")) // TODO: sign with same keypair certificate as certs
		if err != nil {
			panic(err)
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
		})
	})

	remote, err := url.Parse("https://kubernetes.default")
	if err != nil {
		log.Println("could not parse kubernetes endpoint")
	}

	// Load ca
	ca, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err != nil {
		log.Println("could not open cluster ca")
	}

	fmt.Printf("%+v\n", ca)

	satoken, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Println("could not open service account token")
	}

	fmt.Printf("%+v\n", satoken)

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(ca)

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	stripProxy := http.StripPrefix("/v1/proxy", proxy)
	proxyHandler := func(c *gin.Context) {
		authorization := c.Request.Header.Get("Authorization")

		claims := jwt.MapClaims{}
		jwt.ParseWithClaims(strings.Split(authorization, " ")[1], claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		fmt.Printf("%+v\n", claims)

		c.Request.Header.Set("Impersonate-User", claims["user"].(string))
		c.Request.Header.Del("Authorization")
		c.Request.Header.Add("Authorization", "Bearer "+string(satoken))
		stripProxy.ServeHTTP(c.Writer, c.Request)
	}

	// Access proxy endpoints
	router.GET("/v1/proxy/*all", proxyHandler)
	router.POST("/v1/proxy/*all", proxyHandler)
	router.PUT("/v1/proxy/*all", proxyHandler)
	router.PATCH("/v1/proxy/*all", proxyHandler)
	router.DELETE("/v1/proxy/*all", proxyHandler)

	// // SCIM endpoints
	// router.GET("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })
	// router.POST("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })
	// router.PUT("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })
	// router.PATCH("/scim/v2/Users", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// 	fmt.Printf("%+v\n", r)
	// })

	fmt.Printf("Listening on port %v\n", 3001)
	router.Run(":3001")

	return nil
}

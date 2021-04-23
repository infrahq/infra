package server

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Options struct {
	AdminPassword string
	Domain        string
}

type Claims struct {
	jwt.StandardClaims
	User string `json:"user"`
}

func Run(options *Options) error {
	db, err := gorm.Open(sqlite.Open("infra.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&User{})

	adminPassword := options.AdminPassword

	// TODO: make this a better default
	if adminPassword == "" {
		adminPassword = "password"
	}
	var admin User
	if err = db.Where("username = ?", "admin").First(&admin).Error; err != nil {
		password, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
		if err != nil {
			panic("could not hash password")
		}
		db.Create(&User{Username: "admin", HashedPassword: password})
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.GET("/v1/users", func(c *gin.Context) {
		authorization := c.Request.Header.Get("Authorization")
		if authorization == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(strings.Split(authorization, " ")[1], claims, func(token *jwt.Token) (interface{}, error) {
			return []byte("secret"), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		fmt.Println(claims)

		var users []User
		db.Find(&users)
		c.JSON(http.StatusOK, gin.H{"object": "list", "url": "/v1/users", "has_more": false, "data": users})
	})

	router.GET("/v1/users/:id", func(c *gin.Context) {
		type binds struct {
			ID string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.Where("id = ?", params.ID).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	})

	router.POST("/v1/users", func(c *gin.Context) {
		type binds struct {
			Username string `form:"username" binding:"required"`
			Password string `form:"password" binding:"required"`
		}

		var form binds
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

	router.DELETE("/v1/users/:id", func(c *gin.Context) {
		type binds struct {
			ID string `uri:"id" binding:"required"`
		}

		var params binds
		if err := c.BindUri(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result := db.Delete(&User{ID: params.ID})
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"object": "user", "id": params.ID, "deleted": true})
	})

	// Generate credentials for user
	router.POST("/v1/tokens", func(c *gin.Context) {
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

		// Create token
		unsigned := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
			jwt.StandardClaims{
				Issuer:    "http://127.0.0.1:3001", // TODO: replace me with domain or listen address
				IssuedAt:  time.Now().Unix(),
				Subject:   user.ID,
				Audience:  "", // TODO: insert Kubernetes (or other destination) token
				ExpiresAt: time.Now().Add(time.Minute * 5).Unix(),
			},
			user.Username,
		})

		// TODO: use a real key here
		token, err := unsigned.SignedString([]byte("secret"))
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
	satoken, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		log.Println("could not open service account token")
	}
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

		fmt.Println(claims)

		c.Request.Header.Set("Impersonate-User", claims["user"].(string))
		c.Request.Header.Del("Authorization")
		c.Request.Header.Add("Authorization", "Bearer "+string(satoken))
		stripProxy.ServeHTTP(c.Writer, c.Request)
	}

	// Proxy endpoints
	router.GET("/v1/proxy/*all", proxyHandler)
	router.POST("/v1/proxy/*all", proxyHandler)
	router.PUT("/v1/proxy/*all", proxyHandler)
	router.PATCH("/v1/proxy/*all", proxyHandler)
	router.DELETE("/v1/proxy/*all", proxyHandler)

	if options.Domain == "" {
		router.Run(":3001")
	} else {
		log.Fatal(autotls.Run(router, options.Domain))
	}

	fmt.Printf("Listening on port %v\n", 3001)

	return nil
}

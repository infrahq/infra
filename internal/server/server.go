package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Options struct {
	Domain string
}

func initDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("infra.db"), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	db.AutoMigrate(&User{})
	db.AutoMigrate(&Token{})

	return db, nil
}

func TokenAuth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authuser, _, _ := c.Request.BasicAuth()

		tokensk := ""
		if strings.HasPrefix(authuser, "sk_") {
			tokensk = strings.Replace(authuser, "sk_", "", -1)
		}

		authorization := c.Request.Header.Get("Authorization")
		if strings.HasPrefix(authorization, "Bearer sk_") {
			tokensk = strings.Replace(authorization, "Bearer sk_", "", -1)
		}

		if len(tokensk) != TOKEN_LENGTH {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := tokensk[0:16]
		var token Token
		if err := db.Where("id = ?", "tk_"+id).First(&token).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		secret := tokensk[16:TOKEN_LENGTH]
		if err := bcrypt.CompareHashAndPassword(token.HashedSecret, []byte(secret)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		var user User
		if err := db.Where("id = ?", token.UserID).First(&user).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set("user", user.ID)
		c.Set("token", token.ID)

		c.Next()
	}
}

func addRoutes(router *gin.Engine, db *gorm.DB) (err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Println(err)
	} else {
		caFile := config.TLSClientConfig.CAFile
		saToken := config.BearerToken

		ca, err := ioutil.ReadFile(caFile)
		if err != nil {
			fmt.Println(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(ca)

		remote, err := url.Parse(config.Host)
		if err != nil {
			log.Println("could not parse kubernetes endpoint")
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}

		stripProxy := http.StripPrefix("/v1/proxy", proxy)
		proxyHandler := func(c *gin.Context) {
			userID := c.GetString("user")
			user := &User{ID: userID}
			db.First(&user)

			c.Request.Header.Set("Impersonate-User", user.Email)
			c.Request.Header.Del("Authorization")
			c.Request.Header.Add("Authorization", "Bearer "+string(saToken))

			stripProxy.ServeHTTP(c.Writer, c.Request)
		}

		// Proxy endpoints
		// TODO: for usability, should this be based on headers, not a /v1/proxy subpath?
		router.GET("/v1/proxy/*all", proxyHandler)
		router.POST("/v1/proxy/*all", proxyHandler)
		router.PUT("/v1/proxy/*all", proxyHandler)
		router.PATCH("/v1/proxy/*all", proxyHandler)
		router.DELETE("/v1/proxy/*all", proxyHandler)
	}

	updateCRB := func() {
		subjects := []rbacv1.Subject{}

		var users []User
		db.Find(&users)

		for _, v := range users {
			subjects = append(subjects, rbacv1.Subject{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "User",
				Name:     v.Email,
			})
		}

		crb := &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "infra-view",
			},
			Subjects: subjects,
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "view",
			},
		}

		if config != nil {
			// Creates the clientset
			clientset, err := kubernetes.NewForConfig(config)
			if err != nil {
				fmt.Println(err)
			}

			_, err = clientset.RbacV1().ClusterRoleBindings().Update(context.TODO(), crb, metav1.UpdateOptions{})
			if err != nil {
				if k8sErrors.IsNotFound(err) {
					_, err = clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, metav1.CreateOptions{})
					if err != nil {
						fmt.Println(err)
					} else {
						fmt.Println("Cluster role binding added")
					}
				} else {
					fmt.Println(err)
				}
			} else {
				fmt.Println("Cluster role binding patched")
			}
		}
	}

	router.GET("/v1/users", func(c *gin.Context) {
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
			Email string `form:"email" binding:"required"`
		}

		var form binds
		if err := c.Bind(&form); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		user := &User{Email: form.Email}
		if err = db.Create(user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updateCRB()

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

		user := &User{ID: params.ID}
		result := db.First(&user).Delete(&user)
		if result.Error != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": result.Error.Error()})
			return
		}

		if result.RowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		updateCRB()

		// Remove user from view ClusterRoleBinding
		c.JSON(http.StatusOK, gin.H{"object": "user", "id": params.ID, "deleted": true})
	})

	router.POST("/v1/tokens", func(c *gin.Context) {
		type Params struct {
			User string `form:"user"`
		}

		var params Params
		if err := c.ShouldBind(&params); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: verify infra permissions before allowing to specify the user field
		targetUser := params.User
		if targetUser == "" {
			targetUser = c.GetString("user")
		}

		created, token, err := NewToken(db, targetUser)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		}

		// Delete original token
		// TODO: should we do this earlier?
		oldToken := &Token{ID: c.GetString("token")}
		db.Delete(&oldToken)

		c.JSON(http.StatusCreated, gin.H{
			"token":   token,
			"expires": created.Expires,
		})
	})

	return
}

func Run(options *Options) {
	fmt.Println("Starting Infra Engine")

	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	addRoutes(unixRouter, db)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		log.Fatal(err)
	}
	os.Remove(filepath.Join(homeDir, ".infra", "infra.sock"))
	go unixRouter.RunUnix(filepath.Join(homeDir, ".infra", "infra.sock"))

	router := gin.New()
	router.Use(TokenAuth(db))
	addRoutes(router, db)
	if options.Domain == "" {
		fmt.Printf("Listening on port %v\n", 3001)
		router.Run(":3001")
	} else {
		if err = autotls.Run(router, options.Domain); err != nil {
			log.Fatal(err)
		}
	}
}

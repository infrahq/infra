package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TokenAuth(data *Data) gin.HandlerFunc {
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

		if len(tokensk) != SECRET_KEY_LENGTH {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		id := tokensk[0:ID_LENGTH]
		token, err := data.GetToken("tk_"+id, true)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		secret := tokensk[ID_LENGTH:SECRET_KEY_LENGTH]
		if err := bcrypt.CompareHashAndPassword(token.HashedSecret, []byte(secret)); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := data.GetUser(token.User)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		c.Set("user", user.ID)
		c.Set("token", token.ID)

		c.Next()
	}
}

func ProxyHandler(data *Data, kubernetes *Kubernetes) gin.HandlerFunc {
	remote, err := url.Parse(kubernetes.Config.Host)
	if err != nil {
		fmt.Println(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.Transport = kubernetes.Config.Transport
	return func(c *gin.Context) {
		userID := c.GetString("user")
		user, err := data.GetUser(userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid user"})
			return
		}

		c.Request.Header.Del("Authorization")
		c.Request.Header.Set("Impersonate-User", user.Email)

		http.StripPrefix("/v1/proxy", proxy).ServeHTTP(c.Writer, c.Request)
	}
}

func UpdateKubernetesClusterRoleBindings(data *Data, kubernetes *Kubernetes) error {
	if data == nil || kubernetes == nil {
		return nil
	}

	users, err := data.ListUsers()
	if err != nil {
		return err
	}

	emails := []string{}
	for _, v := range users {
		emails = append(emails, v.Email)
	}

	return kubernetes.UpdateRoleBindings(emails)
}

func addRoutes(router *gin.Engine, data *Data, kubernetes *Kubernetes) error {
	router.GET("/v1/users", func(c *gin.Context) {
		users, err := data.ListUsers()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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

		user, err := data.GetUser(params.ID)
		if err != nil {
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

		user, err := data.CreateUser(form.Email)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdateKubernetesClusterRoleBindings(data, kubernetes); err != nil {
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

		if _, err := data.GetUser(params.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user does not exist"})
			return
		}

		if err := data.DeleteUser(params.ID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := UpdateKubernetesClusterRoleBindings(data, kubernetes); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

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

		// TODO(jmorganca): verify permissions before allowing to specify the user field
		targetUser := params.User
		if targetUser == "" {
			targetUser = c.GetString("user")
			data.DeleteToken(c.GetString("token"))
		}

		created, sk, err := data.CreateToken(targetUser)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		}

		c.JSON(http.StatusCreated, gin.H{
			"token":        created,
			"secret_token": sk,
		})
	})

	if kubernetes != nil {
		proxyHandler := ProxyHandler(data, kubernetes)
		router.GET("/v1/proxy/*all", proxyHandler)
		router.POST("/v1/proxy/*all", proxyHandler)
		router.PUT("/v1/proxy/*all", proxyHandler)
		router.PATCH("/v1/proxy/*all", proxyHandler)
		router.DELETE("/v1/proxy/*all", proxyHandler)
	}

	return nil
}

type ServerOptions struct {
	DBPath string
}

func ServerRun(options *ServerOptions) error {
	data, err := NewData(options.DBPath)
	if err != nil {
		return err
	}

	defer data.Close()

	kubernetes, err := NewKubernetes()
	if err != nil {
		fmt.Println("warning: no kubernetes cluster detected.")
	}

	gin.SetMode(gin.ReleaseMode)

	unixRouter := gin.New()
	if err = addRoutes(unixRouter, data, kubernetes); err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	if err = os.MkdirAll(filepath.Join(homeDir, ".infra"), os.ModePerm); err != nil {
		return err
	}

	os.Remove(filepath.Join(homeDir, ".infra", "infra.sock"))
	go func() {
		if err := unixRouter.RunUnix(filepath.Join(homeDir, ".infra", "infra.sock")); err != nil {
			log.Fatal(err)
		}
	}()

	router := gin.New()
	router.Use(TokenAuth(data))
	if err = addRoutes(router, data, kubernetes); err != nil {
		return err
	}

	if err = router.Run(":2378"); err != nil {
		return err
	}

	return nil
}

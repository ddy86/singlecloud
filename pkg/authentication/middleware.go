package authentication

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	CASLoginPath    = "/cas/login"
	CASLogoutPath   = "/cas/logout"
	CASRedirectPath = "/index"
	CASRolePath     = "/cas/role"
)

func indexPath(r *http.Request, index string) string {
	u, _ := url.Parse(r.URL.String())
	u.Host = r.Host
	u.Scheme = "http"
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		u.Scheme = scheme
	} else if r.TLS != nil {
		u.Scheme = "https"
	}
	u.Path = index

	q := u.Query()
	for _, cleanParam := range []string{"ticket", "service"} {
		q.Del(cleanParam)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func (a *Authenticator) RegisterHandler(router gin.IRoutes) error {
	router.GET(CASRolePath, func(c *gin.Context) {
		if a.CasAuth != nil {
			user, _ := a.CasAuth.Authenticate(c.Writer, c.Request)
			body, _ := json.Marshal(map[string]string{
				"user": user,
			})
			c.Writer.WriteHeader(http.StatusOK)
			c.Writer.Write(body)
		}
	})

	router.GET(CASLogoutPath, func(c *gin.Context) {
		if a.CasAuth != nil {
			a.CasAuth.RedirectToLogout(c.Writer, c.Request, CASLoginPath)
		}
	})

	return nil
}

func (a *Authenticator) MiddlewareFunc() gin.HandlerFunc {
	var exceptionalPath = []string{
		"/apis",
		"/login",
		"/assets",
		"/cas",
	}

	return func(c *gin.Context) {
		userName, err := a.Authenticate(c.Writer, c.Request)
		if err != nil {
			log.Errorf("auth failed:%v", err)
			return
		}

		if userName != "" {
			ctx := context.WithValue(c.Request.Context(), types.CurrentUserKey, userName)
			c.Request = c.Request.WithContext(ctx)
		} else {
			path := c.Request.URL.Path
			doRedirect := true
			for _, ep := range exceptionalPath {
				if strings.HasPrefix(path, ep) {
					doRedirect = false
					break
				}
			}
			if doRedirect == false {
				return
			}

			fmt.Printf("--> do redirect for path %v\n", path)
			if a.CasAuth != nil {
				a.CasAuth.RedirectToLogin(c.Writer, c.Request, CASRedirectPath)
			} else {
				http.Redirect(c.Writer, c.Request, indexPath(c.Request, "/login"), http.StatusFound)
			}
		}
	}
}

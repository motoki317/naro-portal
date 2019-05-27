package router

import (
	"fmt"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequestBody struct {
	Username string `json:"username,omitempty" form:"username"`
	Password string `json:"password,omitempty" form:"password"`
}

type User struct {
	Username   string `json:"username,omitempty"  db:"Username"`
	HashedPass string `json:"-"  db:"HashedPass"`
}

// SetupLoginRoutes /login, /signup ルートを置きます
func SetupLoginRoutes(e *echo.Echo, db *sqlx.DB) {
	e.POST("/login", makePostLoginHandler(db))
	e.POST("/signup", makePostSignUpHandler(db))
	e.POST("/logout", makePostLogoutHandler(db), CheckLogin)
}

func makePostLoginHandler(db *sqlx.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		req := LoginRequestBody{}
		c.Bind(&req)

		user := User{}
		err := db.Get(&user, "SELECT * FROM users WHERE username=?", req.Username)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
		}

		err = bcrypt.CompareHashAndPassword([]byte(user.HashedPass), []byte(req.Password))
		if err != nil {
			if err == bcrypt.ErrMismatchedHashAndPassword {
				return c.NoContent(http.StatusForbidden)
			} else {
				return c.NoContent(http.StatusInternalServerError)
			}
		}

		sess, err := session.Get("sessions", c)
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, "something wrong in getting session")
		}
		sess.Values["userName"] = req.Username
		sess.Save(c.Request(), c.Response())

		return c.NoContent(http.StatusOK)
	}
}

func makePostSignUpHandler(db *sqlx.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		req := LoginRequestBody{}
		c.Bind(&req)

		if req.Password == "" || req.Username == "" {
			return c.String(http.StatusBadRequest, "項目が空です")
		}

		hashedPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("bcrypt generate error: %v", err))
		}

		var count int

		err = db.Get(&count, "SELECT COUNT(*) FROM users WHERE Username=?", req.Username)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
		}

		if count > 0 {
			return c.String(http.StatusConflict, "ユーザーが既に存在しています")
		}

		_, err = db.Exec("INSERT INTO users (Username, HashedPass) VALUES (?, ?)", req.Username, hashedPass)
		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("db error: %v", err))
		}
		return c.NoContent(http.StatusCreated)
	}
}

func makePostLogoutHandler(db *sqlx.DB) func(c echo.Context) error {
	return func(c echo.Context) error {
		sess, err := session.Get("sessions", c)

		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, "Something went wrong while processing your session...")
		}

		_, err = db.Exec("DELETE FROM sessions WHERE id=?", sess.ID)

		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, "Something went wrong while processing your session...")
		}

		return c.NoContent(http.StatusNoContent)
	}
}

func CheckLogin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("sessions", c)
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, "Something went wrong while processing your session...")
		}

		if sess.Values["userName"] == nil {
			return c.String(http.StatusForbidden, "You are not logged in.")
		}
		c.Set("userName", sess.Values["userName"].(string))

		return next(c)
	}
}

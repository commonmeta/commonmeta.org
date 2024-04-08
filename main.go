package main

import (
	"fmt"
	"log"
	"regexp"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	// redirect hard-coded legacy urls to docs site
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/", func(c echo.Context) error {
			return c.Redirect(301, "https://docs.commonmeta.org/")
		})
		e.Router.GET("/challenges.html", func(c echo.Context) error {
			return c.Redirect(301, "https://docs.commonmeta.org/challenges.html")
		})
		e.Router.GET("/implementations.html", func(c echo.Context) error {
			return c.Redirect(301, "https://docs.commonmeta.org/implementations.html")
		})
		e.Router.GET("/use-cases.html", func(c echo.Context) error {
			return c.Redirect(301, "https://docs.commonmeta.org/use-cases.html")
		})
		e.Router.GET("/schema.html", func(c echo.Context) error {
			return c.Redirect(301, "https://docs.commonmeta.org/schema.html")
		})
		return nil
	})

	// retrieve a single works collection record and redirect to its url
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/:str", func(c echo.Context) error {
			str := c.PathParam("str")
			if str == "" {
				return c.NoContent(404)
			}
			isDoi, err := regexp.MatchString(`^10\.\d{4,9}/.+$`, str)
			if err != nil {
				return err
			}
			var pid string
			if isDoi {
				pid = fmt.Sprintf("https://doi.org/%s", str)
			} else {
				pid = fmt.Sprintf("https://%s", str)
			}
			record, err := app.Dao().FindFirstRecordByData("works", "pid", pid)
			if err != nil {
				return err
			} else if record == nil {
				return c.NoContent(404)
			} else {
				return c.Redirect(302, record.GetString("url"))
			}
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

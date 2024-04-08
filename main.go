package main

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

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

	// retrieve a single works collection record and either redirect to its url
	// or return metadata depending on the Accept header
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/:str", func(c echo.Context) error {
			// fetch the pid
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

			// Retrieve the content type from the Accept header,
			// alternatively extract from the URL path
			acceptHeaders := c.Request().Header.Get("Accept")
			contentType := strings.Split(acceptHeaders, ",")[0]
			if contentType == "" {
				contentType = "text/html"
			}
			u, err := url.Parse(pid)
			if err != nil {
				return err
			}
			path := strings.Split(u.Path, "/")
			if len(path) > 3 && path[len(path)-3] == "transform" {
				u.Path = strings.Join(path[:len(path)-3], "/")
				pid = u.String()
				contentType = strings.Join(path[len(path)-2:], "/")
			}
			record, err := app.Dao().FindFirstRecordByData("works", "pid", pid)
			if err != nil {
				return err
			} else if record == nil {
				return c.NoContent(404)
			}
			switch contentType {
			case "text/html":
				// redirect to resource
				return c.Redirect(302, record.GetString("url"))
			case "application/vnd.commonmeta+json", "application/json":
				// return metadata in commonmeta JSON format
				return c.JSON(200, record)
			default:
				// all other Content-Types not (yet) supported
				return c.JSON(406, map[string]string{"error": fmt.Sprintf("Content-Type %s not supported", str)})
			}
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

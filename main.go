package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	type Reference struct {
		Doi             string `json:"doi,omitempty"`
		Url             string `json:"url,omitempty"`
		Key             string `json:"key"`
		PublicationYear string `json:"publicationYear,omitempty"`
		Title           string `json:"title,omitempty"`
	}

	// redirect hard-coded legacy urls to docs site
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "https://docs.commonmeta.org/")
		})
		e.Router.GET("/challenges.html", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "https://docs.commonmeta.org/challenges.html")
		})
		e.Router.GET("/implementations.html", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "https://docs.commonmeta.org/implementations.html")
		})
		e.Router.GET("/use-cases.html", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "https://docs.commonmeta.org/use-cases.html")
		})
		e.Router.GET("/schema.html", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "https://docs.commonmeta.org/schema.html")
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
				return c.NoContent(http.StatusNotFound)
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
			if contentType == "" || contentType == "*/*" {
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

			if contentType == "text/html" {
				// redirect to resource
				return c.Redirect(http.StatusFound, record.GetString("url"))
			}

			// extract references and look up their metadata
			var r []Reference
			err = record.UnmarshalJSONField("references", &r)
			if err != nil {
				return err
			}
			if len(r) > 0 {
				refs := make([]interface{}, len(r))
				for i, v := range r {
					if v.Doi != "" {
						refs[i] = v.Doi
					} else if v.Url != "" {
						refs[i] = v.Url
					}
				}
				references, err := app.Dao().FindRecordsByExpr("works",
					dbx.In("pid", refs...))
				if err != nil {
					return err
				}
				if len(references) > 0 {
					record.Set("references", references)
				}
			}

			switch contentType {
			case "application/vnd.commonmeta+json", "application/json":
				// return metadata in Commonmeta format
				return c.JSON(http.StatusOK, record)
			default:
				// all other Content-Types not (yet) supported
				return c.JSON(http.StatusNotAcceptable, map[string]string{"error": fmt.Sprintf("Content-Type %s not supported", contentType)})
			}
		})

		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

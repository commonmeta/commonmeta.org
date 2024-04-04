package main

import (
	"fmt"
	"log"
	"os"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	// retrieve a single works collection record by doi
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/works/:doi", func(c echo.Context) error {
			doi := c.PathParam("doi")
			if doi == "" {
				return c.NoContent(404)
			}

			pid := fmt.Sprintf("https://doi.org/%s", doi)
			record, err := app.Dao().FindFirstRecordByData("works", "pid", pid)
			if err != nil {
				return err
			} else if record == nil {
				return c.NoContent(404)
			} else {
				return c.JSON(200, record)
			}
		})

		return nil
	})

	// serves static files from the provided public dir (if exists)
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS("./pb_public"), false))
		return nil
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}

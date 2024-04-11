package main

import (
	"encoding/json"
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
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
)

// ensures that the Work struct satisfy the models.Model interface
var _ models.Model = (*Work)(nil)

type Work struct {
	models.BaseModel

	// required fields
	Pid  string `db:"pid" json:"id"`
	Type string `db:"type" json:"type"`

	// optional fields
	AdditionalType       string        `db:"additional_type" json:"additional_type,omitempty"`
	Url                  string        `db:"url" json:"url,omitempty"`
	Contributors         types.JsonRaw `db:"contributors" json:"contributors,omitempty"`
	Publisher            types.JsonRaw `db:"publisher" json:"publisher,omitempty"`
	Date                 types.JsonRaw `db:"date" json:"date,omitempty"`
	Titles               types.JsonRaw `db:"titles" json:"titles,omitempty"`
	Container            types.JsonRaw `db:"container" json:"container,omitempty"`
	Subjects             types.JsonRaw `db:"subjects" json:"subjects,omitempty"`
	Sizes                types.JsonRaw `db:"sizes" json:"sizes,omitempty"`
	Formats              types.JsonRaw `db:"formats" json:"formats,omitempty"`
	Language             string        `db:"language" json:"language,omitempty"`
	License              types.JsonRaw `db:"license" json:"license,omitempty"`
	Version              string        `db:"version" json:"version,omitempty"`
	References           types.JsonRaw `db:"references" json:"references,omitempty"`
	Relations            types.JsonRaw `db:"relations" json:"relations,omitempty"`
	FundingReferences    types.JsonRaw `db:"funding_references" json:"funding_references,omitempty"`
	Descriptions         types.JsonRaw `db:"descriptions" json:"descriptions,omitempty"`
	GeoLocations         types.JsonRaw `db:"geo_locations" json:"geo_locations,omitempty"`
	Provider             string        `db:"provider" json:"provider,omitempty"`
	AlternateIdentifiers types.JsonRaw `db:"alternate_identifiers" json:"alternate_identifiers,omitempty"`
	Files                types.JsonRaw `db:"files" json:"files,omitempty"`
	ArchiveLocations     types.JsonRaw `db:"archive_locations" json:"archive_locations,omitempty"`
	Created              string        `db:"created" json:"created"`
	Updated              string        `db:"updated" json:"updated"`
}

func (m *Work) TableName() string {
	return "works" // the name of your collection
}

func main() {
	app := pocketbase.New()

	type Reference struct {
		Doi             string `json:"doi,omitempty"`
		Url             string `json:"url,omitempty"`
		Key             string `json:"key"`
		PublicationYear string `json:"publicationYear,omitempty"`
		Title           string `json:"title,omitempty"`
	}

	type File struct {
		Url      string `json:"url"`
		MimeType string `json:"mimeType"`
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

			work, err := FindWorkByPid(app.Dao(), pid)
			if err != nil {
				return err
			} else if work == nil {
				return c.NoContent(404)
			}

			if contentType == "text/html" {
				// redirect to resource
				return c.Redirect(http.StatusFound, work.Url)
			}

			// extract pids of references and look up their metadata
			var r []Reference
			err = json.Unmarshal(work.References, &r)
			if err != nil {
				return err
			}
			if len(r) > 0 {
				// generate a list of pid strings
				refs := make([]string, len(r))
				for i, v := range r {
					if v.Doi != "" {
						refs[i] = v.Doi
					} else if v.Url != "" {
						refs[i] = v.Url
					}
				}
				references, err := FindWorksByPids(app.Dao(), refs...)
				if err != nil {
					return err
				}
				if len(references) > 0 {
					work.References, err = json.Marshal(references)
					if err != nil {
						return err
					}
					// dont save the updated references yet
					// if err := app.Dao().Save(work); err != nil {
					// 	return err
					// }
				}
			}

			// extract files and look up their metadata
			var f []File
			err = json.Unmarshal(work.Files, &f)
			if err != nil {
				return err
			}
			files := make(map[string]string)
			for _, v := range f {
				files[v.MimeType] = v.Url
			}
			markdownUrl := files["text/markdown"]
			pdfUrl := files["application/pdf"]
			jatsUrl := files["application/xml"]
			if jatsUrl == "" {
				jatsUrl = files["application/vnd.jats+xml"]
			}

			switch contentType {
			case "application/vnd.commonmeta+json", "application/json":
				// return metadata in Commonmeta format
				return c.JSON(http.StatusOK, work)
			case "text/markdown":
				// redirect to markdown version of the resource if available
				if markdownUrl == "" {
					return c.JSON(http.StatusNotAcceptable, map[string]string{"error": "Markdown version not available"})
				}
				return c.Redirect(http.StatusFound, markdownUrl)
			case "application/vnd.jats+xml", "application/xml":
				// redirect to JATS XML version of the resource if available
				if jatsUrl == "" {
					return c.JSON(http.StatusNotAcceptable, map[string]string{"error": "JATS XML version not available"})
				}
				return c.Redirect(http.StatusFound, jatsUrl)
			case "application/pdf":
				// redirect to PDF version of the resource if available
				if pdfUrl == "" {
					return c.JSON(http.StatusNotAcceptable, map[string]string{"error": "PDF version not available"})
				}
				return c.Redirect(http.StatusFound, pdfUrl)
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

func WorkQuery(dao *daos.Dao) *dbx.SelectQuery {
	return dao.ModelQuery(&Work{})
}

// find single work by pid
func FindWorkByPid(dao *daos.Dao, pid string) (*Work, error) {
	work := &Work{}

	err := WorkQuery(dao).
		// case insensitive match
		AndWhere(dbx.NewExp("LOWER(pid)={:pid}", dbx.Params{
			"pid": strings.ToLower(pid),
		})).
		Limit(1).
		One(work)

	if err != nil {
		return nil, err
	}

	return work, nil
}

// find multiple works by their pids. Use variadic arguments to pass in the pids
func FindWorksByPids(dao *daos.Dao, pids ...string) ([]*Work, error) {
	works := []*Work{}

	// convert pids to a slice of interface{} to use with dbx.In
	refs := make([]interface{}, len(pids))
	for i, v := range pids {
		refs[i] = v
	}

	err := WorkQuery(dao).
		AndWhere(dbx.In("pid", refs...)).
		All(&works)

	if err != nil {
		return nil, err
	}

	return works, nil
}

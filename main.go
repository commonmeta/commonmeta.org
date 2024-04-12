package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"slices"
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
	AdditionalType       string         `db:"additional_type" json:"additional_type,omitempty"`
	Url                  string         `db:"url" json:"url,omitempty"`
	Contributors         types.JsonRaw  `db:"contributors" json:"contributors,omitempty"`
	Publisher            types.JsonRaw  `db:"publisher" json:"publisher,omitempty"`
	Date                 types.JsonRaw  `db:"date" json:"date,omitempty"`
	Titles               types.JsonRaw  `db:"titles" json:"titles,omitempty"`
	Container            types.JsonRaw  `db:"container" json:"container,omitempty"`
	Subjects             types.JsonRaw  `db:"subjects" json:"subjects,omitempty"`
	Sizes                types.JsonRaw  `db:"sizes" json:"sizes,omitempty"`
	Formats              types.JsonRaw  `db:"formats" json:"formats,omitempty"`
	Language             string         `db:"language" json:"language,omitempty"`
	License              types.JsonRaw  `db:"license" json:"license,omitempty"`
	Version              string         `db:"version" json:"version,omitempty"`
	References           types.JsonRaw  `db:"references" json:"references,omitempty"`
	Relations            types.JsonRaw  `db:"relations" json:"relations,omitempty"`
	FundingReferences    types.JsonRaw  `db:"funding_references" json:"funding_references,omitempty"`
	Descriptions         types.JsonRaw  `db:"descriptions" json:"descriptions,omitempty"`
	GeoLocations         types.JsonRaw  `db:"geo_locations" json:"geo_locations,omitempty"`
	Provider             string         `db:"provider" json:"provider,omitempty"`
	AlternateIdentifiers types.JsonRaw  `db:"alternate_identifiers" json:"alternate_identifiers,omitempty"`
	Files                types.JsonRaw  `db:"files" json:"files,omitempty"`
	ArchiveLocations     types.JsonRaw  `db:"archive_locations" json:"archive_locations,omitempty"`
	Created              types.DateTime `db:"created" json:"created"`
	Updated              types.DateTime `db:"updated" json:"updated"`
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
		e.Router.GET("/robots.txt", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "https://docs.commonmeta.org/robots.txt")
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

			// extract optional content type from the URL path
			u, err := url.Parse(pid)
			if err != nil {
				return err
			}
			contentType := ""
			path := strings.Split(u.Path, "/")
			if len(path) > 3 && path[len(path)-3] == "transform" {
				u.Path = strings.Join(path[:len(path)-3], "/")
				pid = u.String()
				str = u.Path[1:]
				contentType = strings.Join(path[len(path)-2:], "/")
			}

			// alternatively extract the content type from the Accept header
			if contentType == "" {
				acceptHeaders := c.Request().Header.Get("Accept")
				contentType = strings.Split(acceptHeaders, ",")[0]
			}
			if contentType == "" || contentType == "*/*" {
				contentType = "text/html"
			}

			work, err := FindWorkByPid(app.Dao(), pid)
			if err != nil {
				return err
			}

			// supported content types
			contentTypes := []string{"text/html", "application/vnd.commonmeta+json", "application/json", "text/markdown", "application/vnd.jats+xml", "application/xml", "application/pdf"}
			if work == nil || !slices.Contains(contentTypes, contentType) {
				if contentType == "text/html" {
					// look up minimal metadata and store in works collection
					log.Printf("%s not found, finding elsewhere ...", pid)
					work, err = CreateWorkByPid(app.Dao(), pid)
					if err != nil {
						return c.JSON(http.StatusNotFound, map[string]string{"error": "Not found"})
					}
					return c.Redirect(http.StatusFound, work.Url)
				} else if contentType == "application/vnd.commonmeta+json" || contentType == "application/json" {
					// cant (yet) handle commonmeta content type, and not supported by Crossref or DataCite content negotiation
					log.Printf("%s not converted to commonmeta", pid)
					return c.JSON(http.StatusNotFound, map[string]string{"error": "Work not yet converted to Commonmeta format"})
				}
				// content negotiation is not supported for redirects
				// look up the DOI registration agency in works table and use link-based content negotiation
				ra, err := FindDoiRegistrationAgency(app.Dao(), pid)
				if err != nil {
					return err
				}
				switch ra {
				case "Crossref":
					return c.Redirect(http.StatusFound, fmt.Sprintf("https://api.crossref.org/works/%s/transform/%s", str, contentType))
				case "DataCite":
					return c.Redirect(http.StatusFound, fmt.Sprintf("https://data.crosscite.org/%s/%s", contentType, str))
				default:
					log.Printf("Doi registration agency for %s not found", pid)
					return c.JSON(http.StatusNotFound, map[string]string{"error": "Work not found and content negotiation not supported"})
				}
			}

			if contentType == "text/html" {
				// redirect to resource if work found
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

			// return error if work not yet converted to Commonmeta format
			if work.Type == "" {
				return c.JSON(http.StatusNotAcceptable, map[string]string{"error": "Work not yet converted to Commonmeta format"})
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

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return work, nil
}

// create work by pid, currently only supports DOIs
func CreateWorkByPid(dao *daos.Dao, pid string) (*Work, error) {
	u, err := url.Parse(pid)
	if err != nil {
		return nil, err
	}
	isDoi := u.Host == "doi.org"
	if !isDoi {
		return nil, fmt.Errorf("Only DOIs are supported")
	}

	// disable http redirects
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Head(pid)
	if err != nil {
		return nil, err
	}
	url := resp.Header.Get("Location")
	provider, err := FindDoiRegistrationAgency(dao, pid)
	if err != nil {
		return nil, err
	}
	// create minimal record in the works collection
	work := &Work{
		Pid:                  pid,
		Type:                 "",
		AdditionalType:       "",
		Url:                  url,
		Contributors:         types.JsonRaw("[]"),
		Publisher:            types.JsonRaw("{}"),
		Date:                 types.JsonRaw("{}"),
		Titles:               types.JsonRaw("[]"),
		Container:            types.JsonRaw("{}"),
		Subjects:             types.JsonRaw("[]"),
		Sizes:                types.JsonRaw("[]"),
		Formats:              types.JsonRaw("[]"),
		Language:             "",
		License:              types.JsonRaw("{}"),
		Version:              "",
		References:           types.JsonRaw("[]"),
		Relations:            types.JsonRaw("[]"),
		FundingReferences:    types.JsonRaw("[]"),
		Descriptions:         types.JsonRaw("[]"),
		GeoLocations:         types.JsonRaw("[]"),
		Provider:             provider,
		AlternateIdentifiers: types.JsonRaw("[]"),
		Files:                types.JsonRaw("[]"),
		ArchiveLocations:     types.JsonRaw("[]"),
		Created:              types.NowDateTime(),
		Updated:              types.NowDateTime(),
	}

	if err := dao.Save(work); err != nil {
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

// find DOI registration agency from works collection
func FindDoiRegistrationAgency(dao *daos.Dao, doi string) (string, error) {
	substr := doi[0:24]
	work := &Work{}

	err := WorkQuery(dao).
		AndWhere(dbx.NewExp("pid LIKE {:substr}", dbx.Params{
			"substr": substr + "%",
		})).
		Limit(1).
		One(work)

	if err == sql.ErrNoRows {
		ra, err := FindDoiRegistrationAgencyFromHandle(dao, substr)
		if err != nil {
			return "", err
		}
		return ra, nil
	} else if err != nil {
		return "", err
	}

	return work.Provider, nil
}

// find DOI registration agency for prefix from handle service
func FindDoiRegistrationAgencyFromHandle(dao *daos.Dao, doi string) (string, error) {
	type Response []struct {
		DOI string `json:"DOI"`
		RA  string `json:"RA"`
	}
	u, err := url.Parse(doi)
	if err != nil {
		return "", err
	}
	substr := u.Path[1:]
	resp, err := http.Get(fmt.Sprintf("https://doi.org/ra/%s", substr))
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	return result[0].RA, nil
}

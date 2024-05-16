package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/front-matter/commonmeta/commonmeta"
	"github.com/front-matter/commonmeta/crossref"
	"github.com/front-matter/commonmeta/crossrefxml"
	"github.com/front-matter/commonmeta/csl"
	"github.com/front-matter/commonmeta/datacite"
	"github.com/front-matter/commonmeta/doiutils"
	"github.com/front-matter/commonmeta/schemaorg"
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
	AdditionalType    string        `db:"additionalType" json:"additionalType,omitempty"`
	ArchiveLocations  types.JsonRaw `db:"archiveLocations" json:"archiveLocations,omitempty"`
	Container         types.JsonRaw `db:"container" json:"container,omitempty"`
	Contributors      types.JsonRaw `db:"contributors" json:"contributors,omitempty"`
	Date              types.JsonRaw `db:"date" json:"date,omitempty"`
	Descriptions      types.JsonRaw `db:"descriptions" json:"descriptions,omitempty"`
	Files             types.JsonRaw `db:"files" json:"files,omitempty"`
	FundingReferences types.JsonRaw `db:"fundingReferences" json:"fundingReferences,omitempty"`
	GeoLocations      types.JsonRaw `db:"geoLocations" json:"geoLocations,omitempty"`
	Identifiers       types.JsonRaw `db:"identifiers" json:"identifiers,omitempty"`
	Language          string        `db:"language" json:"language,omitempty"`
	License           types.JsonRaw `db:"license" json:"license,omitempty"`
	Provider          string        `db:"provider" json:"provider,omitempty"`
	Publisher         types.JsonRaw `db:"publisher" json:"publisher,omitempty"`
	References        types.JsonRaw `db:"references" json:"references,omitempty"`
	Relations         types.JsonRaw `db:"relations" json:"relations,omitempty"`
	Subjects          types.JsonRaw `db:"subjects" json:"subjects,omitempty"`
	Titles            types.JsonRaw `db:"titles" json:"titles,omitempty"`
	Url               string        `db:"url" json:"url,omitempty"`
	Version           string        `db:"version" json:"version,omitempty"`

	// database fields
	Created types.DateTime `db:"created" json:"created"`
	Updated types.DateTime `db:"updated" json:"updated"`
}

func (m *Work) TableName() string {
	return "works" // the name of your collection
}

func main() {
	app := pocketbase.New()

	type File struct {
		Url      string `json:"url"`
		MimeType string `json:"mimeType"`
	}

	type Reference struct {
		Key             string `json:"key"`
		ID              string `json:"id,omitempty"`
		Type            string `json:"type,omitempty"`
		Title           string `json:"title,omitempty"`
		PublicationYear string `json:"publicationYear,omitempty"`
		Unstructured    string `json:"unstructured,omitempty"`
	}

	// redirect hard-coded legacy urls to docs site
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.GET("/", func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, "https://docs.commonmeta.org/")
		})
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/commonmeta_v0.14.json", func(c echo.Context) error {
			return c.Redirect(http.StatusTemporaryRedirect, "https://docs.commonmeta.org/commonmeta_v0.14.json")
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
			isDoi, err := regexp.MatchString(`10\.\d{4,9}/.+`, str)
			if err != nil {
				return err
			}
			var pid string
			if isDoi {
				pid = fmt.Sprintf("https://doi.org/%s", str)
			} else {
				pid = fmt.Sprintf("https://%s", str)
			}

			// check if the pid is a valid URL
			u, err := url.ParseRequestURI(pid)
			if err != nil {
				return err
			}

			// extract optional content type from the URL path
			contentType := ""
			path := strings.Split(u.Path, "/")
			if len(path) > 3 && path[len(path)-3] == "transform" {
				// Crossref link-based content type requests
				u.Path = strings.Join(path[:len(path)-3], "/")
				pid, _ = url.QueryUnescape(u.String())
				str = u.Path[1:]
				contentType = strings.Join(path[len(path)-2:], "/")
			} else if len(path) > 3 && (path[1] == "application" || path[1] == "text") {
				// DataCite link-based content type requests
				u.Path = strings.Join(path[3:], "/")
				pid, _ = url.QueryUnescape(u.String())
				str = u.Path[1:]
				contentType = strings.Join(path[1:3], "/")
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

			// create a new work record if not found and the pid is a Crossref DOI
			if work == nil {
				ra, err := FindDoiRegistrationAgency(app.Dao(), pid)
				if err != nil {
					return err
				}
				if isDoi && ra == "Crossref" {
					log.Printf("%s not found, looking up metadata with Crossref ...", pid)
					data, err := crossref.Fetch(pid)

					if err != nil {
						return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
					}
					newWork := GetWorkFromCommonmeta(data)
					if err := app.Dao().Save(newWork); err != nil {
						return err
					}
					work, err = FindWorkByPid(app.Dao(), newWork.Pid)
					if err != nil {
						return err
					}
				} else if isDoi && ra == "DataCite" {
					log.Printf("%s not found, looking up metadata with DataCite ...", pid)
					data, err := datacite.Fetch(pid)
					if err != nil {
						return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
					}
					newWork := GetWorkFromCommonmeta(data)
					log.Printf("New work: %+v\n", newWork)
					if err := app.Dao().Save(newWork); err != nil {
						return err
					}
					work, err = FindWorkByPid(app.Dao(), newWork.Pid)
					if err != nil {
						return err
					}
				}
			}
			if work == nil {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Not found"})
			}

			// redirect for content types supported by Crossref or DataCite DOI content negotiation
			contentTypes := []string{"text/html", "application/vnd.commonmeta+json", "application/json", "application/vnd.datacite.datacite+json", "application/vnd.citationstyles.csl+json", "application/vnd.crossref.unixsd+xml", "application/vnd.schemaorg.ld+json", "text/markdown", "application/vnd.jats+xml", "application/xml", "application/pdf"}
			if !slices.Contains(contentTypes, contentType) {
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
				refs := make([]string, 0)
				for _, v := range r {
					if v.ID != "" {
						refs = append(refs, v.ID)
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
					// if err := app.Dao().Save(work); err != nil {
					// 	return err
					// }
				}
				// TODO: change how we store references in the works collection,
				// should be a slice of strings instead of a slice of structs,
				// and uses the pid as the key. This will enable simpler sql queries.
				// citations, err := FindWorksByCitation(app.Dao(), pid)
				// if err != nil {
				// 	return err
				// }
				// if len(citations) > 0 {
				// 	log.Printf("Citations: %+v\n", citations)
				// }
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
			var markdownUrl, pdfUrl, jatsUrl string
			markdownUrl = files["text/markdown"]
			pdfUrl = files["application/pdf"]
			jatsUrl = files["application/xml"]
			if jatsUrl == "" {
				jatsUrl = files["application/vnd.jats+xml"]
			}

			var data commonmeta.Data
			if slices.Contains([]string{"application/vnd.commonmeta+json", "application/json", "application/vnd.datacite.datacite+json", "application/vnd.citationstyles.csl+json", "application/vnd.schemaorg.ld+json", "application/vnd.crossref.unixsd+xml"}, contentType) {
				data, err = WriteWorkToCommonmeta(work)
				if err != nil {
					log.Println("error:", err)
				}
			}
			switch contentType {
			case "application/vnd.commonmeta+json", "application/json":
				// return metadata in Commonmeta format
				return c.JSON(http.StatusOK, work)
			case "application/vnd.crossref.unixsd+xml":
				// return metadata in Crossref UNIXREF xml format
				out, err := crossrefxml.Convert(data)
				if err != nil {
					log.Println("error:", err)
				}
				return c.XML(http.StatusOK, out)
			case "application/vnd.datacite.datacite+json":
				// return metadata in Datacite format
				out, err := datacite.Convert(data)
				if err != nil {
					log.Println("error:", err)
				}
				return c.JSON(http.StatusOK, out)
			case "application/vnd.citationstyles.csl+json":
				// return metadata in CSL format
				out, err := csl.Convert(data)
				if err != nil {
					log.Println("error:", err)
				}
				return c.JSON(http.StatusOK, out)
			case "application/vnd.schemaorg.ld+json":
				// return metadata in SchemaOrg format
				out, err := schemaorg.Convert(data)
				if err != nil {
					log.Println("error:", err)
				}
				return c.JSON(http.StatusOK, out)
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

// GetWorkFromCommonmeta returns a Work struct from a commonmeta.Data struct
func GetWorkFromCommonmeta(data commonmeta.Data) *Work {
	work := &Work{
		Pid:               data.ID,
		Type:              data.Type,
		AdditionalType:    data.AdditionalType,
		ArchiveLocations:  marshalSlice(data.ArchiveLocations),
		Container:         marshalStruct(data.Container),
		Contributors:      marshalSlice(data.Contributors),
		Date:              marshalStruct(data.Date),
		Descriptions:      marshalSlice(data.Descriptions),
		Files:             marshalSlice(data.Files),
		FundingReferences: marshalSlice(data.FundingReferences),
		GeoLocations:      marshalSlice(data.GeoLocations),
		Identifiers:       marshalSlice(data.Identifiers),
		Language:          data.Language,
		License:           marshalStruct(data.License),
		Provider:          data.Provider,
		Publisher:         marshalStruct(data.Publisher),
		References:        marshalSlice(data.References),
		Relations:         marshalSlice(data.Relations),
		Subjects:          marshalSlice(data.Subjects),
		Titles:            marshalSlice(data.Titles),
		Url:               data.URL,
		Version:           data.Version,
		Created:           types.NowDateTime(),
		Updated:           types.NowDateTime(),
	}
	return work
}

// WriteWorkToCommonmeta returns a commonmeta.Data struct from a Work struct
func WriteWorkToCommonmeta(w *Work) (commonmeta.Data, error) {
	var data commonmeta.Data
	var err error

	data.ID = w.Pid
	data.Type = w.Type
	data.AdditionalType = w.AdditionalType
	data.ArchiveLocations = []string{}
	err = json.Unmarshal(w.Container, &data.Container)
	err = json.Unmarshal(w.Contributors, &data.Contributors)
	err = json.Unmarshal(w.Date, &data.Date)
	err = json.Unmarshal(w.Descriptions, &data.Descriptions)
	err = json.Unmarshal(w.Files, &data.Files)
	err = json.Unmarshal(w.FundingReferences, &data.FundingReferences)
	err = json.Unmarshal(w.GeoLocations, &data.GeoLocations)
	err = json.Unmarshal(w.Identifiers, &data.Identifiers)
	data.Language = w.Language
	err = json.Unmarshal(w.License, &data.License)
	data.Provider = w.Provider
	err = json.Unmarshal(w.Publisher, &data.Publisher)
	err = json.Unmarshal(w.References, &data.References)
	err = json.Unmarshal(w.Relations, &data.Relations)
	err = json.Unmarshal(w.Subjects, &data.Subjects)
	err = json.Unmarshal(w.Titles, &data.Titles)
	data.URL = w.Url
	data.Version = w.Version
	if err != nil {
		return data, err
	}
	return data, nil
}

func marshalSlice(data interface{}) types.JsonRaw {
	b, err := json.Marshal(data)
	if err != nil {
		log.Println("error:", err)
		return types.JsonRaw("[]")
	}
	return types.JsonRaw(b)
}

func marshalStruct(data interface{}) types.JsonRaw {
	b, err := json.Marshal(data)
	if err != nil {
		log.Println("error:", err)
		return types.JsonRaw("{}")
	}
	return types.JsonRaw(b)
}

func unmarshal(data types.JsonRaw) interface{} {
	var v interface{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		log.Println("error:", err)
		return nil
	}
	return v
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

// find multiple works by the citations of a pid.
func FindWorksByCitation(dao *daos.Dao, pid string) ([]*Work, error) {
	works := []*Work{}

	err := WorkQuery(dao).
		AndWhere(dbx.In("references.0.id", pid)).
		All(&works)

	if err != nil {
		return nil, err
	}

	return works, nil
}

// find DOI registration agency from works collection
func FindDoiRegistrationAgency(dao *daos.Dao, doi string) (string, error) {
	prefix, ok := doiutils.ValidatePrefix(doi)
	if !ok {
		return "", fmt.Errorf("invalid DOI")
	}
	work := &Work{}
	err := WorkQuery(dao).
		AndWhere(dbx.NewExp("pid LIKE {:substr}", dbx.Params{
			"substr": "https://doi.org/" + prefix + "%",
		})).
		Limit(1).
		One(work)

	if err == sql.ErrNoRows {
		// if not found in works collection, look up DOI registration agency from doi.org service

		ra, ok := doiutils.GetDOIRA(prefix)
		log.Printf("RA: %s %v \n", ra, ok)
		if !ok {
			return "", nil
		}
		return ra, nil
	} else if err != nil {
		return "", err
	}

	return work.Provider, nil
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

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

type Content struct {
	PID            string     `json:"pid"`
	Type           string     `json:"type"`
	Attributes     Attributes `json:"attributes"`
	Abstract       string     `json:"abstract"`
	Archive        []string   `json:"archive"`
	Author         []Author   `json:"author"`
	Blog           Blog       `json:"blog"`
	ContainerTitle []string   `json:"container-title"`
	DOI            string     `json:"doi"`
	Files          []struct{} `json:"files"`
	Funder         []struct {
		DOI   string   `json:"DOI"`
		Name  string   `json:"name"`
		Award []string `json:"award"`
	} `json:"funder"`
	GUID   string `json:"guid"`
	Issue  string `json:"issue"`
	Issued struct {
		DateAsParts []DateParts `json:"date-parts"`
		DateTime    string      `json:"date-time"`
	} `json:"issued"`
	Created struct {
		DateAsParts []DateParts `json:"date-parts"`
		DateTime    string      `json:"date-time"`
	} `json:"created"`
	ISSN     []string `json:"ISSN"`
	ISBNType []struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	} `json:"isbn-type"`
	Language string `json:"language"`
	License  []struct {
		Url            string `json:"URL"`
		ContentVersion string `json:"content-version"`
	} `json:"license"`
	Link []struct {
		ContentType string `json:"content-type"`
		Url         string `json:"url"`
	} `json:"link"`
	Page        string `json:"page"`
	PublishedAt string `json:"published_at"`
	Publisher   string `json:"publisher"`
	Reference   []struct {
		Key          string `json:"key"`
		DOI          string `json:"DOI"`
		ArticleTitle string `json:"article-title"`
		Year         string `json:"year"`
		Unstructured string `json:"unstructured"`
	} `json:"reference"`
	Relation struct {
		IsNewVersionOf      []CrossrefRelation `json:"is-new-version-of"`
		IsPreviousVersionOf []CrossrefRelation `json:"is-previous-version-of"`
		IsVersionOf         []CrossrefRelation `json:"is-version-of"`
		HasVersion          []CrossrefRelation `json:"has-version"`
		IsPartOf            []CrossrefRelation `json:"is-part-of"`
		HasPart             []CrossrefRelation `json:"has-part"`
		IsVariantFormOf     []CrossrefRelation `json:"is-variant-form-of"`
		IsOriginalFormOf    []CrossrefRelation `json:"is-original-form-of"`
		IsIdenticalTo       []CrossrefRelation `json:"is-identical-to"`
		IsTranslationOf     []CrossrefRelation `json:"is-translation-of"`
		IsReviewedBy        []CrossrefRelation `json:"is-reviewed-by"`
		Reviews             []CrossrefRelation `json:"reviews"`
		HasReview           []CrossrefRelation `json:"has-review"`
		IsPreprintOf        []CrossrefRelation `json:"is-preprint-of"`
		HasPreprint         []CrossrefRelation `json:"has-preprint"`
		IsSupplementTo      []CrossrefRelation `json:"is-supplement-to"`
		IsSupplementedBy    []CrossrefRelation `json:"is-supplemented-by"`
	}
	Resource struct {
		Primary struct {
			ContentType string `json:"content_type"`
			URL         string `json:"url"`
		} `json:"primary"`
	} `json:"resource"`
	Subject   []string `json:"subject"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	Title     []string `json:"title"`
	UpdatedAt string   `json:"updated_at"`
	Url       string   `json:"url"`
	Version   string   `json:"version"`
	Via       string   `json:"via"`
	Volume    string   `json:"volume"`
}

type Attributes struct {
	DOI             string    `json:"doi"`
	Prefix          string    `json:"prefix"`
	Suffix          string    `json:"suffix"`
	Creators        []Creator `json:"creators"`
	Publisher       string    `json:"publisher"`
	Container       Container `json:"container"`
	PublicationYear int       `json:"publicationYear"`
	Titles          []Title   `json:"titles"`
	Url             string    `json:"url"`
}

type Author struct {
	Given       string `json:"given"`
	Family      string `json:"family"`
	Name        string `json:"name"`
	ORCID       string `json:"ORCID"`
	Sequence    string `json:"sequence"`
	Affiliation []struct {
		ROR  string `json:"ror"`
		Name string `json:"name"`
	} `json:"affiliation"`
}

type Blog struct {
	ISSN        string `json:"issn"`
	License     string `json:"license"`
	Title       string `json:"title"`
	HomePageUrl string `json:"home_page_url"`
}

type Container struct {
	Identifier     string `json:"identifier,omitempty"`
	IdentifierType string `json:"identifierType,omitempty"`
	Type           string `json:"type,omitempty"`
	Title          string `json:"title,omitempty"`
	FirstPage      string `json:"firstPage,omitempty"`
	LastPage       string `json:"lastPage,omitempty"`
	Volume         string `json:"volume,omitempty"`
	Issue          string `json:"issue,omitempty"`
}

type Contributor struct {
	ID           string `json:"id,omitempty"`
	Type         string `json:"type"`
	Name         string `json:"name,omitempty"`
	GivenName    string `json:"givenName,omitempty"`
	FamilyName   string `json:"familyName,omitempty"`
	Affiliations []struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"affiliations,omitempty"`
	ContributorRoles []string `json:"contributorRoles"`
}

type Creator struct {
	Type           string `json:"type"`
	Identifier     string `json:"identifier"`
	IdentifierType string `json:"identifierType"`
	Name           string `json:"name"`
}

type CrossrefRelation struct {
	ID     string `json:"id"`
	IDType string `json:"id-type"`
}

type File struct {
	Bucket   string `json:"bucket,omitempty"`
	Key      string `json:"key,omitempty"`
	Checksum string `json:"checksum,omitempty"`
	Url      string `json:"url"`
	Size     int    `json:"size,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

type FundingReference struct {
	FunderIdentifier     string `json:"funderIdentifier,omitempty"`
	FunderIdentifierType string `json:"funderIdentifierType,omitempty"`
	FunderName           string `json:"funderName"`
	AwardNumber          string `json:"awardNumber,omitempty"`
	AwardURI             string `json:"award_uri,omitempty"`
}

type License struct {
	ID  string `json:"id,omitempty"`
	Url string `json:"url,omitempty"`
}

type Publisher struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type Reference struct {
	Key             string `json:"key"`
	Doi             string `json:"doi,omitempty"`
	Title           string `json:"title,omitempty"`
	PublicationYear string `json:"publicationYear,omitempty"`
	Unstructured    string `json:"unstructured,omitempty"`
}

type Relation struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type Subject struct {
	Subject string `json:"subject"`
}

type Title struct {
	Title    string `json:"title"`
	Type     string `json:"type"`
	Language string `json:"language"`
}

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

	// database fields
	Created types.DateTime `db:"created" json:"created"`
	Updated types.DateTime `db:"updated" json:"updated"`
}

type DateParts []int

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
		Doi             string `json:"doi,omitempty"`
		Url             string `json:"url,omitempty"`
		Key             string `json:"key"`
		PublicationYear string `json:"publicationYear,omitempty"`
		Title           string `json:"title,omitempty"`
		Unstructured    string `json:"unstructured,omitempty"`
	}

	type Relation struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}

	type Subject struct {
		Subject string `json:"subject"`
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

			// check if the pid is a valid URL
			u, err := url.ParseRequestURI(pid)
			if err != nil {
				return err
			}

			// extract optional content type from the URL path
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
					ra, err := FindDoiRegistrationAgency(app.Dao(), pid)
					if err != nil {
						return err
					}
					if isDoi && ra == "Crossref" {
						log.Printf("%s not found, looking up metadata...", pid)
						content, err := GetCrossref(pid)
						if err != nil {
							return err
						}
						work, err := ReadCrossref(content)
						if err != nil {
							return err
						}
						return c.JSON(http.StatusOK, work)
					}
					// don't redirect non-DOI URLs
					return c.JSON(http.StatusNotFound, map[string]string{"error": "Not found"})
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

func GetCrossref(pid string) (Content, error) {
	// the envelope for the JSON response from the Crossref API
	type Response struct {
		Status         string  `json:"status"`
		MessageType    string  `json:"message-type"`
		MessageVersion string  `json:"message-version"`
		Message        Content `json:"message"`
	}

	var response Response
	doi, err := DOIFromUrl(pid)
	if err != nil {
		return response.Message, err
	}
	url := "https://api.crossref.org/works/" + doi
	client := http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	if err != nil {
		return response.Message, err
	}
	if resp.StatusCode >= 400 {
		return response.Message, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response.Message, err
	}
	log.Println(string(body))
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("error:", err)
	}
	return response.Message, err
}

// read Crossref JSON response and return work struct in Commonmeta format
func ReadCrossref(content Content) (*Work, error) {

	// source: http://api.crossref.org/types
	CRToCMMappings := map[string]string{
		"book-chapter":        "BookChapter",
		"book-part":           "BookPart",
		"book-section":        "BookSection",
		"book-series":         "BookSeries",
		"book-set":            "BookSet",
		"book-track":          "BookTrack",
		"book":                "Book",
		"component":           "Component",
		"database":            "Database",
		"dataset":             "Dataset",
		"dissertation":        "Dissertation",
		"edited-book":         "Book",
		"grant":               "Grant",
		"journal-article":     "JournalArticle",
		"journal-issue":       "JournalIssue",
		"journal-volume":      "JournalVolume",
		"journal":             "Journal",
		"monograph":           "Book",
		"other":               "Other",
		"peer-review":         "PeerReview",
		"posted-content":      "Article",
		"proceedings-article": "ProceedingsArticle",
		"proceedings-series":  "ProceedingsSeries",
		"proceedings":         "Proceedings",
		"reference-book":      "Book",
		"reference-entry":     "Entry",
		"report-component":    "ReportComponent",
		"report-series":       "ReportSeries",
		"report":              "Report",
		"standard":            "Standard",
	}

	CrossrefContainerTypes := map[string]string{
		"book-chapter":        "book",
		"dataset":             "database",
		"journal-article":     "journal",
		"journal-issue":       "journal",
		"monograph":           "book-series",
		"proceedings-article": "proceedings",
		"posted-content":      "periodical",
	}

	CRToCMContainerTranslations := map[string]string{
		"book":        "Book",
		"book-series": "BookSeries",
		"database":    "DataRepository",
		"journal":     "Journal",
		"proceedings": "Proceedings",
		"periodical":  "Periodical",
	}

	pid := DOIAsUrl(content.DOI)
	url := content.Resource.Primary.URL
	provider := "Crossref"
	Type := CRToCMMappings[content.Type]

	var contributors = func() types.JsonRaw {
		contributors := make([]Contributor, 0)
		for _, v := range content.Author {
			if v.Name != "" || v.Given != "" || v.Family != "" {
				affiliations := make([]struct {
					ID   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				}, len(v.Affiliation))
				for i, a := range v.Affiliation {
					affiliations[i] = struct {
						ID   string `json:"id,omitempty"`
						Name string `json:"name,omitempty"`
					}{ID: a.ROR, Name: a.Name}
				}
				contributors = append(contributors, Contributor{
					ID:               v.ORCID,
					Type:             "Person",
					GivenName:        v.Given,
					FamilyName:       v.Family,
					Name:             v.Name,
					Affiliations:     affiliations,
					ContributorRoles: []string{"Author"},
				})
			}
		}
		b, err := json.Marshal(contributors)
		if err != nil {
			return types.JsonRaw("[]")
		}
		return types.JsonRaw(b)
	}
	var publisher = func() types.JsonRaw {
		return types.JsonRaw(fmt.Sprintf(`{"name": "%s"}`, content.Publisher))
	}
	var date = func() types.JsonRaw {
		if content.Issued.DateTime != "" {
			return types.JsonRaw(fmt.Sprintf(`{"published": "%s"}`, content.Issued.DateTime))
		} else if len(content.Issued.DateAsParts) > 1 {
			published := GetDateFromDateParts(content.Issued.DateAsParts)
			return types.JsonRaw(fmt.Sprintf(`{"published": "%s"}`, published))
		} else if content.Created.DateTime != "" {
			return types.JsonRaw(fmt.Sprintf(`{"created": "%s"}`, content.Created.DateTime))
		} else if len(content.Created.DateAsParts) > 1 {
			created := GetDateFromDateParts(content.Created.DateAsParts)
			return types.JsonRaw(fmt.Sprintf(`{"created": "%s"}`, created))
		} else {
			return types.JsonRaw("{}")
		}
	}
	var titles = func() types.JsonRaw {
		if len(content.Title) > 0 {
			return types.JsonRaw(fmt.Sprintf(`[{"title": "%s"}]`, content.Title[0]))
		}
		return types.JsonRaw("[]")
	}
	var descriptions = func() types.JsonRaw {
		if content.Abstract != "" {
			return types.JsonRaw(fmt.Sprintf(`[{"description": "%s", "descriptionType": "Abstract"}]`, content.Abstract))
		}
		return types.JsonRaw("[]")
	}
	var container = func() types.JsonRaw {
		containerType := CrossrefContainerTypes[content.Type]
		containerType = CRToCMContainerTranslations[containerType]
		var identifier, identifierType string
		if content.ISSN != nil {
			identifier = IssnAsUrl(content.ISSN[0])
			identifierType = "ISSN"
		}
		if len(content.ISBNType) > 0 {
			identifier = content.ISBNType[0].Value
			identifierType = "ISBN"
		}
		var containerTitle string
		if len(content.ContainerTitle) > 0 {
			containerTitle = content.ContainerTitle[0]
		}
		var lastPage string
		pages := strings.Split(content.Page, "-")
		firstPage := pages[0]
		if len(pages) > 1 {
			lastPage = pages[1]
		}
		container := Container{
			Identifier:     identifier,
			IdentifierType: identifierType,
			Type:           containerType,
			Title:          containerTitle,
			Volume:         content.Volume,
			Issue:          content.Issue,
			FirstPage:      firstPage,
			LastPage:       lastPage,
		}
		b, err := json.Marshal(container)
		if err != nil {
			return types.JsonRaw("{}")
		}
		return types.JsonRaw(b)
	}
	var subjects = func() types.JsonRaw {
		if len(content.Subject) > 0 {
			subjects := make([]Subject, len(content.Subject))
			for i, v := range content.Subject {
				subjects[i] = Subject{
					Subject: v,
				}
			}
			b, err := json.Marshal(subjects)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var references = func() types.JsonRaw {
		if len(content.Reference) > 0 {
			references := make([]Reference, len(content.Reference))
			for i, v := range content.Reference {
				references[i] = Reference{
					Key:             v.Key,
					Doi:             DOIAsUrl(v.DOI),
					Title:           v.ArticleTitle,
					PublicationYear: v.Year,
					Unstructured:    v.Unstructured,
				}
			}
			b, err := json.Marshal(references)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var relations = func() types.JsonRaw {
		relations := make([]Relation, 0)
		fields := reflect.VisibleFields(reflect.TypeOf(content.Relation))
		for _, field := range fields {
			// relation types to include
			relationTypes := []string{"IsPartOf", "HasPart", "IsVariantFormOf", "IsOriginalFormOf", "IsIdenticalTo", "IsTranslationOf", "IsReviewedBy", "Reviews", "HasReview", "IsPreprintOf", "HasPreprint", "IsSupplementTo", "IsSupplementedBy"}
			if slices.Contains(relationTypes, field.Name) {
				relationByType := reflect.ValueOf(content.Relation).FieldByName(field.Name)
				for _, v := range relationByType.Interface().([]CrossrefRelation) {
					relations = append(relations, Relation{
						ID:   DOIAsUrl(v.ID),
						Type: field.Name,
					})
				}
			}
		}
		b, err := json.Marshal(relations)
		if err != nil {
			return types.JsonRaw("[]")
		}
		return types.JsonRaw(b)
	}
	var fundingReferences = func() types.JsonRaw {
		if content.Funder != nil {
			fundingReferences := make([]FundingReference, 0)
			for _, v := range content.Funder {
				funderIdentifier := DOIAsUrl(v.DOI)
				var funderIdentifierType string
				if strings.HasPrefix(v.DOI, "10.13039") {
					funderIdentifierType = "Crossref Funder ID"
				}
				if len(v.Award) > 0 {
					for _, award := range v.Award {
						fundingReferences = append(fundingReferences, FundingReference{
							FunderIdentifier:     funderIdentifier,
							FunderIdentifierType: funderIdentifierType,
							FunderName:           v.Name,
							AwardNumber:          award,
						})
					}
				} else {
					fundingReferences = append(fundingReferences, FundingReference{
						FunderIdentifier:     funderIdentifier,
						FunderIdentifierType: funderIdentifierType,
						FunderName:           v.Name,
					})
				}
			}
			fundingReferences = RemoveDuplicates(fundingReferences)
			b, err := json.Marshal(fundingReferences)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var license = func() types.JsonRaw {
		if content.License != nil && len(content.License) > 0 {
			url, _ := NormalizeCCUrl(content.License[0].Url)
			id := UrlToSPDX(url)
			if id == "" {
				log.Printf("License URL %s not found in SPDX", url)
			}
			license := License{
				ID:  id,
				Url: url,
			}
			b, err := json.Marshal(license)
			if err != nil {
				return types.JsonRaw("{}")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("{}")
	}
	var files = func() types.JsonRaw {
		if len(content.Link) > 0 {
			files := make([]File, 0)
			for _, v := range content.Link {
				if v.ContentType != "unspecified" {
					files = append(files, File{
						Url:      v.Url,
						MimeType: v.ContentType,
					})
				}
			}
			files = RemoveDuplicates(files)
			b, err := json.Marshal(files)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var archiveLocations = func() types.JsonRaw {
		if len(content.Archive) > 0 {
			archiveLocations := make([]string, len(content.Archive))
			copy(archiveLocations, content.Archive)
			b, err := json.Marshal(archiveLocations)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}

	work := &Work{
		Pid:                  pid,
		Type:                 Type,
		AdditionalType:       "",
		Url:                  url,
		Contributors:         contributors(),
		Publisher:            publisher(),
		Date:                 date(),
		Titles:               titles(),
		Container:            container(),
		Subjects:             subjects(),
		Sizes:                types.JsonRaw(nil),
		Formats:              types.JsonRaw(nil),
		Language:             content.Language,
		License:              license(),
		Version:              content.Version,
		References:           references(),
		Relations:            relations(),
		FundingReferences:    fundingReferences(),
		Descriptions:         descriptions(),
		GeoLocations:         types.JsonRaw(nil),
		Provider:             provider,
		AlternateIdentifiers: types.JsonRaw(nil),
		Files:                files(),
		ArchiveLocations:     archiveLocations(),
		Created:              types.NowDateTime(),
		Updated:              types.NowDateTime(),
	}
	return work, nil
}

// extract DOI from URL
func DOIFromUrl(str string) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return str, nil
	}
	if u.Host != "doi.org" || !strings.HasPrefix(u.Path, "/10.") {
		return "", nil
	}
	return strings.TrimLeft(u.Path, "/"), nil
}

func DOIAsUrl(str string) string {
	if str == "" {
		return ""
	}
	return "https://doi.org/" + str
}

func GetDateFromDateParts(dateAsParts []DateParts) string {
	switch len(dateAsParts) {
	case 0:
		return ""
	case 1:
		year := dateAsParts[0][0]
		if year == 0 {
			return ""
		}
		return GetDateFromParts(year)
	case 2:
		year, month := dateAsParts[0][0], dateAsParts[0][1]
		return GetDateFromParts(year, month)
	case 3:
		year, month, day := dateAsParts[0][0], dateAsParts[0][1], dateAsParts[0][2]
		return GetDateFromParts(year, month, day)
	}
	return ""
}

func GetDateFromParts(parts ...int) string {
	var arr []string
	switch len(parts) {
	case 0:
		return ""
	case 1:
		year := fmt.Sprintf("%04d", parts[0])
		arr = []string{year}
	case 2:
		year, month := fmt.Sprintf("%04d", parts[0]), fmt.Sprintf("%02d", parts[1])
		arr = []string{year, month}
	case 3:
		year, month, day := fmt.Sprintf("%04d", parts[0]), fmt.Sprintf("%02d", parts[1]), fmt.Sprintf("%02d", parts[2])
		arr = []string{year, month, day}
	}
	return strings.Join(arr, "-")
}

// ISSN as URL
func IssnAsUrl(issn string) string {
	if issn == "" {
		return ""
	}
	return fmt.Sprintf("https://portal.issn.org/resource/ISSN/%s", issn)
}

// https://stackoverflow.com/questions/66643946/how-to-remove-duplicates-strings-or-int-from-slice-in-go/76948712#76948712
func RemoveDuplicates[T comparable](s []T) []T {
	alreadySeen := make(map[T]struct{}, len(s))
	return slices.DeleteFunc(s, func(val T) bool {
		_, duplicate := alreadySeen[val]
		alreadySeen[val] = struct{}{}
		return duplicate
	})
}

func NormalizeCCUrl(url string) (string, error) {
	NormalizedLicenses := map[string]string{
		"https://creativecommons.org/licenses/by/1.0":          "https://creativecommons.org/licenses/by/1.0/legalcode",
		"https://creativecommons.org/licenses/by/2.0":          "https://creativecommons.org/licenses/by/2.0/legalcode",
		"https://creativecommons.org/licenses/by/2.5":          "https://creativecommons.org/licenses/by/2.5/legalcode",
		"https://creativecommons.org/licenses/by/3.0":          "https://creativecommons.org/licenses/by/3.0/legalcode",
		"https://creativecommons.org/licenses/by/3.0/us":       "https://creativecommons.org/licenses/by/3.0/legalcode",
		"https://creativecommons.org/licenses/by/4.0":          "https://creativecommons.org/licenses/by/4.0/legalcode",
		"https://creativecommons.org/licenses/by-nc/1.0":       "https://creativecommons.org/licenses/by-nc/1.0/legalcode",
		"https://creativecommons.org/licenses/by-nc/2.0":       "https://creativecommons.org/licenses/by-nc/2.0/legalcode",
		"https://creativecommons.org/licenses/by-nc/2.5":       "https://creativecommons.org/licenses/by-nc/2.5/legalcode",
		"https://creativecommons.org/licenses/by-nc/3.0":       "https://creativecommons.org/licenses/by-nc/3.0/legalcode",
		"https://creativecommons.org/licenses/by-nc/4.0":       "https://creativecommons.org/licenses/by-nc/4.0/legalcode",
		"https://creativecommons.org/licenses/by-nd-nc/1.0":    "https://creativecommons.org/licenses/by-nd-nc/1.0/legalcode",
		"https://creativecommons.org/licenses/by-nd-nc/2.0":    "https://creativecommons.org/licenses/by-nd-nc/2.0/legalcode",
		"https://creativecommons.org/licenses/by-nd-nc/2.5":    "https://creativecommons.org/licenses/by-nd-nc/2.5/legalcode",
		"https://creativecommons.org/licenses/by-nd-nc/3.0":    "https://creativecommons.org/licenses/by-nd-nc/3.0/legalcode",
		"https://creativecommons.org/licenses/by-nd-nc/4.0":    "https://creativecommons.org/licenses/by-nd-nc/4.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-sa/1.0":    "https://creativecommons.org/licenses/by-nc-sa/1.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-sa/2.0":    "https://creativecommons.org/licenses/by-nc-sa/2.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-sa/2.5":    "https://creativecommons.org/licenses/by-nc-sa/2.5/legalcode",
		"https://creativecommons.org/licenses/by-nc-sa/3.0":    "https://creativecommons.org/licenses/by-nc-sa/3.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-sa/3.0/us": "https://creativecommons.org/licenses/by-nc-sa/3.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-sa/4.0":    "https://creativecommons.org/licenses/by-nc-sa/4.0/legalcode",
		"https://creativecommons.org/licenses/by-nd/1.0":       "https://creativecommons.org/licenses/by-nd/1.0/legalcode",
		"https://creativecommons.org/licenses/by-nd/2.0":       "https://creativecommons.org/licenses/by-nd/2.0/legalcode",
		"https://creativecommons.org/licenses/by-nd/2.5":       "https://creativecommons.org/licenses/by-nd/2.5/legalcode",
		"https://creativecommons.org/licenses/by-nd/3.0":       "https://creativecommons.org/licenses/by-nd/3.0/legalcode",
		"https://creativecommons.org/licenses/by-nd/4.0":       "https://creativecommons.org/licenses/by-nd/2.0/legalcode",
		"https://creativecommons.org/licenses/by-sa/1.0":       "https://creativecommons.org/licenses/by-sa/1.0/legalcode",
		"https://creativecommons.org/licenses/by-sa/2.0":       "https://creativecommons.org/licenses/by-sa/2.0/legalcode",
		"https://creativecommons.org/licenses/by-sa/2.5":       "https://creativecommons.org/licenses/by-sa/2.5/legalcode",
		"https://creativecommons.org/licenses/by-sa/3.0":       "https://creativecommons.org/licenses/by-sa/3.0/legalcode",
		"https://creativecommons.org/licenses/by-sa/4.0":       "https://creativecommons.org/licenses/by-sa/4.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-nd/1.0":    "https://creativecommons.org/licenses/by-nc-nd/1.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-nd/2.0":    "https://creativecommons.org/licenses/by-nc-nd/2.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-nd/2.5":    "https://creativecommons.org/licenses/by-nc-nd/2.5/legalcode",
		"https://creativecommons.org/licenses/by-nc-nd/3.0":    "https://creativecommons.org/licenses/by-nc-nd/3.0/legalcode",
		"https://creativecommons.org/licenses/by-nc-nd/4.0":    "https://creativecommons.org/licenses/by-nc-nd/4.0/legalcode",
		"https://creativecommons.org/licenses/publicdomain":    "https://creativecommons.org/licenses/publicdomain/",
		"https://creativecommons.org/publicdomain/zero/1.0":    "https://creativecommons.org/publicdomain/zero/1.0/legalcode",
	}

	if url == "" {
		return "", nil
	}
	var err error
	url, err = NormalizeUrl(url, true, true)
	if err != nil {
		return "", err
	}
	normalizedUrl, ok := NormalizedLicenses[url]
	if !ok {
		return url, fmt.Errorf("License URL not found")
	}
	return normalizedUrl, nil
}

func UrlToSPDX(url string) string {
	// appreviated list from https://spdx.org/licenses/
	SPDXLicenses := map[string]string{
		"https://creativecommons.org/licenses/by/3.0/legalcode": "CC-BY-3.0",
		"https://creativecommons.org/licenses/by/4.0/legalcode": "CC-BY-4.0",
	}
	id := SPDXLicenses[url]
	return id
}

// Normalize URL
func NormalizeUrl(str string, secure bool, lower bool) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	if u.Path[len(u.Path)-1] == '/' {
		u.Path = u.Path[:len(u.Path)-1]
	}
	if secure && u.Scheme == "http" {
		u.Scheme = "https"
	}
	if lower {
		return strings.ToLower(u.String()), nil
	}
	return u.String(), nil
}

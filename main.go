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
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/sym01/htmlsanitizer"
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
	DOI                  string `json:"doi"`
	Prefix               string `json:"prefix"`
	Suffix               string `json:"suffix"`
	AlternateIdentifiers []struct {
		Identifier     string `json:"identifier"`
		IdentifierType string `json:"identifierType"`
	} `json:"alternateIdentifiers"`
	Creators []struct {
		Name            string `json:"name"`
		GivenName       string `json:"givenName"`
		FamilyName      string `json:"familyName"`
		NameType        string `json:"nameType"`
		NameIdentifiers []struct {
			SchemeURI            string `json:"schemeUri"`
			NameIdentifier       string `json:"nameIdentifier"`
			NameIdentifierScheme string `json:"nameIdentifierScheme"`
		} `json:"nameIdentifiers"`
		Affiliation []string `json:"affiliation"`
	} `json:"creators"`
	Publisher string `json:"publisher"`
	Container struct {
		Type           string `json:"type"`
		Identifier     string `json:"identifier"`
		IdentifierType string `json:"identifierType"`
		Title          string `json:"title"`
		Volume         string `json:"volume"`
		Issue          string `json:"issue"`
		FirstPage      string `json:"firstPage"`
		LastPage       string `json:"lastPage"`
	} `json:"container"`
	PublicationYear int `json:"publicationYear"`
	Titles          []struct {
		Title     string `json:"title"`
		TitleType string `json:"titleType"`
		Lang      string `json:"lang"`
	} `json:"titles"`
	Url      string `json:"url"`
	Subjects []struct {
		Subject string `json:"subject"`
	} `json:"subjects"`
	Contributors []struct {
		Name            string `json:"name"`
		GivenName       string `json:"givenName"`
		FamilyName      string `json:"familyName"`
		NameType        string `json:"nameType"`
		NameIdentifiers []struct {
			SchemeURI            string `json:"schemeUri"`
			NameIdentifier       string `json:"nameIdentifier"`
			NameIdentifierScheme string `json:"nameIdentifierScheme"`
		} `json:"nameIdentifiers"`
		Affiliation     []string `json:"affiliation"`
		ContributorType string   `json:"contributorType"`
	} `json:"contributors"`
	Dates []struct {
		Date            string `json:"date"`
		DateType        string `json:"dateType"`
		DateInformation string `json:"dateInformation"`
	} `json:"dates"`
	Language string `json:"language"`
	Types    struct {
		ResourceTypeGeneral string `json:"resourceTypeGeneral"`
		ResourceType        string `json:"resourceType"`
	} `json:"types"`
	RelatedIdentifiers []struct {
		RelatedIdentifier     string `json:"relatedIdentifier"`
		RelatedIdentifierType string `json:"relatedIdentifierType"`
		RelationType          string `json:"relationType"`
	} `json:"relatedIdentifiers"`
	Sizes      []string `json:"sizes"`
	Formats    []string `json:"formats"`
	Version    string   `json:"version"`
	RightsList []struct {
		Rights                 string `json:"rights"`
		RightsURI              string `json:"rightsUri"`
		SchemeURI              string `json:"schemeUri"`
		RightsIdentifier       string `json:"rightsIdentifier"`
		RightsIdentifierScheme string `json:"rightsIdentifierScheme"`
	}
	Descriptions []struct {
		Description     string `json:"description"`
		DescriptionType string `json:"descriptionType"`
		Lang            string `json:"lang"`
	} `json:"descriptions"`
	GeoLocations []struct {
		GeoLocationPoint struct {
			PointLongitude float64 `json:"pointLongitude"`
			PointLatitude  float64 `json:"pointLatitude"`
		} `json:"geoLocationPoint"`
		GeoLocationBox struct {
			WestBoundLongitude float64 `json:"westBoundLongitude"`
			EastBoundLongitude float64 `json:"eastBoundLongitude"`
			SouthBoundLatitude float64 `json:"southBoundLatitude"`
			NorthBoundLatitude float64 `json:"northBoundLatitude"`
		} `json:"geoLocationBox"`
		GeoLocationPlace string `json:"geoLocationPlace"`
	} `json:"geoLocations"`
	FundingReferences []struct {
		FunderName           string `json:"funderName"`
		FunderIdentifier     string `json:"funderIdentifier"`
		FunderIdentifierType string `json:"funderIdentifierType"`
		AwardNumber          string `json:"awardNumber"`
		AwardURI             string `json:"awardUri"`
	} `json:"fundingReferences"`
}

type AlternateIdentifier struct {
	Identifier     string `json:"identifier"`
	IdentifierType string `json:"identifierType"`
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
	NameType        string `json:"nameType"`
	Name            string `json:"name"`
	GivenName       string `json:"givenName"`
	FamilyName      string `json:"familyName"`
	NameIdentifiers []struct {
		SchemeURI            string `json:"schemeUri"`
		NameIdentifier       string `json:"nameIdentifier"`
		NameIdentifierScheme string `json:"nameIdentifierScheme"`
	} `json:"nameIdentifiers"`
	Affiliation []string `json:"affiliation"`
}

type CrossrefRelation struct {
	ID     string `json:"id"`
	IDType string `json:"id-type"`
}

type Date struct {
	Created     string `json:"created,omitempty"`
	Submitted   string `json:"submitted,omitempty"`
	Accepted    string `json:"accepted,omitempty"`
	Published   string `json:"published,omitempty"`
	Updated     string `json:"updated,omitempty"`
	Accessed    string `json:"accessed,omitempty"`
	Available   string `json:"available,omitempty"`
	Copyrighted string `json:"copyrighted,omitempty"`
	Collected   string `json:"collected,omitempty"`
	Valid       string `json:"valid,omitempty"`
	Withdrawn   string `json:"withdrawn,omitempty"`
	Other       string `json:"other,omitempty"`
}

type Description struct {
	Description string `json:"description"`
	Type        string `json:"type,omitempty"`
	Language    string `json:"language,omitempty"`
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

type GeoLocation struct {
	GeoLocationPlace string           `json:"geoLocationPlace,omitempty"`
	GeoLocationPoint GeoLocationPoint `json:"geoLocationPoint,omitempty"`
	GeoLocationBox   struct {
		EastBoundLongitude float64 `json:"eastBoundLongitude"`
		WestBoundLongitude float64 `json:"westBoundLongitude"`
		SouthBoundLatitude float64 `json:"southBoundLatitude"`
		NorthBoundLatitude float64 `json:"northBoundLatitude"`
	} `json:"geoLocationBox,omitempty"`
}

type GeoLocationPoint struct {
	PointLongitude float64 `json:"pointLongitude,omitempty"`
	PointLatitude  float64 `json:"pointLatitude,omitempty"`
}

type GeoLocationBox struct {
	EastBoundLongitude float64 `json:"eastBoundLongitude"`
	WestBoundLongitude float64 `json:"westBoundLongitude"`
	SouthBoundLatitude float64 `json:"southBoundLatitude"`
	NorthBoundLatitude float64 `json:"northBoundLatitude"`
}

type GeoLocationPolygon struct {
	PolygonPoints  []GeoLocationPoint `json:"polygon_points"`
	InPolygonPoint GeoLocationPoint   `json:"in_polygon_point,omitempty"`
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
	Type     string `json:"type,omitempty"`
	Language string `json:"language,omitempty"`
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

			// create a new work record if not found and the pid is a Crossref DOI
			if work == nil {
				ra, err := FindDoiRegistrationAgency(app.Dao(), pid)
				if err != nil {
					return err
				}
				if isDoi && ra == "Crossref" {
					log.Printf("%s not found, looking up metadata with Crossref ...", pid)
					content, err := GetCrossref(pid)
					if err != nil {
						return err
					}
					newWork, err := ReadCrossref(content)
					if err != nil {
						return err
					}
					if err := app.Dao().Save(newWork); err != nil {
						return err
					}

					work, err = FindWorkByPid(app.Dao(), newWork.Pid)
					if err != nil {
						return err
					}
				} else if isDoi && ra == "DataCite" {
					log.Printf("%s not found, looking up metadata with DataCite ...", pid)
					content, err := GetDatacite(pid)
					if err != nil {
						return err
					}
					newWork, err := ReadDatacite(content)
					if err != nil {
						return err
					}
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

			// redirect for content types supported by DOI content negotiation
			contentTypes := []string{"text/html", "application/vnd.commonmeta+json", "application/json", "text/markdown", "application/vnd.jats+xml", "application/xml", "application/pdf"}
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
					if v.Doi != "" {
						refs = append(refs, v.Doi)
					} else if v.Url != "" {
						refs = append(refs, v.Url)
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
				// return metadata in Commonmeta format, handle JSON parsing errors
				_, err := json.Marshal(work)
				if err != nil {
					log.Println("error:", err)
					message := fmt.Sprintf("%+v\n", work)
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error marshalling JSON", "message": message})
				}
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
		// if not found in works collection, look up DOI registration agency from handle service
		ra, err := FindDoiRegistrationAgencyFromHandle(dao, substr)
		log.Printf("DOI registration agency for %s: %s", doi, ra)
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
	prefix, err := PrefixFromUrl(doi)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(fmt.Sprintf("https://doi.org/ra/%s", prefix))
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
	Url := content.Resource.Primary.URL
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
			sanitizedHTML, err := htmlsanitizer.SanitizeString(content.Abstract)
			if err != nil {
				log.Println(err)
				sanitizedHTML = ""
			}
			d := make([]Description, 0)
			d = append(d, Description{
				Description: strings.Trim(sanitizedHTML, "\n"),
				Type:        "Abstract",
			})
			b, err := json.Marshal(d)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
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
			f := make([]FundingReference, 0)
			for _, v := range content.Funder {
				funderIdentifier := DOIAsUrl(v.DOI)
				var funderIdentifierType string
				if strings.HasPrefix(v.DOI, "10.13039") {
					funderIdentifierType = "Crossref Funder ID"
				}
				if len(v.Award) > 0 {
					for _, award := range v.Award {
						f = append(f, FundingReference{
							FunderIdentifier:     funderIdentifier,
							FunderIdentifierType: funderIdentifierType,
							FunderName:           v.Name,
							AwardNumber:          award,
						})
					}
				} else {
					f = append(f, FundingReference{
						FunderIdentifier:     funderIdentifier,
						FunderIdentifierType: funderIdentifierType,
						FunderName:           v.Name,
					})
				}
			}
			f = dedupeSlice(f)
			b, err := json.Marshal(f)
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
			f := make([]File, 0)
			for _, v := range content.Link {
				if v.ContentType != "unspecified" {
					f = append(f, File{
						Url:      v.Url,
						MimeType: v.ContentType,
					})
				}
			}
			f = dedupeSlice(f)
			b, err := json.Marshal(f)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var archiveLocations = func() types.JsonRaw {
		if len(content.Archive) > 0 {
			a := make([]string, len(content.Archive))
			copy(a, content.Archive)
			b, err := json.Marshal(a)
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
		Url:                  Url,
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

func GetDatacite(pid string) (Content, error) {
	// the envelope for the JSON response from the DataCite API
	type Response struct {
		Data Content `json:"data"`
	}

	var response Response
	doi, err := DOIFromUrl(pid)
	if err != nil {
		return response.Data, err
	}
	url := "https://api.datacite.org/dois/" + doi
	client := http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(url)
	if err != nil {
		return response.Data, err
	}
	if resp.StatusCode >= 400 {
		return response.Data, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response.Data, err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("error:", err)
	}
	return response.Data, err
}

// read DataCite JSON response and return work struct in Commonmeta format
func ReadDatacite(content Content) (*Work, error) {

	// source: https://github.com/datacite/schema/blob/master/source/meta/kernel-4/include/datacite-resourceType-v4.xsd
	DCToCMTranslations := map[string]string{
		"Audiovisual":           "Audiovisual",
		"BlogPosting":           "Article",
		"Book":                  "Book",
		"BookChapter":           "BookChapter",
		"Collection":            "Collection",
		"ComputationalNotebook": "ComputationalNotebook",
		"ConferencePaper":       "ProceedingsArticle",
		"ConferenceProceeding":  "Proceedings",
		"DataPaper":             "JournalArticle",
		"Dataset":               "Dataset",
		"Dissertation":          "Dissertation",
		"Event":                 "Event",
		"Image":                 "Image",
		"Instrument":            "Instrument",
		"InteractiveResource":   "InteractiveResource",
		"Journal":               "Journal",
		"JournalArticle":        "JournalArticle",
		"Model":                 "Model",
		"OutputManagementPlan":  "OutputManagementPlan",
		"PeerReview":            "PeerReview",
		"PhysicalObject":        "PhysicalObject",
		"Poster":                "Presentation",
		"Preprint":              "Article",
		"Report":                "Report",
		"Service":               "Service",
		"Software":              "Software",
		"Sound":                 "Sound",
		"Standard":              "Standard",
		"StudyRegistration":     "StudyRegistration",
		"Text":                  "Document",
		"Thesis":                "Dissertation",
		"Workflow":              "Workflow",
		"Other":                 "Other",
	}
	// from commonmeta schema
	CommonmetaContributorRoles := []string{
		"Author",
		"Editor",
		"Chair",
		"Reviewer",
		"ReviewAssistant",
		"StatsReviewer",
		"ReviewerExternal",
		"Reader",
		"Translator",
		"ContactPerson",
		"DataCollector",
		"DataManager",
		"Distributor",
		"HostingInstitution",
		"Producer",
		"ProjectLeader",
		"ProjectManager",
		"ProjectMember",
		"RegistrationAgency",
		"RegistrationAuthority",
		"RelatedPerson",
		"ResearchGroup",
		"RightsHolder",
		"Researcher",
		"Sponsor",
		"WorkPackageLeader",
		"Conceptualization",
		"DataCuration",
		"FormalAnalysis",
		"FundingAcquisition",
		"Investigation",
		"Methodology",
		"ProjectAdministration",
		"Resources",
		"Software",
		"Supervision",
		"Validation",
		"Visualization",
		"WritingOriginalDraft",
		"WritingReviewEditing",
		"Maintainer",
		"Other",
	}

	pid := DOIAsUrl(content.Attributes.DOI)
	Url := content.Attributes.Url
	provider := "DataCite"
	Type := DCToCMTranslations[content.Attributes.Types.ResourceTypeGeneral]
	additionalType := DCToCMTranslations[content.Attributes.Types.ResourceType]
	if additionalType != "" {
		Type = additionalType
		additionalType = ""
	} else {
		additionalType = content.Attributes.Types.ResourceType
	}
	var contributors = func() types.JsonRaw {
		c := make([]Contributor, 0)
		for _, v := range content.Attributes.Creators {
			if v.Name != "" || v.GivenName != "" || v.FamilyName != "" {
				type_ := v.NameType[:len(v.NameType)-2]
				var id string
				if len(v.NameIdentifiers) > 0 {
					ni := v.NameIdentifiers[0]
					id = ni.NameIdentifier
					u, _ := url.Parse(ni.NameIdentifier)
					schemeUri := ni.SchemeURI
					if schemeUri == "" {
						u.Path = ""
						schemeUri = u.String()
					}
					if schemeUri == "https://orcid.org" {
						type_ = "Person"
					} else if schemeUri == "https://ror.org" {
						type_ = "Organization"
					}
				}
				name := v.Name
				if type_ == "" && (v.GivenName != "" || v.FamilyName != "") {
					type_ = "Person"
					name = ""
				} else if type_ == "" {
					type_ = "Organization"
				}
				affiliations := make([]struct {
					ID   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				}, 0)
				for _, a := range v.Affiliation {
					an := struct {
						ID   string `json:"id,omitempty"`
						Name string `json:"name,omitempty"`
					}{ID: "", Name: a}
					affiliations = append(affiliations, an)
				}
				c = append(c, Contributor{
					ID:               id,
					Type:             type_,
					GivenName:        v.GivenName,
					FamilyName:       v.FamilyName,
					Name:             name,
					ContributorRoles: []string{"Author"},
					Affiliations:     affiliations,
				})
			}
		}
		// merge creators and contributors
		for _, v := range content.Attributes.Contributors {
			if v.Name != "" || v.GivenName != "" || v.FamilyName != "" {
				var type_ string
				if len(v.NameType) > 2 {
					type_ = v.NameType[:len(v.NameType)-2]
				}
				var id string
				if len(v.NameIdentifiers) > 0 {
					ni := v.NameIdentifiers[0]
					if ni.NameIdentifierScheme == "ORCID" {
						id = ni.NameIdentifier
						type_ = "Person"
					} else if ni.NameIdentifierScheme == "ROR" {
						id = ni.NameIdentifier
						type_ = "Organization"
					} else {
						id = ni.NameIdentifier
					}
				}
				name := v.Name
				if type_ == "" && (v.GivenName != "" || v.FamilyName != "") {
					type_ = "Person"
					name = ""
				} else if type_ == "" {
					type_ = "Organization"
				}
				affiliations := make([]struct {
					ID   string `json:"id,omitempty"`
					Name string `json:"name,omitempty"`
				}, 0)
				for _, a := range v.Affiliation {
					an := struct {
						ID   string `json:"id,omitempty"`
						Name string `json:"name,omitempty"`
					}{ID: "", Name: a}
					affiliations = append(affiliations, an)
				}
				roles := make([]string, 0)
				if slices.Contains(CommonmetaContributorRoles, v.ContributorType) {
					roles = append(roles, v.ContributorType)
				}
				containsID := slices.ContainsFunc(c, func(e Contributor) bool {
					return e.ID == id
				})
				if containsID {
					log.Printf("Contributor with ID %s already exists", id)
				} else {
					c = append(c, Contributor{
						ID:               id,
						Type:             type_,
						GivenName:        v.GivenName,
						FamilyName:       v.FamilyName,
						Name:             name,
						ContributorRoles: roles,
						Affiliations:     affiliations,
					})
				}
			}
		}
		b, err := json.Marshal(c)
		if err != nil {
			return types.JsonRaw("[]")
		}
		return types.JsonRaw(b)
	}
	var publisher = func() types.JsonRaw {
		return types.JsonRaw(fmt.Sprintf(`{"name": "%s"}`, content.Attributes.Publisher))
	}
	var date = func() types.JsonRaw {
		if len(content.Attributes.Dates) > 0 {
			var date Date
			for _, v := range content.Attributes.Dates {
				if v.DateType == "Accepted" {
					date.Accepted = v.Date
				}
				if v.DateType == "Available" {
					date.Available = v.Date
				}
				if v.DateType == "Collected" {
					date.Collected = v.Date
				}
				if v.DateType == "Copyrighted" {
					date.Copyrighted = v.Date
				}
				if v.DateType == "Created" {
					date.Created = v.Date
				}
				if v.DateType == "Issued" {
					date.Published = v.Date
				}
				if v.DateType == "Submitted" {
					date.Submitted = v.Date
				}
				if v.DateType == "Updated" {
					date.Updated = v.Date
				}
				if v.DateType == "Valid" {
					date.Valid = v.Date
				}
				if v.DateType == "Withdrawn" {
					date.Withdrawn = v.Date
				}
				if v.DateType == "Other" {
					date.Other = v.Date
				}
			}
			b, err := json.Marshal(date)
			if err != nil {
				return types.JsonRaw("{}")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("{}")
	}
	var titles = func() types.JsonRaw {
		if len(content.Attributes.Titles) > 0 {
			titles := make([]Title, len(content.Attributes.Titles))
			for i, v := range content.Attributes.Titles {
				var type_ string
				if slices.Contains([]string{"MainTitle", "Subtitle", "TranslatedTitle"}, v.TitleType) {
					type_ = v.TitleType
				} else {
					type_ = ""
				}
				titles[i] = Title{
					Title:    v.Title,
					Type:     type_,
					Language: v.Lang,
				}
			}
			b, err := json.Marshal(titles)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var container = func() types.JsonRaw {
		c := Container{
			Identifier:     content.Attributes.Container.Identifier,
			IdentifierType: content.Attributes.Container.IdentifierType,
			Type:           content.Attributes.Container.Type,
			Title:          content.Attributes.Container.Title,
			Volume:         content.Attributes.Container.Volume,
			Issue:          content.Attributes.Container.Issue,
			FirstPage:      content.Attributes.Container.FirstPage,
			LastPage:       content.Attributes.Container.LastPage,
		}
		b, err := json.Marshal(c)
		if err != nil {
			return types.JsonRaw("{}")
		}
		return types.JsonRaw(b)
	}
	var subjects = func() types.JsonRaw {
		if len(content.Attributes.Subjects) > 0 {
			subjects := make([]Subject, len(content.Attributes.Subjects))
			for i, v := range content.Attributes.Subjects {
				subjects[i] = Subject{
					Subject: v.Subject,
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
	var license = func() types.JsonRaw {
		if len(content.Attributes.RightsList) > 0 {
			id := UrlToSPDX(content.Attributes.RightsList[0].RightsURI)
			if id == "" {
				log.Printf("License URL %s not found in SPDX", content.Attributes.RightsList[0].RightsURI)
			}
			license := License{
				ID:  id,
				Url: content.Attributes.RightsList[0].RightsURI,
			}
			b, err := json.Marshal(license)
			if err != nil {
				return types.JsonRaw("{}")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("{}")
	}
	var sizes = func() types.JsonRaw {
		if len(content.Attributes.Sizes) > 0 {
			s := make([]string, len(content.Attributes.Sizes))
			copy(s, content.Attributes.Sizes)
			b, err := json.Marshal(s)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var formats = func() types.JsonRaw {
		if len(content.Attributes.Formats) > 0 {
			f := make([]string, len(content.Attributes.Formats))
			copy(f, content.Attributes.Formats)
			b, err := json.Marshal(f)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var references = func() types.JsonRaw {
		if len(content.Attributes.RelatedIdentifiers) > 0 {
			r := make([]Reference, 0)
			supportedRelations := []string{
				"Cites",
				"References",
			}
			for i, v := range content.Attributes.RelatedIdentifiers {
				if slices.Contains(supportedRelations, v.RelationType) {
					isDoi, _ := regexp.MatchString(`^10\.\d{4,9}/.+$`, v.RelatedIdentifier)
					var doi, unstructured string
					if isDoi {
						doi = DOIAsUrl(v.RelatedIdentifier)
					} else {
						unstructured = v.RelatedIdentifier
					}
					r = append(r, Reference{
						Key:          "ref" + strconv.Itoa(i+1),
						Doi:          doi,
						Unstructured: unstructured,
					})
				}
			}
			b, err := json.Marshal(r)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var relations = func() types.JsonRaw {
		if len(content.Attributes.RelatedIdentifiers) > 0 {
			r := make([]Relation, 0)
			supportedRelations := []string{
				"IsNewVersionOf",
				"IsPreviousVersionOf",
				"IsVersionOf",
				"HasVersion",
				"IsPartOf",
				"HasPart",
				"IsVariantFormOf",
				"IsOriginalFormOf",
				"IsIdenticalTo",
				"IsTranslationOf",
				"IsReviewedBy",
				"Reviews",
				"IsPreprintOf",
				"HasPreprint",
				"IsSupplementTo",
			}
			for _, v := range content.Attributes.RelatedIdentifiers {
				if slices.Contains(supportedRelations, v.RelationType) {
					isDoi, _ := regexp.MatchString(`^10\.\d{4,9}/.+$`, v.RelatedIdentifier)
					identifier := v.RelatedIdentifier
					if isDoi {
						identifier = DOIAsUrl(v.RelatedIdentifier)
					}
					r = append(r, Relation{
						ID:   identifier,
						Type: v.RelationType,
					})
				}
			}
			b, err := json.Marshal(r)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var fundingReferences = func() types.JsonRaw {
		if len(content.Attributes.FundingReferences) > 0 {
			f := make([]FundingReference, len(content.Attributes.FundingReferences))
			for i, v := range content.Attributes.FundingReferences {
				f[i] = FundingReference{
					FunderIdentifier:     v.FunderIdentifier,
					FunderIdentifierType: v.FunderIdentifierType,
					FunderName:           v.FunderName,
					AwardNumber:          v.AwardNumber,
					AwardURI:             v.AwardURI,
				}
			}
			b, err := json.Marshal(f)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var descriptions = func() types.JsonRaw {
		if len(content.Attributes.Descriptions) > 0 {
			d := make([]Description, len(content.Attributes.Descriptions))
			for i, v := range content.Attributes.Descriptions {
				var type_ string
				if slices.Contains([]string{"Abstract", "Summary", "Methods", "TechnicalInfo", "Other"}, v.DescriptionType) {
					type_ = v.DescriptionType
				} else {
					type_ = ""
				}
				d[i] = Description{
					Description: v.Description,
					Type:        type_,
					Language:    v.Lang,
				}
			}
			b, err := json.Marshal(d)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var geoLocations = func() types.JsonRaw {
		if len(content.Attributes.GeoLocations) > 0 {
			g := make([]GeoLocation, len(content.Attributes.GeoLocations))
			for i, v := range content.Attributes.GeoLocations {
				g[i] = GeoLocation{
					GeoLocationPoint: GeoLocationPoint{
						PointLongitude: v.GeoLocationPoint.PointLongitude,
						PointLatitude:  v.GeoLocationPoint.PointLatitude,
					},
					GeoLocationPlace: v.GeoLocationPlace,
					GeoLocationBox: GeoLocationBox{
						EastBoundLongitude: v.GeoLocationBox.EastBoundLongitude,
						WestBoundLongitude: v.GeoLocationBox.WestBoundLongitude,
						SouthBoundLatitude: v.GeoLocationBox.SouthBoundLatitude,
						NorthBoundLatitude: v.GeoLocationBox.NorthBoundLatitude,
					},
				}
			}
			b, err := json.Marshal(g)
			if err != nil {
				return types.JsonRaw("[]")
			}
			return types.JsonRaw(b)
		}
		return types.JsonRaw("[]")
	}
	var alternateIdentifiers = func() types.JsonRaw {
		if len(content.Attributes.AlternateIdentifiers) > 0 {
			a := make([]AlternateIdentifier, len(content.Attributes.AlternateIdentifiers))
			for i, v := range content.Attributes.AlternateIdentifiers {
				a[i] = AlternateIdentifier{
					Identifier:     v.Identifier,
					IdentifierType: v.IdentifierType,
				}
			}
			b, err := json.Marshal(a)
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
		AdditionalType:       additionalType,
		Url:                  Url,
		Contributors:         contributors(),
		Publisher:            publisher(),
		Date:                 date(),
		Titles:               titles(),
		Container:            container(),
		Subjects:             subjects(),
		Sizes:                sizes(),
		Formats:              formats(),
		Language:             content.Attributes.Language,
		License:              license(),
		Version:              content.Attributes.Version,
		References:           references(),
		Relations:            relations(),
		FundingReferences:    fundingReferences(),
		Descriptions:         descriptions(),
		GeoLocations:         geoLocations(),
		Provider:             provider,
		AlternateIdentifiers: alternateIdentifiers(),
		Files:                types.JsonRaw(nil),
		ArchiveLocations:     types.JsonRaw(nil),
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

// extract DOI prefix from URL
func PrefixFromUrl(str string) (string, error) {
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
	path := strings.Split(u.Path, "/")
	return path[1], nil
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
func dedupeSlice[T comparable](sliceList []T) []T {
	dedupeMap := make(map[T]struct{})
	list := []T{}

	for _, slice := range sliceList {
		if _, exists := dedupeMap[slice]; !exists {
			dedupeMap[slice] = struct{}{}
			list = append(list, slice)
		}
	}

	return list
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
		"https://creativecommons.org/licenses/by/3.0/legalcode":       "CC-BY-3.0",
		"https://creativecommons.org/licenses/by/4.0/legalcode":       "CC-BY-4.0",
		"https://creativecommons.org/licenses/by-nc/3.0/legalcode":    "CC-BY-NC-3.0",
		"https://creativecommons.org/licenses/by-nc/4.0/legalcode":    "CC-BY-NC-4.0",
		"https://creativecommons.org/licenses/by-nc-nd/3.0/legalcode": "CC-BY-NC-ND-3.0",
		"https://creativecommons.org/licenses/by-nc-nd/4.0/legalcode": "CC-BY-NC-ND-4.0",
		"https://creativecommons.org/licenses/by-nc-sa/3.0/legalcode": "CC-BY-NC-SA-3.0",
		"https://creativecommons.org/licenses/by-nc-sa/4.0/legalcode": "CC-BY-NC-SA-4.0",
		"https://creativecommons.org/licenses/by-nd/3.0/legalcode":    "CC-BY-ND-3.0",
		"https://creativecommons.org/licenses/by-nd/4.0/legalcode":    "CC-BY-ND-4.0",
		"https://creativecommons.org/licenses/by-sa/3.0/legalcode":    "CC-BY-SA-3.0",
		"https://creativecommons.org/licenses/by-sa/4.0/legalcode":    "CC-BY-SA-4.0",
		"https://creativecommons.org/publicdomain/zero/1.0/legalcode": "CC0-1.0",
		"https://creativecommons.org/licenses/publicdomain/":          "CC0-1.0",
		"https://opensource.org/licenses/MIT":                         "MIT",
		"https://opensource.org/licenses/Apache-2.0":                  "Apache-2.0",
		"https://opensource.org/licenses/GPL-3.0":                     "GPL-3.0",
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

package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	errs "github.com/Quavke/AnimeParsersGo/errors"
	"github.com/Quavke/AnimeParsersGo/models"
	t "github.com/Quavke/AnimeParsersGo/tools"
)

var genres_list = []string{"1-Action", "2-Adventure", "3-Racing", "4-Comedy", "5-Avant-Garde", "6-Mythology", "7-Mystery", "8-Drama", "9-Ecchi", "10-Fantasy", "11-Strategy-Game", "13-Historical", "14-Horror", "15-Kids", "17-Martial-Arts", "18-Mecha", "19-Music", "20-Parody", "21-Samurai", "22-Romance", "23-School", "24-Sci-Fi", "25-Shoujo", "27-Shounen", "29-Space", "30-Sports", "31-Super-Power", "32-Vampire", "35-Harem", "36-Slice-of-Life", "37-Supernatural", "38-Military", "39-Detective", "40-Psychological", "42-Seinen", "43-Josei", "102-Team-Sports", "103-Video-Game", "104-Adult-Cast", "105-Gore", "106-Reincarnation", "107-Love-Polygon", "108-Visual-Arts", "111-Time-Travel", "112-Gag-Humor", "114-Award-Winning", "117-Suspense", "118-Combat-Sports", "119-CGDCT", "124-Mahou-Shoujo", "125-Reverse-Harem", "130-Isekai", "131-Delinquents", "134-Childcare", "135-Magical-Sex-Shift", "136-Showbiz", "137-Otaku-Culture", "138-Organized-Crime", "139-Workplace", "140-Iyashikei", "141-Survival", "142-Performing-Arts", "143-Anthropomorphic", "144-Crossdressing", "145-Idols-(Female)", "146-High-Stakes-Game", "147-Medical", "148-Pets", "149-Educational", "150-Idols-(Male)", "151-Romantic-Subtext", "543-Gourmet"}

type ShikimoriParser struct {
	dmn     string
	context context.Context
}

func NewShikimoriParser(mirror string) *ShikimoriParser {
	var dmn string
	if mirror != "" {
		dmn = mirror
	} else {
		dmn = "shikimori.one"
	}
	return &ShikimoriParser{
		dmn:     dmn,
		context: context.Background(),
	}
}

type SHSearchResult struct {
	Genres        []string `json:"genres"`
	Link          string   `json:"link"`
	OriginalTitle string   `json:"original_title"`
	Poster        string   `json:"poster"`
	ShikimoriID   string   `json:"shikimori_parser"`
	Status        string   `json:"status"`
	Studio        string   `json:"studio"`
	Title         string   `json:"title"`
	Type          string   `json:"type"`
	Year          string   `json:"year"`
}

type SHJsonResponse struct {
	Content string `json:"content"`
}

func (jr *SHJsonResponse) Decode(r io.Reader) error {
	if err := json.NewDecoder(r).Decode(&jr); err != nil {
		return err
	}
	return nil
}

func (sh *ShikimoriParser) Search(title string) ([]*SHSearchResult, error) {
	headers := models.Headers{
		"User-Agent":       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0",
		"Accept":           "application/json, text/plain, */*",
		"X-Requested-With": "XMLHttpRequest",
	}

	params := models.Params{
		"search": title,
	}

	URL := fmt.Sprintf("https://%s/animes/autocomplete/v2", sh.dmn)

	response, err := t.RequestWithContext(sh.context, "GET", URL, params, headers, true, &SHJsonResponse{})
	if err != nil {
		error_message := fmt.Sprintf("Shikimori parser error : Search : RequestWithContext вернул ошибку: %v", err)
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}

	json_response, ok := response.Json.(*SHJsonResponse)
	if !ok {
		error_message := "Shikimori parser error : Search : не смог привести result.Json к *models.JsonResponse"
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}

	content := json_response.Content

	htmlContent := html.UnescapeString(content)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		error_message := fmt.Sprintf("Shikimori parser error : Search : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}

	res := make([]*SHSearchResult, 0)

	doc.Find("div.b-db_entry-variant-list_item").Each(func(i int, s *goquery.Selection) {
		c_data := &SHSearchResult{}
		data_type, exists := s.Attr("data-type")
		if !exists || data_type == "" {
			error_message := "Shikimori parser error : Search : goquery не смог найти атрибут data-type в контейнере с классом b-db_entry-variant-list_item"
			log.Println(error_message)
			return
		}
		if data_type != "anime" {
			return
		}

		link, exists := s.Attr("data-url")
		if !exists || link == "" {
			error_message := "Shikimori parser error : Search : goquery не смог найти атрибут data-url в контейнере с классом b-db_entry-variant-list_item"
			log.Println(error_message)
			return
		}
		c_data.Link = link

		sh_id, exists := s.Attr("data-id")
		if !exists || sh_id == "" {
			error_message := "Shikimori parser error : Search : goquery не смог найти атрибут data-id в контейнере с классом b-db_entry-variant-list_item"
			log.Println(error_message)
			return
		}
		c_data.ShikimoriID = sh_id

		image := s.Find("div.image").First()
		if image.Length() != 0 {
			poster, exists := image.Find("picture").First().Find("img").First().Attr("srcset")
			if !exists || poster == "" {
				error_message := "Shikimori parser error : Search : goquery не смог найти атрибут srcset в контейнере с классом b-db_entry-variant-list_item в div.image"
				log.Println(error_message)
				return
			}
			c_data.Poster = strings.Replace(poster, " 2x", "", 1)
		}

		info := s.Find("div.info").First()
		original_title, exists := info.Find("div.name").First().Find("a").First().Attr("title")
		if !exists || original_title == "" {
			error_message := "Shikimori parser error : Search : goquery не смог найти атрибут title в контейнере с классом b-db_entry-variant-list_item в div.info"
			log.Println(error_message)
			return
		}
		c_data.OriginalTitle = original_title

		title := strings.Split(info.Find("div.name").First().Find("a").First().Text(), "/")[0]
		c_data.Title = title

		if info.Find("div.line").First().Find("div.key").First().Text() == "Тип:" {
			value := info.Find("div.line").First().Find("div.value").First()

			b_tag := value.Find("div.b-tag")

			type_ := b_tag.Text()
			if type_ == "" {
				error_message := fmt.Sprintf("Shikimori parser error : Search : goquery не смог текст в контейнере с классом b-db_entry-variant-list_item в div.info в div.line:div.value:div.b-tag. Ошибка: %v", err)
				log.Println(error_message)
				return
			}
			c_data.Type = type_

			div_status_tag := value.Find("div.b-anime_status_tag")

			status, exists := div_status_tag.Last().Attr("data-text")
			if !exists || status == "" {
				error_message := fmt.Sprintf("Shikimori parser error : Search : goquery не смог найти атрибут data-text в контейнере с классом b-db_entry-variant-list_item в div.info в div.line:div.value: в последнем div.b-anime_status_tag. Ошибка: %v", err)
				log.Println(error_message)
				return
			}
			c_data.Status = status

			if div_status_tag.Length() > 1 {
				studio, exists := div_status_tag.First().Attr("data-text")
				if !exists || studio == "" {
					error_message := fmt.Sprintf("Shikimori parser error : Search : goquery не смог найти атрибут data-text в контейнере с классом b-db_entry-variant-list_item в div.info в div.line:div.value: в первом div.b-anime_status_tag. Ошибка: %v", err)
					log.Println(error_message)
					return
				}
				c_data.Studio = studio
			}
			if b_tag.Length() > 1 {
				c_data.Year = strings.Replace(b_tag.Last().Text(), " год", "", 1)
			}
		}

		c_data.Genres = make([]string, 0)

		for _, genre := range info.Find("span.genre-ru").EachIter() {
			c_data.Genres = append(c_data.Genres, genre.Text())
		}
		res = append(res, c_data)
	})

	return res, nil
}

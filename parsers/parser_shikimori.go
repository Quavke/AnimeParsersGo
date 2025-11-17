package parsers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"unicode/utf8"

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

type SHAnimeInfoResult struct {
	Dates           string   `json:"dates"`
	Description     string   `json:"description"`
	EpisodeDuration string   `json:"episode_duration"`
	Episodes        string   `json:"episodes"`
	Genres          []string `json:"genres"`
	Licensed        string   `json:"licensed"`
	LicensedInRU    string   `json:"licensed_in_ru"`
	NextEpisode     string   `json:"next_episode"`
	OriginalTitle   string   `json:"original_title"`
	Picture         string   `json:"picture"`
	PremiereInRU    string   `json:"premiere_in_ru"`
	Rating          string   `json:"rating"`
	Score           string   `json:"score"`
	Status          string   `json:"status"`
	Studio          string   `json:"studio"`
	Themes          []string `json:"themes"`
	Title           string   `json:"title"`
	Type            string   `json:"type"`
}

type SHRelated struct {
	Date     string `json:"date"`
	Name     string `json:"name"`
	Picture  string `json:"picture"`
	Relation string `json:"relation"`
	Type     string `json:"type"`
	Url      string `json:"url"`
}

type SHStaff struct {
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
	Link  string   `json:"link"`
}

type SHMainCharacters struct {
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type SHVideos struct {
	Name string `json:"name"`
	Link string `json:"link"`
}

type SHSimilar struct {
	Name    string `json:"name"`
	Picture string `json:"picture"`
	Link    string `json:"link"`
}

type SHAdditionalAnimeInfo struct {
	Related        []*SHRelated        `json:"related"`
	Staff          []*SHStaff          `json:"staff"`
	MainCharacters []*SHMainCharacters `json:"main_characters"`
	Screenshots    []string            `json:"screenshots"`
	Videos         []*SHVideos         `json:"videos"`
	Similar        []*SHSimilar        `json:"similar"`
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

// Быстрый поиск аниме по названию (ограничено по количеству результатов).
//
// :title: название аниме
//
// Возвращает список ссылок на SHSearchResult
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

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
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
			log.Println("Shikimori parser error : Search : goquery не смог найти атрибут data-type в контейнере с классом b-db_entry-variant-list_item")
			return
		}
		if data_type != "anime" {
			return
		}

		link, exists := s.Attr("data-url")
		if !exists || link == "" {
			log.Println("Shikimori parser error : Search : goquery не смог найти атрибут data-url в контейнере с классом b-db_entry-variant-list_item")
			return
		}
		c_data.Link = link

		sh_id, exists := s.Attr("data-id")
		if !exists || sh_id == "" {
			log.Println("Shikimori parser error : Search : goquery не смог найти атрибут data-id в контейнере с классом b-db_entry-variant-list_item")
			return
		}
		c_data.ShikimoriID = sh_id

		image := s.Find("div.image").First()
		if image.Length() != 0 {
			poster, exists := image.Find("picture").First().Find("img").First().Attr("srcset")
			if !exists || poster == "" {
				log.Println("Shikimori parser error : Search : goquery не смог найти атрибут srcset в контейнере с классом b-db_entry-variant-list_item в div.image")
				return
			}
			c_data.Poster = strings.Replace(poster, " 2x", "", 1)
		}

		info := s.Find("div.info").First()
		original_title, exists := info.Find("div.name").First().Find("a").First().Attr("title")
		if !exists || original_title == "" {
			log.Println("Shikimori parser error : Search : goquery не смог найти атрибут title в контейнере с классом b-db_entry-variant-list_item в div.info")
			return
		}
		c_data.OriginalTitle = original_title

		title := strings.Split(info.Find("div.name").First().Find("a").First().Text(), "/")[0]
		c_data.Title = title

		if info.Find("div.line").First().Find("div.key").First().Text() == "Тип:" {
			value := info.Find("div.line").First().Find("div.value").First()

			b_tag := value.Find("div.b-tag")

			type_ := b_tag.First().Text()
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

// Получение данных по аниме парсингом.
//
// :shikimori_link: ссылка на страницу шикимори с информацией (прим: https://shikimori.one/animes/z20-naruto)
//
// Возвращает ссылку на SHAnimeInfoResult:
func (sh *ShikimoriParser) AnimeInfo(shikimori_link string) (*SHAnimeInfoResult, error) {
	result := &SHAnimeInfoResult{
		Genres: make([]string, 0),
		Themes: make([]string, 0),
	}

	headers := models.Headers{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0",
	}

	resp, err := t.RequestWithContext(sh.context, "GET", shikimori_link, nil, headers, false, nil)
	if err != nil {
		error_message := fmt.Sprintf("Shikimori parser error : AnimeInfo : RequestWithContext вернул ошибку: %v", err)
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Data))
	if err != nil {
		error_message := fmt.Sprintf("Shikimori parser error : AnimeInfo : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}
	title := strings.Split(doc.Find("header.head").First().Find("h1").First().Text(), " / ")
	result.Title = title[0]
	result.OriginalTitle = title[1]

	picture := doc.Find("picture").First()
	if picture.Length() > 0 {
		srcset, exists := picture.Find("img").First().Attr("srcset")
		if !exists || srcset == "" {
			error_message := "Shikimori parser error : AnimeInfo : в picture:img не было найдено атрибута srcset"
			log.Println(error_message)
			return nil, errs.NewServiceError(error_message)
		}
		result.Picture = strings.Replace(srcset, " 2x", "", 1)
	}

	info := doc.Find("div.c-info-left").First().Find("div.block").First()
	info.Find("div.line").Each(func(i int, s *goquery.Selection) {
		key := s.Find("div.key").First().Text()
		value := s.Find("div.value").First()
		value_text := value.Text()
		value_span := value.Find("span").First()

		switch key {
		case "Тип:":
			result.Type = value_text
		case "Эпизоды:":
			result.Episodes = value_text
		case "Следующий эпизод:":
			next_episode, exists := value_span.Attr("data-datetime")
			if !exists || next_episode == "" {
				log.Println("Shikimori parser error : AnimeInfo : goquery не смог найти атрибут data-datetime в div.c-info-left:div.line:span")
				return
			}
			result.NextEpisode = next_episode
		case "Длительность эпизода:":
			result.EpisodeDuration = value_text
		case "Статус:":
			status, exists := value_span.Attr("data-text")
			if !exists || status == "" {
				log.Println("Shikimori parser error : AnimeInfo : goquery не смог найти атрибут data-text в div.c-info-left:div.line:span")
				return
			}
			result.Status = status

			value_all_spans := value.Find("span")

			if value_all_spans.Length() > 1 {
				result.Dates = value_all_spans.Last().Text()
			} else {
				result.Dates = strings.TrimSpace(value_text)
			}
		case "Жанры:":
			for _, genre := range value.Find("span.genre-ru").EachIter() {
				result.Genres = append(result.Genres, genre.Text())
			}
		case "Темы:", "Тема:":
			for _, theme := range value.Find("span.genre-ru").EachIter() {
				result.Themes = append(result.Themes, theme.Text())
			}
		case "Рейтинг:":
			result.Rating = value_text
		case "Лицензировано:":
			result.Licensed = value_text
		case "Лицензировано в РФ под названием:":
			result.LicensedInRU = value_text
		case "Премьера в РФ":
			result.PremiereInRU = value_text
		}
	})

	return result, nil
}

//FIXME
// Парсится только первое видео вместо всех четырех

// Получение дополнительных данных об аниме.
// Получаемые данные: связанные аниме (продолжение, предыстория, альтернативное и т.п.), Авторы (автор манги, режиссер), Главные герои, Скриншоты, Ролики, Похожее
//
// :shikimori_link: ссылка на страницу шикимори с информацией (прим: https://shikimori.one/animes/z20-naruto)
//
// Возвращает ссылку на SHAdditionalAnimeInfo
func (sh *ShikimoriParser) AdditionalAnimeInfo(shikimori_link string) (*SHAdditionalAnimeInfo, error) {
	var link string
	r, _ := utf8.DecodeLastRuneInString(shikimori_link)
	if r == '/' {
		link = shikimori_link + "resources"
	} else {
		link = shikimori_link + "/resources"
	}

	headers := models.Headers{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0",
	}

	resp, err := t.RequestWithContext(sh.context, "GET", link, nil, headers, false, nil)
	if err != nil {
		error_message := fmt.Sprintf("Shikimori parser error : AdditionalAnimeInfo : RequestWithContext вернул ошибку: %v", err)
		log.Println(error_message)
		return nil, errs.NewServiceError(error_message)
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(resp.Data))

	res := &SHAdditionalAnimeInfo{
		Related:        make([]*SHRelated, 0),
		Staff:          make([]*SHStaff, 0),
		MainCharacters: make([]*SHMainCharacters, 0),
		Screenshots:    make([]string, 0),
		Videos:         make([]*SHVideos, 0),
		Similar:        make([]*SHSimilar, 0),
	}

	r1 := doc.Find("div.cc-related-authors").First()
	r1.Find("div.c-column").Each(func(i int, s *goquery.Selection) {
		col_type := s.Find("div.subheadline").First().Text()
		switch col_type {
		case "Связанное":
			for _, entry := range s.Find("div.b-db_entry-variant-list_item").EachIter() {
				c_data := &SHRelated{}
				url, exists := entry.Attr("data-url")
				if !exists || url == "" {
					log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут data-url в div.cc-related-authors:div.c-column:div.subheadline:div.b-db_entry-variant-list_item")
					continue
				}
				c_data.Url = url

				if entry.Find("picture").First().Length() > 0 {
					picture, exists := entry.Find("picture").First().Find("img").First().Attr("srcset")
					if !exists || picture == "" {
						log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут srcset в div.cc-related-authors:div.c-column:div.subheadline:div.b-db_entry-variant-list_item:picture:img")
						continue
					}
					c_data.Picture = strings.Replace(picture, " 2x", "", 1)
				}
				div_name := entry.Find("div.name").First()
				if div_name.Length() == 0 {
					continue
				}
				name_ru := div_name.Find("span.name-ru").First()
				name_en := div_name.Find("span.name-en").First()
				if name_ru.Length() > 0 {
					c_data.Name = name_ru.Text()
				} else if name_en.Length() > 0 {
					c_data.Name = name_en.Text()
				}

				for _, other := range entry.Find("div.line").First().Find("div").EachIter() {
					other_text := other.Text()
					cls, exists := other.Attr("class")
					if !exists || cls == "" {
						log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут class в div.cc-related-authors:div.c-column:div.subheadline:div.b-db_entry-variant-list_item:div.line:div")
						continue
					}
					if strings.Contains(cls, "b-anime_status_tag") {
						c_data.Relation = other_text
					} else if strings.Contains(cls, "linkeable") {
						link, exists := other.Attr("data-href")
						if !exists || cls == "" {
							log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут data-href в div.cc-related-authors:div.c-column:div.subheadline:div.b-db_entry-variant-list_item:div.line:div")
							continue
						}
						if strings.Contains(link, "/kind/") {
							c_data.Type = other_text
						} else if strings.Contains(link, "/season/") {
							c_data.Date = other_text
						}
					}
				}
				if c_data.Type == "Клип" && c_data.Name == "" {
					c_data.Name = entry.Find("div.name").First().Find("a").First().Text()
				}

				res.Related = append(res.Related, c_data)
			}
		case "Авторы":
			for _, entry := range s.Find("div.b-db_entry-variant-list_item").EachIter() {
				c_data := &SHStaff{
					Roles: make([]string, 0),
				}
				link, exists := entry.Attr("data-url")
				if !exists || link == "" {
					log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут data-url в div.cc-related-authors:div.c-column:div.subheadline:div.b-db_entry-variant-list_item")
					continue
				}
				c_data.Link = link

				name, exists := entry.Attr("data-text")
				if !exists || name == "" {
					log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут data-text в div.cc-related-authors:div.c-column:div.subheadline:div.b-db_entry-variant-list_item")
					continue
				}
				c_data.Name = name

				for _, role := range entry.Find("div.line").First().Find("div.b-tag").EachIter() {
					c_data.Roles = append(c_data.Roles, role.Text())
				}
				res.Staff = append(res.Staff, c_data)
			}
		}

	})

	r1 = doc.Find("div.c-characters").First()
	if r1.Length() > 0 {
		r1.Find("article").Each(func(i int, s *goquery.Selection) {
			c_data := &SHMainCharacters{}
			meta := s.Find("meta[itemprop=\"image\"]").First()
			if meta.Length() > 0 {
				picture, exists := meta.Attr("content")
				if !exists || picture == "" {
					log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут content в div.c-characters:article:meta itemprop = \"image\"")
					return
				}
				c_data.Picture = picture
			}
			if tmp := s.Find("span.name-ru").First(); tmp.Length() > 0 {
				c_data.Name = tmp.Text()
			}
			res.MainCharacters = append(res.MainCharacters, c_data)
		})
	}

	r1 = doc.Find("div.two-videos").First()
	if r1.Length() > 0 {
		if r1.Find("div.c-screenshots").First().Length() > 0 {
			r1.Find("a.c-screenshot").Each(func(i int, s *goquery.Selection) {
				href, exists := s.Attr("href")
				if !exists || href == "" {
					log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут href в div.two-videos:div.c-screenshots:a.c-screenshot")
					return
				}
				res.Screenshots = append(res.Screenshots, href)
			})
		}
		if r1.Find("div.c-videos").First().Length() > 0 {
			r1.Find("div.c-video").Each(func(i int, s *goquery.Selection) {
				c_data := &SHVideos{}
				link, exists := s.Find("a").First().Attr("href")
				if !exists || link == "" {
					log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут href в div.two-videos:div.c-videos:div.c-video")
					return
				}
				c_data.Link = link
				if tmp := s.Find("span.name").First(); tmp.Length() > 0 {
					c_data.Name = tmp.Text()
				}
				res.Videos = append(res.Videos, c_data)
			})
		}

		r1 = doc.Find("div.block")
		if r1.Length() > 0 {
			r1.Find("article").Each(func(i int, s *goquery.Selection) {
				c_data := &SHSimilar{}
				img := s.Find("meta[itemprop=\"image\"]").First()
				if img.Length() > 0 {
					content, exists := img.Attr("content")
					if !exists || content == "" {
						log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут content в div.block:article:meta itemprop = \"image\"")
						return
					}
					c_data.Picture = content
				}
				c_data.Name = s.Find("span.name-ru").First().Text()
				link, exists := s.Find("div").First().Attr("data-href")
				if !exists || link == "" {
					log.Println("Shikimori parser error : AdditionalAnimeInfo : goquery не смог найти атрибут data-href в div.block:article:div")
					return
				}
				c_data.Link = link
				res.Similar = append(res.Similar, c_data)
			})
		}
	}

	return res, nil
}

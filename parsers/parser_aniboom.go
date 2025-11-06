package parsers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	parsers_errors "github.com/Quavke/AnimeParsersGo/errors"
)

type Client struct {
	httpClient *http.Client
}

type AniboomParser struct {
	dmn     string
	Client  *Client
	context context.Context
}

func NewAniboomParser(mirror string) *AniboomParser {
	var dmn string
	if mirror != "" {
		dmn = mirror
	} else {
		dmn = "animego.me"
	}
	client := &Client{
		&http.Client{Timeout: 10 * time.Second},
	}
	return &AniboomParser{
		dmn:    dmn,
		Client: client,
	}
}

type FastSearchResult struct {
	Title      string `json:"title"`
	Year       string `json:"year"`
	OtherTitle string `json:"other_title"`
	Type       string `json:"type"`
	Link       string `json:"link"`
	AnimegoID  string `json:"animego_id"`
}

type EpisodeInfo struct {
	Num    string `json:"num"`
	Title  string `json:"title"`
	Date   string `json:"date"`
	Status string `json:"status"`
}

type Translation struct {
	Name           string `json:"name"`
	TranslationsID string `json:"translation_id"`
}

type OtherAnimeInfo struct {
	AgeRests       string   `json:"age_restrictions"`
	ReleaseDate    string   `json:"release_date"`
	MainCharacters []string `json:"main_characters"`
	Duration       string   `json:"duration"`
	OriginalSource string   `json:"original_source"`
	MPAARating     string   `json:"mpaa_rating"`
	Season         string   `json:"season"`
	OriginalRanobe string   `json:"ranobe"`
	OriginalManga  string   `json:"manga"`
	Studio         string   `json:"studio"`
}
type SearchResult struct {
	Title        string            `json:"title"`
	OtherTitle   string            `json:"other_title"`
	Status       string            `json:"status"`
	Type         string            `json:"type"`
	Genres       []string          `json:"genres"`
	Description  string            `json:"description"`
	Episodes     string            `json:"episodes"`
	EpisodesInfo []EpisodeInfo     `json:"episodes_info"`
	Translations []Translation     `json:"translations"`
	PosterURL    string            `json:"poster_url"`
	Trailer      string            `json:"trailer"`
	Screenshots  []string          `json:"screenshots"`
	OtherInfo    *OtherAnimeInfo   `json:"other_info"`
	Link         string            `json:"link"`
	AnimegoID    string            `json:"animego_id"`
	Unparsed     map[string]string `json:"unparsed"`
}

/*
Быстрый поиск через animego.me

:title: Название аниме

Возвращает массив из ссылок на модель FastSearchResult
*/
func (ab *AniboomParser) FastSearch(title string) ([]*FastSearchResult, error) {
	params := url.Values{}

	params.Set("type", "small")
	params.Set("q", title)
	domain := fmt.Sprintf("https://%s/", ab.dmn)
	url := fmt.Sprintf("%ssearch/all?%s", domain, params.Encode())
	request, err := http.NewRequestWithContext(ab.context, "get", url, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")
	request.Header.Set("Referer", domain)

	resp, err := ab.Client.httpClient.Do(request)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Aniboom parser error : FastSearch : http клиент не смог выполнить запрос, код %d. Ошибка: %v", resp.StatusCode, err)
		return nil, parsers_errors.ServiceError
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Aniboom parser error : FastSearch : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		return nil, parsers_errors.ServiceError
	}
	res := make([]*FastSearchResult, 0)
	items := doc.Find("div.result-search-anime").First().Find("div.result-search-item")

	if items.Length() == 0 {
		log.Printf("Aniboom parser error : FastSearch : в контейнере result-search-anime не найдено ни одного элемента div.result-search-item")
		return nil, parsers_errors.NoResults
	}

	items.Each(func(i int, s *goquery.Selection) {
		c_data := &FastSearchResult{}
		c_data.Title = strings.TrimSpace(s.Find("h5").First().Text())
		c_data.Year = strings.TrimSpace(s.Find("span.anime-year").First().Text())
		c_data.OtherTitle = s.Find("div.text-truncate").First().Text()
		c_data.Type = s.Find("a[href*=\"anime/type\"]").First().Text()
		link := s.Find("h5 a").First()
		var rawLink string
		if link.Length() > 0 {
			href, exists := link.Attr("href")
			if exists {
				rawLink = href
			}
		}
		c_data.Link = fmt.Sprintf("https:%s", ab.dmn) + rawLink
		fullLink := c_data.Link
		lastDashIndex := strings.LastIndex(fullLink, "-")
		if lastDashIndex != -1 && lastDashIndex < len(fullLink)-1 {
			c_data.AnimegoID = fullLink[lastDashIndex+1:]
		} else {
			c_data.AnimegoID = ""
		}
		res = append(res, c_data)
	})

	return res, nil
}

/*
Расширенный поиск через animego.me. Собирает дополнительные данные об аниме.

:title: Название

Возвращает массив из ссылок на SearchResult
*/
func (ab *AniboomParser) Search(title string) ([]*SearchResult, error) {
	elements, err := ab.FastSearch(title)
	if err != nil {
		log.Printf("Aniboom parser error : search : FastSearch не смог найти данные для title %s. Ошибка: %v", title, err)
		return nil, err
	}
	res := make([]*SearchResult, 0)
	for _, anime := range elements {
		c_data, err := ab.AnimeInfo(anime.Link)
		if err != nil {
			log.Printf("Aniboom parser error : search : AnimeInfo не смог найти данные для %s по ссылке %s. Ошибка: %v", anime.Title, anime.Link, err)
			continue
		}
		res = append(res, c_data)
	}

	return res, nil
}

/*
Получение данных об аниме с его страницы на animego.me.

:link: Ссылка на страницу (прим: https:animego.me/anime/volchica-i-pryanosti-torgovec-vstrechaet-mudruyu-volchicu-2546)

Возвращает модель SearchResult
*/
func (ab *AniboomParser) AnimeInfo(link string) (*SearchResult, error) {
	var c_data SearchResult

	request, err := http.NewRequestWithContext(ab.context, "get", link, nil)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s/search/all?q=anime", ab.dmn)

	request.Header.Set("Referer", url)

	resp, err := ab.Client.httpClient.Do(request)
	if err != nil {
		log.Printf("Aniboom parser error : AnimeInfo : http клиент не смог выполнить запрос, код: %d. Ошибка: %v", resp.StatusCode, err)
		return nil, parsers_errors.ServiceError
	} else if resp.StatusCode == http.StatusTooManyRequests {
		log.Println("Aniboom parser error : AnimeInfo : Сервер вернул код ошибки 429. Слишком частые запросы")
		return nil, parsers_errors.TooManyRequests
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("Aniboom parser error : AnimeInfo : Сервер не вернул ожидаемый код 200. Код: %d\n", resp.StatusCode)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Aniboom parser error : AnimeInfo : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		return nil, parsers_errors.ServiceError
	}
	c_data.Link = link
	fullLink := c_data.Link
	lastDashIndex := strings.LastIndex(fullLink, "-")
	if lastDashIndex != -1 && lastDashIndex < len(fullLink)-1 {
		c_data.AnimegoID = fullLink[lastDashIndex+1:]
	} else {
		c_data.AnimegoID = ""
	}
	dmn := fmt.Sprintf("https://%s", ab.dmn)
	c_data.Title = strings.TrimSpace(doc.Find("div.anime-title h1").Text())
	poster_url_doc := doc.Find("img").First()
	if poster_url_doc.Length() > 0 {
		src, exists := poster_url_doc.Attr("src")
		if exists {
			c_data.PosterURL = src

			SlashIndex := strings.Index(c_data.PosterURL, "/upload")
			if SlashIndex != -1 && SlashIndex < len(c_data.PosterURL)-1 {
				c_data.PosterURL = c_data.PosterURL[SlashIndex+1:]
			} else {
				c_data.PosterURL = ""
			}
			if len(c_data.PosterURL) > 0 {
				c_data.PosterURL = dmn + c_data.PosterURL
			}
		}
	}
	anime_info := doc.Find("div.anime-info dl").First()
	if anime_info.Length() == 0 {
		log.Printf("Aniboom parser error : AnimeInfo : doc.Find(\"div.anime-info dl\") не смог найти тег dl")
		return nil, parsers_errors.NoResults
	}
	var allDTs []*goquery.Selection
	var allDDs []*goquery.Selection

	anime_info.Find("dt").Each(func(i int, s *goquery.Selection) {
		allDTs = append(allDTs, s)
	})

	anime_info.Find("dd").Each(func(i int, s *goquery.Selection) {
		skip := false

		if s.HasClass("mt-2") && s.HasClass("col-12") {
			skip = true
		}

		if s.Find("hr").First().Length() > 0 {
			skip = true
		}

		if !skip {
			allDDs = append(allDDs, s)
		}
	})
	other_anime_info := make(map[string]string, 0)
	var OtherAniInfo OtherAnimeInfo
	minLen := len(allDTs)
	if minLen > len(allDDs) {
		minLen = len(allDDs)
	}

	for i := 0; i < minLen; i++ {
		key := strings.TrimSpace(allDTs[i].Text())
		value := strings.TrimSpace(allDDs[i].Text())

		if key != "" && value != "" {
			other_anime_info[key] = value
		}
	}

	for i := 0; i < len(allDTs); i++ {
		key := strings.TrimSpace(allDTs[i].Text())
		value := strings.TrimSpace(allDDs[i].Text())
		if key == "Озвучка" {
			continue
		} else if key == "Жанр" {
			genres := make([]string, 0)
			allDDs[i].Find("a").Each(func(i int, s *goquery.Selection) {
				genres = append(genres, strings.TrimSpace(s.Text()))
			})
			c_data.Genres = genres
		} else if key == "Главные герои" {
			main_characters := make([]string, 0)
			allDDs[i].Find("a").Each(func(i int, s *goquery.Selection) {
				main_characters = append(main_characters, strings.TrimSpace(s.Text()))
			})
			OtherAniInfo.MainCharacters = main_characters
		} else if key == "Эпизоды" {
			c_data.Episodes = value
		} else if key == "Статус" {
			c_data.Status = value
		} else if key == "Тип" {
			c_data.Type = value
		} else {
			switch key {
			case "Возрастные ограничения":
				OtherAniInfo.AgeRests = value
			case "Выпуск":
				OtherAniInfo.ReleaseDate = value
			case "Длительность":
				OtherAniInfo.Duration = value
			case "Первоисточник":
				OtherAniInfo.OriginalSource = value
			case "Рейтинг MPAA":
				OtherAniInfo.MPAARating = value
			case "Сезон":
				OtherAniInfo.Season = value
			case "Снят по ранобэ":
				OtherAniInfo.OriginalRanobe = value
			case "Снят по манге":
				OtherAniInfo.OriginalManga = value
			case "Студия":
				OtherAniInfo.Studio = value
			default:
				c_data.Unparsed[key] = value
			}
		}
	}

	c_data.Description = strings.TrimSpace(doc.Find("div.description").Text())
	screenshots_urls := make([]string, 0)
	doc.Find("a.screenshots-item").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			url := dmn + strings.TrimSpace(href)
			screenshots_urls = append(screenshots_urls, url)
		}
	})
	c_data.Screenshots = screenshots_urls

	trailer_cont := doc.Find("a.video-block")
	if trailer_cont.Length() > 0 {
		href, exists := trailer_cont.Find("a.video-item").Attr("href")
		if exists {
			c_data.Trailer = strings.TrimSpace(href)
		}
	}

	c_data.EpisodesInfo = ab.EpisodesInfo(link)

	translations_info, err := ab.GetTranslationsInfo(c_data.AnimegoID)
	if err == parsers_errors.ContentBlocked {
		log.Printf("Aniboom parser warning : AnimeInfo : GerTranslationsInfo вернул ошибку ContentBlocked")
	} else if err != nil {
		log.Printf("Aniboom parser error : AnimeInfo : GerTranslationsInfo вернул неожиданную ошибку: %v", err)
	} else {
		c_data.Translations = translations_info
	}

	c_data.OtherInfo = &OtherAniInfo
	return &c_data, nil
}

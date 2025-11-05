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
	Title        string           `json:"title"`
	OtherTitle   string           `json:"other_title"`
	Status       string           `json:"status"`
	Type         string           `json:"type"`
	Genres       []string         `json:"genres"`
	Description  string           `json:"description"`
	Episodes     string           `json:"episodes"`
	EpisodesInfo []EpisodeInfo    `json:"episodes_info"`
	Translations []Translation    `json:"translations"`
	PosterURL    string           `json:"poster_url"`
	Trailer      string           `json:"trailer"`
	Screenshots  []string         `json:"screenshots"`
	OtherInfo    []OtherAnimeInfo `json:"other_info"`
	Link         string           `json:"link"`
	AnimegoID    string           `json:"animego_id"`
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
		log.Printf("Aniboom parser : FastSearch : http клиент не смог выполнить запрос, код %d. Ошибка: %v", resp.StatusCode, err)
		return nil, parsers_errors.ServiceError
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Aniboom parser : FastSearch : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
		return nil, parsers_errors.ServiceError
	}
	res := make([]*FastSearchResult, 0)
	items := doc.Find("div.result-search-anime").First().Find("div.result-search-item")

	if items.Length() == 0 {
		log.Printf("Aniboom parser : FastSearch : в контейнере result-search-anime не найдено ни одного элемента div.result-search-item")
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
		log.Printf("Aniboom parser : search : FastSearch не смог найти данные для title %s. Ошибка: %v", title, err)
		return nil, err
	}
	res := make([]*SearchResult, 0)
	for _, anime := range elements {
		c_data, err := ab.AnimeInfo(anime.Link)
		if err != nil {
			log.Printf("Aniboom parser : search : AnimeInfo не смог найти данные для %s по ссылке %s. Ошибка: %v", anime.Title, anime.Link, err)
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
		log.Printf("Aniboom parser : AnimeInfo : http клиент не смог выполнить запрос, код: %d. Ошибка: %v", resp.StatusCode, err)
		return nil, parsers_errors.ServiceError
	} else if resp.StatusCode == http.StatusTooManyRequests {
		log.Println("Aniboom parser : AnimeInfo : Сервер вернул код ошибки 429. Слишком частые запросы")
		return nil, parsers_errors.TooManyRequests
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("Aniboom parser : AnimeInfo : Сервер не вернул ожидаемый код 200. Код: %d\n", resp.StatusCode)
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Aniboom parser : FastSearch : goquery не смог преобразовать ответ в документ. Ошибка: %v", err)
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
	c_data.Title = strings.TrimSpace(doc.Find("div.anime-title h1").Text())
	return &c_data, nil
}

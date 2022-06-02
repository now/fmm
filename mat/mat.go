package mat

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/now/x/log"
	xjson "github.com/now/x/encoding/json"
	xhttp "github.com/now/x/net/http"
)

const (
	GetCategoryTreeURL    = "https://mat.se/api/product/getCategoryTree"
	ListCategoryURLPrefix = "https://mat.se/api/product/listCategory?categoryId="
)

type Category struct {
	Count                     int         `json:"count"`
	Id                        xjson.Value `json:"id"`
	Name                      xjson.Value `json:"name"`
	MaxImmediateChildren      xjson.Value `json:"maxImmediateChildren"`
	SubCategories             []Category  `json:"subCategories"`
	BannerImage               xjson.Value `json:"bannerImage"`
	BannerActive              xjson.Value `json:"bannerActive'`
	NameFullPath              xjson.Value `json:"nameFullpath"`
	CountOwn                  int         `json:"countOwn"`
	Top5Products              []Product   `json:"top5Products"`
	SwedishProductsPercentage float64     `json:"swedishProductsPercentage"`
	Products                  []Product   `json:"-"`
}

type Product struct {
	Id                           xjson.Value `json:"id"`
	DateCreated                  xjson.Value `json:"dateCreated"`
	Name                         xjson.Value `json:"name"`
	ShowPrice                    xjson.Value `json:"showPrice"`
	FriendlyURL                  xjson.Value `json:"friendlyUrl"`
	FormattedName                xjson.Value `json:"formattedName"`
	EAN                          xjson.Value `json:"ean"`
	CountryOfOrigin              *string     `json:"countryOfOrigin"`
	TranslatedCountryName        xjson.Value `json:"translatedCountryName"`
	LabelTags                    xjson.Value `json:"labelTags"`
	HasImage                     xjson.Value `json:"hasImage"`
	ImageURL                     xjson.Value `json:"imageUrl"`
	Brand                        xjson.Value `json:"brand"`
	Categories                   xjson.Value `json:"categories"`
	DisplayBrand                 xjson.Value `json:"displayBrand"`
	FavouriteCount               xjson.Value `json:favouriteCount"`
	Active                       xjson.Value `json:"active"`
	InStockQuantity              xjson.Value `json:"inStockQuantity"`
	Orderable                    xjson.Value `json:"orderable"`
	Price                        xjson.Value `json:"price"`
	PriceWithoutVat              xjson.Value `json:"priceWithoutVat"`
	RecommendedRetailPrice       xjson.Value `json:"recommendedRetailPrice"`
	ReadableComparisonPrice      xjson.Value `json:"readableComparisonPrice"`
	Discount                     xjson.Value `json:"discount"`
	DiscountPrice                xjson.Value `json:"discountedPrice"`
	Size                         xjson.Value `json:"size"`
	WeightProduct                xjson.Value `json:"weightProduct"`
	VAT                          xjson.Value `json:"vat"`
	Content                      xjson.Value `json:"content"`
	Durability                   xjson.Value `json:"durability"`
	Recycling                    xjson.Value `json:"recycling"`
	Detail                       xjson.Value `json:"detail"`
	Storage                      xjson.Value `json:"storage"`
	CombinedNameWithBrand        xjson.Value `json:"combinedNameWithBrand"`
	AverageProductLife           xjson.Value `json:"averageProductLife"`
	MinimumDaysBeforeExpire      xjson.Value `json:"minimumDaysBeforeExpire"`
	AverageUnitWeight            xjson.Value `json:"averageUnitWeight"`
	Weight                       xjson.Value `json:"weight"`
	IsMixable                    xjson.Value `json:"isMixable"`
	IdRequired                   xjson.Value `json:"idRequired"`
	TagNames                     xjson.Value `json:"tagNames"`
	Featured                     xjson.Value `json:"featured"`
	CampaignRank                 xjson.Value `json:"campaignRank"`
	AvailableQuantities          xjson.Value `json:"availableQuantities"`
	Emission                     xjson.Value `json:"emission"`
	Allergies                    xjson.Value `json:"allergies"`
	WarehouseProductMappingId    xjson.Value `json:"warehouseProductMappingId"`
	ProductRelationship          xjson.Value `json:"productRelationship"`
	SoldCount                    int         `json:"soldCount"`
	Variety                      xjson.Value `json:"variety"`
	IsAgreeWithPersonalTerms     xjson.Value `json:"isAgreeWithPersonalTerms"`
	HasDeclinedPersonalAgreement xjson.Value `json:"hasDeclinedPersonalAgreement"`
	Probability                  xjson.Value `json:"probability"`
	ProductCountries             xjson.Value `json:"productCountries"`
	ProductVarieties             xjson.Value `json:"productVarieties"`
	ProductBanner                xjson.Value `json:"productBanner"`
	IsShowOnTopProduct           xjson.Value `json:"isShowOnTopProduct"`
	PrimaryCategoryNamePath      xjson.Value `json:"primaryCategoryNamePath"`
}

type BySoldCount []Product

func (a BySoldCount) Len() int           { return len(a) }
func (a BySoldCount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySoldCount) Less(i, j int) bool { return a[i].SoldCount > a[j].SoldCount }

func ProcessCategoryTree(ctx context.Context) (*Category, error) {
	var c Category
	log.Entry(ctx, "fetching category tree")
	if resp, err := xhttp.In(ctx).Get(GetCategoryTreeURL); err != nil {
		return nil, err
	} else if err := xjson.DecodeAndClose(resp.Body, &c); err != nil {
		return nil, fmt.Errorf("mat: can’t decode JSON returned by %s: %w", GetCategoryTreeURL, err)
	} else {
		ctx, cancel := context.WithCancel(ctx)
		var wg sync.WaitGroup
		cs := make(chan *Category)
		for i := 0; i < 6; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						wg.Done()
						return
					case c := <-cs:
						if err := ProcessCategory(ctx, c); err != nil {
							log.Entry(ctx, err.Error())
							cancel()
						}
						if ctx.Err() != nil {
							return
						}
					}
				}
			}()
		}
		var send func(c *Category)
		send = func(c *Category) {
			cs <- c
			for i := range c.SubCategories {
				send(&c.SubCategories[i])
			}
		}
		send(&c)
		close(cs)
		wg.Wait()
		return &c, nil
	}
}

func ProcessCategory(ctx context.Context, c *Category) error {
	var ps []Product
	url := fmt.Sprintf("%s%v", ListCategoryURLPrefix, c.Id)
	log.Entry(ctx, "fetching category", log.Reflect("id", c.Id))
	if resp, err := xhttp.In(ctx).Get(url); err != nil {
		return err
	} else if err := xjson.DecodeAndClose(resp.Body, &ps); err != nil {
		return fmt.Errorf("mat: can’t parse JSON returned by %s: %w", url, err)
	} else {
		subCategoriesCount := 0
		for i := range c.SubCategories {
			subCategoriesCount += c.SubCategories[i].Count
		}
		c.CountOwn = c.Count - subCategoriesCount

		sort.Sort(BySoldCount(ps))
		n := 5
		if n > len(ps) {
			n = len(ps)
		}
		c.Top5Products = ps[0:n]

		swedish := 0
		for i := range ps {
			if ps[i].CountryOfOrigin != nil && *ps[i].CountryOfOrigin == "SE" {
				swedish += 1
			}
		}
		c.SwedishProductsPercentage = math.Round(100 * (float64(swedish) / float64(len(ps))))

		return nil
	}
}

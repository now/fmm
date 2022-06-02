package mat_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/now/x/log"
	xhttp "github.com/now/x/net/http"
	"github.com/now/x/net/httptest"

	"github.com/now/future-memories/mat"
)

func TestProcessCategoryTree(t *testing.T) {
	if c, err := mat.ProcessCategoryTree(log.Nop(httptest.Using(context.Background(), func(req *http.Request) (*http.Response, error) {
		path := "testdata/categoryTree.json"
		if strings.HasPrefix(req.URL.String(), mat.ListCategoryURLPrefix) {
			path = fmt.Sprintf("testdata/%s.json", req.URL.String()[len(mat.ListCategoryURLPrefix):])
		}
		if f, err := os.Open(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
			return nil, nil
		} else {
			return xhttp.NewResponseBuilder().ContentType("application/json").Body(f).Build(http.StatusOK), nil
		}
	}))); err != nil {
		t.Errorf("mat.ProcessCategoryTree(…) = %#v, want <nil>", err)
	} else {
		type product struct {
			SoldCount int
		}
		type category struct {
			CountOwn                  int
			Top5Products              []product
			SwedishProductsPercentage float64
			SubCategories             []category
		}
		var simplifyProduct func(*mat.Product) product
		simplifyProduct = func(p *mat.Product) product {
			return product{p.SoldCount}
		}
		var simplifyCategory func(*mat.Category) category
		simplifyCategory = func(c *mat.Category) category {
			cs := make([]category, len(c.SubCategories))
			for i := range c.SubCategories {
				cs[i] = simplifyCategory(&c.SubCategories[i])
			}
			ps := make([]product, len(c.Top5Products))
			for i := range c.Top5Products {
				ps[i] = simplifyProduct(&c.Top5Products[i])
			}
			return category{c.CountOwn, ps, c.SwedishProductsPercentage, cs}
		}
		got := simplifyCategory(c)
		want := category{
			CountOwn: 0,
			Top5Products: []product{
				{SoldCount: 18005},
				{SoldCount: 9033},
				{SoldCount: 8932},
				{SoldCount: 8661},
				{SoldCount: 8302},
			},
			SwedishProductsPercentage: 17,
			SubCategories: []category{
				{
					CountOwn: 0,
					Top5Products: []product{
						{SoldCount: 18005},
						{SoldCount: 9033},
						{SoldCount: 8932},
						{SoldCount: 8661},
						{SoldCount: 8302},
					},
					SwedishProductsPercentage: 17,
					SubCategories: []category{
						{
							CountOwn: 67,
							Top5Products: []product{
								{SoldCount: 8302},
								{SoldCount: 7900},
								{SoldCount: 5229},
								{SoldCount: 3690},
								{SoldCount: 3420},
							},
							SwedishProductsPercentage: 19,
							SubCategories:             []category{},
						},
						{
							CountOwn: 3,
							Top5Products: []product{
								{SoldCount: 192},
								{SoldCount: 24},
								{SoldCount: 19},
							},
							SwedishProductsPercentage: 33,
							SubCategories:             []category{},
						},
					},
				},
			},
		}
		if diff := cmp.Diff(got, want); diff != "" {
			t.Errorf("mat.ProcessCategoryTree(…) diff -got +want\n%s", diff)
		}
	}
}

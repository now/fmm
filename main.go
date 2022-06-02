package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/now/x/log"
	xzap "github.com/now/x/log/zap"
	"go.uber.org/zap"

	"github.com/now/future-memories/mat"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("can’t initialize logger: %v", err))
	}
	defer logger.Sync()
	ctx := log.Using(context.Background(), xzap.Logger{logger})
	if c, err := mat.ProcessCategoryTree(ctx); err != nil {
		log.Entry(ctx, "can’t process categories", log.Error(err))
		panic(err)
	} else {
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		e.Encode(c)
	}
}

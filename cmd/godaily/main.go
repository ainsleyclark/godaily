package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ainsleyclark/godaily/internal/news"
	"github.com/ainsleyclark/godaily/internal/source"
)

func main() {
	ctx := context.Background()

	fetch, err := source.NewDevTo().Fetch(context.Background())
	if err != nil {
		slog.ErrorContext(ctx, err.Error())
		return
	}



	var fetchers []news.Fetcher = {

	}

	fmt.Println(fetch)
}

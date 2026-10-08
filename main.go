package main

import (
	"embed"
	"github.com/cronitorio/crontab-dashboard/internal/dashboard"
)

//go:embed web/static
var assets embed.FS

func main() {
	dashboard.SetWebAssets(assets)
	dashboard.Execute()
}

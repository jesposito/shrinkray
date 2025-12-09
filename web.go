package shrinkray

import "embed"

//go:embed web/templates/* web/debug/*
var WebFS embed.FS

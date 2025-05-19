package web

import "embed"

//go:embed static/*
//go:embed template/*
var WebFs embed.FS

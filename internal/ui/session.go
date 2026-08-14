package ui

import "github.com/pragmaticcs/cawder/internal/core/memory"

type SessionItem struct {
	Title string
	Meta  memory.SessionMeta
	IsNew bool
}
